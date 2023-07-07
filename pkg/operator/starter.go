package operator

import (
	"context"
	"fmt"
	"os"
	"strings"

	apiextclient "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	kubeclient "k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"

	opv1 "github.com/openshift/api/operator/v1"
	configclient "github.com/openshift/client-go/config/clientset/versioned"
	configinformers "github.com/openshift/client-go/config/informers/externalversions"
	opclient "github.com/openshift/client-go/operator/clientset/versioned"
	opinformers "github.com/openshift/client-go/operator/informers/externalversions"
	"github.com/openshift/ibm-vpc-block-csi-driver-operator/assets"
	"github.com/openshift/ibm-vpc-block-csi-driver-operator/pkg/controller/secret"
	"github.com/openshift/ibm-vpc-block-csi-driver-operator/pkg/util"
	"github.com/openshift/library-go/pkg/controller/controllercmd"
	"github.com/openshift/library-go/pkg/controller/factory"
	"github.com/openshift/library-go/pkg/operator/csi/csicontrollerset"
	"github.com/openshift/library-go/pkg/operator/csi/csidrivercontrollerservicecontroller"
	"github.com/openshift/library-go/pkg/operator/csi/csidrivernodeservicecontroller"
	goc "github.com/openshift/library-go/pkg/operator/genericoperatorclient"
	"github.com/openshift/library-go/pkg/operator/v1helpers"
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
	configMapInformer := kubeInformersForNamespaces.InformersFor(util.OperatorNamespace).Core().V1().ConfigMaps()
	nodeInformer := kubeInformersForNamespaces.InformersFor("").Core().V1().Nodes()

	// Create apiextension client. This is used to verify is a VolumeSnapshotClass CRD exists.
	apiExtClient, err := apiextclient.NewForConfig(rest.AddUserAgent(controllerConfig.KubeConfig, util.OperatorName))
	if err != nil {
		return err
	}

	// Create config clientset and informer. This is used to get the cluster ID
	configClient := configclient.NewForConfigOrDie(rest.AddUserAgent(controllerConfig.KubeConfig, util.OperatorName))
	configInformers := configinformers.NewSharedInformerFactory(configClient, util.Resync)

	// operator.openshift.io client, used for ClusterCSIDriver
	operatorClientSet := opclient.NewForConfigOrDie(rest.AddUserAgent(controllerConfig.KubeConfig, util.OperatorName))
	operatorInformers := opinformers.NewSharedInformerFactory(operatorClientSet, util.Resync)

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
			"cabundle_cm.yaml",
			"rbac/main_attacher_binding.yaml",
			"rbac/provisioner_binding.yaml",
			"rbac/provisioner_role.yaml",
			"rbac/main_provisioner_binding.yaml",
			"rbac/volumesnapshot_reader_provisioner_binding.yaml",
			"rbac/configmap_and_secret_reader_provisioner_binding.yaml",
			"rbac/node_privileged_binding.yaml",
			"rbac/privileged_role.yaml",
			"rbac/node_label_updater_binding.yaml",
			"rbac/label_updater_role.yaml",
			"rbac/resizer_role.yaml",
			"rbac/resizer_rolebinding.yaml",
			"rbac/main_resizer_binding.yaml",
			"rbac/initcontainer_role.yaml",
			"rbac/initcontainer_rolebinding.yaml",
			"rbac/snapshotter_binding.yaml",
			"rbac/snapshotter_role.yaml",
			"rbac/main_snapshotter_binding.yaml",
			"rbac/lease_leader_election_role.yaml",
			"rbac/lease_leader_election_rolebinding.yaml",
		},
	).WithConditionalStaticResourcesController(
		"IBMBlockDriverConditionalStaticResourcesController",
		kubeClient,
		dynamicClient,
		kubeInformersForNamespaces,
		assets.ReadFile,
		[]string{
			"volumesnapshotclass.yaml",
		},
		// Only install when CRD exists.
		func() bool {
			name := "volumesnapshotclasses.snapshot.storage.k8s.io"
			_, err := apiExtClient.ApiextensionsV1().CustomResourceDefinitions().Get(context.TODO(), name, metav1.GetOptions{})
			return err == nil
		},
		// Don't ever remove.
		func() bool {
			return false
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
			configMapInformer.Informer(),
		},
		csidrivercontrollerservicecontroller.WithObservedProxyDeploymentHook(),
		csidrivercontrollerservicecontroller.WithCABundleDeploymentHook(
			util.OperatorNamespace,
			util.TrustedCAConfigMap,
			configMapInformer,
		),
	).WithCSIDriverNodeService(
		"IBMBlockDriverNodeServiceController",
		readFileAndReplace,
		"node.yaml",
		kubeClient,
		kubeInformersForNamespaces.InformersFor(util.OperatorNamespace),
		[]factory.Informer{configMapInformer.Informer()},
		csidrivernodeservicecontroller.WithObservedProxyDaemonSetHook(),
		csidrivernodeservicecontroller.WithCABundleDaemonSetHook(
			util.OperatorNamespace,
			util.TrustedCAConfigMap,
			configMapInformer,
		),
	).WithStorageClassController(
		"IBMBlockStorageClassController",
		assets.ReadFile,
		[]string{
			"storageclass/vpc-block-10iopsTier-StorageClass.yaml",
			"storageclass/vpc-block-5iopsTier-StorageClass.yaml",
			"storageclass/vpc-block-custom-StorageClass.yaml",
		},
		kubeClient,
		kubeInformersForNamespaces.InformersFor(""),
		operatorInformers,
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
	go operatorInformers.Start(ctx.Done())

	klog.Info("Starting controllerset")
	go secretSyncController.Run(ctx, 1)
	go csiControllerSet.Run(ctx, 1)

	<-ctx.Done()

	return fmt.Errorf("stopped")
}
