package operator

import (
	opv1 "github.com/openshift/api/operator/v1"
	oplisterv1 "github.com/openshift/client-go/operator/listers/operator/v1"
	"github.com/openshift/library-go/pkg/operator/csi/csistorageclasscontroller"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/klog/v2"
)

// getEncryptionKeyHook checks for IBMCloudCSIDriverConfigSpec in the
// ClusterCSIDriver object. If it contains EncryptionKeyCRN, it sets the
// corresponding parameters in the StorageClass. This allows the admin to
// specify a customer managed key to be used by default.
func getEncryptionKeyHook(ccdLister oplisterv1.ClusterCSIDriverLister) csistorageclasscontroller.StorageClassHookFunc {
	return func(_ *opv1.OperatorSpec, class *storagev1.StorageClass) error {
		ccd, err := ccdLister.Get(class.Provisioner)
		if err != nil {
			return err
		}

		driverConfig := ccd.Spec.DriverConfig
		if driverConfig.DriverType != opv1.IBMCloudDriverType || driverConfig.IBMCloud == nil {
			klog.V(4).Infof("No IBMCloudCSIDriverConfigSpec defined for %s", class.Provisioner)
			return nil
		}

		crn := driverConfig.IBMCloud.EncryptionKeyCRN
		if crn == "" {
			klog.V(4).Infof("Not setting empty %s parameter in StorageClass %s", encryptionKeyParameter, class.Name)
			return nil
		}

		if class.Parameters == nil {
			class.Parameters = map[string]string{}
		}
		klog.V(4).Infof("Setting %s = %s in StorageClass %s", encryptionKeyParameter, crn, class.Name)
		class.Parameters[encryptionKeyParameter] = crn
		class.Parameters[encryptedParameter] = "true"
		return nil
	}
}
