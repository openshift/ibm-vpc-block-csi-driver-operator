package csidriver

import (
	"context"
	"github.com/IBM/ibm-vpc-block-csi-driver-operator/pkg/util"
	operatorv1 "github.com/openshift/api/operator/v1"
	"github.com/openshift/library-go/pkg/controller/factory"
	"github.com/openshift/library-go/pkg/operator/events"
	"github.com/openshift/library-go/pkg/operator/v1helpers"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/kubernetes"
	storagelisters "k8s.io/client-go/listers/storage/v1"
	"k8s.io/klog/v2"
)

// This VPCBlockController watches ibm-cloud-credentials running on
// operator namespace and clod-conf configmap running on the
// openshift-cloud-controller-manager namespace. It creates a secret
// storage-secret-store after getting necessary values from the above resources.

type VPCBlockController struct {
	operatorClient     v1helpers.OperatorClient
	kubeClient         kubernetes.Interface
	storageClassLister storagelisters.StorageClassLister
	csiControllers     []Runnable
	controllersRunning bool
	eventRecorder      events.Recorder
}

type Runnable interface {
	Run(ctx context.Context, workers int)
}

func NewVPCBlockController(
	operatorClient v1helpers.OperatorClient,
	kubeClient kubernetes.Interface,
	informers v1helpers.KubeInformersForNamespaces,
	csiControllers []Runnable,
	eventRecorder events.Recorder) factory.Controller {

	c := &VPCBlockController{
		operatorClient: operatorClient,
		kubeClient:     kubeClient,
		csiControllers: csiControllers,
		eventRecorder:  eventRecorder.WithComponentSuffix("ibm-vpc-block-csi-driver-operator"),
	}
	return factory.New().WithSync(c.sync).WithSyncDegradedOnError(operatorClient).ResyncEvery(util.Resync).WithInformers(
		operatorClient.Informer(),
	).ToController("VPCBlockCSIController", eventRecorder)
}

func (c *VPCBlockController) sync(ctx context.Context, syncCtx factory.SyncContext) error {
	klog.V(4).Infof("IBM VPC Block CSI driver operator sync started")
	defer klog.V(4).Infof("IBM VPC Block CSI driver operator sync finished")

	opSpec, _, _, err := c.operatorClient.GetOperatorState()
	if err != nil {
		return err
	}
	if opSpec.ManagementState != operatorv1.Managed {
		return nil
	}
	klog.V(4).Infof("Starting CSI driver controllers")
	for _, ctrl := range c.csiControllers {
		if ctrl == nil {
			continue
		}
		go func(ctrl Runnable) {
			defer utilruntime.HandleCrash()
			ctrl.Run(ctx, 1)
		}(ctrl)
	}

	return nil
}
