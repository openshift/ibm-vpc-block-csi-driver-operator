package operator

import (
	opv1 "github.com/openshift/api/operator/v1"
	oplisterv1 "github.com/openshift/client-go/operator/listers/operator/v1"
	"k8s.io/klog/v2"
)

const (
	encryptionKey = "encryptionKey"
)

var crn string

// setEncryptionKey checks for IBMCloudCSIDriverConfigSpec in the ClusterCSIDriver object.
// If it contains EncryptionKeyCRN, it sets the corresponding parameter in the StorageClass.
// This allows the admin to specify a customer managed key to be used by default in storage classes, which will be picked during pvc creation.

func setEncryptionKey(ccdLister oplisterv1.ClusterCSIDriverLister) {
	ccd, err := ccdLister.Get("vpc.block.csi.ibm.io")
	if err != nil {
		klog.V(4).Infof("Error %s", err.Error())
		return
	}

	driverConfig := ccd.Spec.DriverConfig
	if driverConfig.DriverType != opv1.IBMCloudDriverType || driverConfig.IBMCloud == nil {
		klog.V(4).Info("No IBMCloudCSIDriverConfigSpec defined")
		return
	}

	crn = driverConfig.IBMCloud.EncryptionKeyCRN
	if crn == "" {
		klog.V(4).Info("EncryptionKeyCRN is empty")
	}
}
