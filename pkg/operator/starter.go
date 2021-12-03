package operator

import (
	"context"
	"fmt"
	"github.com/openshift/library-go/pkg/controller/factory"
	"k8s.io/client-go/dynamic"
	"time"

	kubeclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"
	"os"
	"strings"

	opv1 "github.com/openshift/api/operator/v1"
	configclient "github.com/openshift/client-go/config/clientset/versioned"
	configinformers "github.com/openshift/client-go/config/informers/externalversions"
	"github.com/openshift/library-go/pkg/controller/controllercmd"
	"github.com/openshift/library-go/pkg/operator/csi/csicontrollerset"
	"github.com/openshift/library-go/pkg/operator/csi/csidrivercontrollerservicecontroller"
	"github.com/openshift/library-go/pkg/operator/csi/csidrivernodeservicecontroller"
	goc "github.com/openshift/library-go/pkg/operator/genericoperatorclient"
	"github.com/openshift/library-go/pkg/operator/v1helpers"

	"github.com/IBM/ibm-vpc-block-csi-driver-operator/assets"
	"github.com/IBM/ibm-vpc-block-csi-driver-operator/pkg/controller/secret"
	"github.com/IBM/ibm-vpc-block-csi-driver-operator/pkg/util"
)

func readFileAndReplace(name string) ([]byte, error) {
	pairs := []string{
		"${NODE_LABEL_IMAGE}", os.Getenv("NODE_LABEL_IMAGE"),
	}
	fileBytes, err := assets.ReadFile(name)
	if err != nil {
		return nil, err
	}
	policyReplacer := strings.NewReplacer(pairs...)
	transformedString := policyReplacer.Replace(string(fileBytes))
	return []byte(transformedString), nil
}

func RunOperator(ctx context.Context, controllerConfig *controllercmd.ControllerContext) error {
	// Create core clientset and informers
	kubeClient := kubeclient.NewForConfigOrDie(rest.AddUserAgent(controllerConfig.KubeConfig, util.OperatorName))
	kubeInformersForNamespaces := v1helpers.NewKubeInformersForNamespaces(kubeClient, util.OperatorNamespace, "", util.ConfigMapNamespace)
	secretInformer := kubeInformersForNamespaces.InformersFor(util.OperatorNamespace).Core().V1().Secrets()
	nodeInformer := kubeInformersForNamespaces.InformersFor("").Core().V1().Nodes()

	// Create config clientset and informer. This is used to get the cluster ID
	configClient := configclient.NewForConfigOrDie(rest.AddUserAgent(controllerConfig.KubeConfig, util.OperatorName))
	configInformers := configinformers.NewSharedInformerFactory(configClient, 20*time.Minute)

	// Create GenericOperatorclient. This is used by the library-go controllers created down below
	gvr := opv1.GroupVersion.WithResource("clustercsidrivers")
	operatorClient, dynamicInformers, err := goc.NewClusterScopedOperatorClientWithConfigName(controllerConfig.KubeConfig, gvr, util.InstanceName)
	if err != nil {
		return err
	}

	dynamicClient, err := dynamic.NewForConfig(controllerConfig.KubeConfig)
	if err != nil {
		return err
	}

	csiControllerSet := csicontrollerset.NewCSIControllerSet(
		operatorClient,
		controllerConfig.EventRecorder,
	).WithLogLevelController().WithManagementStateController(
		util.OperandName,
		false,
	).WithStaticResourcesController(
		"IBMBlockDriverStaticResourcesController",
		kubeClient,
		dynamicClient,
		kubeInformersForNamespaces,
		assets.ReadFile,
		[]string{
			"configmap.yaml",
			"controller_sa.yaml",
			"csidriver.yaml",
			"node_sa.yaml",
			"rbac/attacher_role.yaml",
			"rbac/attacher_rolebinding.yaml",
			"rbac/provisioner_binding.yaml",
			"rbac/provisioner_role.yaml",
			"rbac/registrar_binding.yaml",
			"rbac/registrar_role.yaml",
			"rbac/resizer_role.yaml",
			"rbac/resizer_rolebinding.yaml",
			"rbac/initcontainer_role.yaml",
			"rbac/initcontainer_rolebinding.yaml",
			"storageclass/vpc-block-10iopsTier-StorageClass.yaml",
			"storageclass/vpc-block-5iopsTier-StorageClass.yaml",
			"storageclass/vpc-block-custom-StorageClass.yaml",
		},
	).WithCSIConfigObserverController(
		"IBMBlockDriverCSIConfigObserverController",
		configInformers,
	).WithCSIDriverControllerService(
		"IBMBlockDriverControllerServiceController",
		assets.ReadFile,
		"controller.yaml",
		kubeClient,
		kubeInformersForNamespaces.InformersFor(util.OperatorNamespace),
		configInformers,
		[]factory.Informer{
			nodeInformer.Informer(),
			secretInformer.Informer(),
		},
		csidrivercontrollerservicecontroller.WithObservedProxyDeploymentHook(),
	).WithCSIDriverNodeService(
		"IBMBlockDriverNodeServiceController",
		readFileAndReplace,
		"node.yaml",
		kubeClient,
		kubeInformersForNamespaces.InformersFor(util.OperatorNamespace),
		nil,
		csidrivernodeservicecontroller.WithObservedProxyDaemonSetHook(),
	)

	if err != nil {
		return err
	}

	secretSyncController := secret.NewSecretSyncController(
		operatorClient,
		kubeClient,
		kubeInformersForNamespaces,
		util.Resync,
		controllerConfig.EventRecorder)

	klog.Info("Starting the informers")
	go kubeInformersForNamespaces.Start(ctx.Done())
	go dynamicInformers.Start(ctx.Done())
	go configInformers.Start(ctx.Done())

	klog.Info("Starting controllerset")
	go secretSyncController.Run(ctx, 1)
	go csiControllerSet.Run(ctx, 1)

	<-ctx.Done()

	return fmt.Errorf("stopped")
}
