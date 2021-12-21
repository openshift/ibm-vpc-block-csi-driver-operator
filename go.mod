module github.com/openshift/ibm-vpc-block-csi-driver-operator

go 1.16

require (
	github.com/IBM/go-sdk-core/v5 v5.7.2
	github.com/IBM/platform-services-go-sdk v0.22.2
	github.com/openshift/api v0.0.0-20211209135129-c58d9f695577
	github.com/openshift/build-machinery-go v0.0.0-20210806203541-4ea9b6da3a37
	github.com/openshift/client-go v0.0.0-20211209144617-7385dd6338e3
	github.com/openshift/library-go v0.0.0-20211214183058-58531ccbde67
	github.com/spf13/cobra v1.2.1
	github.com/stretchr/testify v1.7.0
	k8s.io/api v0.23.0
	k8s.io/apimachinery v0.23.0
	k8s.io/client-go v0.23.0
	k8s.io/component-base v0.23.0
	k8s.io/klog/v2 v2.30.0
)
