package secret

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/IBM/go-sdk-core/v5/core"
	"github.com/IBM/platform-services-go-sdk/resourcemanagerv2"
	operatorv1 "github.com/openshift/api/operator/v1"
	"github.com/openshift/ibm-vpc-block-csi-driver-operator/pkg/util"
	"github.com/openshift/library-go/pkg/controller/factory"
	"github.com/openshift/library-go/pkg/operator/events"
	"github.com/openshift/library-go/pkg/operator/resource/resourceapply"
	"github.com/openshift/library-go/pkg/operator/v1helpers"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	corelisters "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog/v2"
)

// This SecretSyncController translates Secret provided by cloud-credential-operator into
// format required by the CSI driver.
type SecretSyncController struct {
	operatorClient  v1helpers.OperatorClient
	kubeClient      kubernetes.Interface
	secretLister    corelisters.SecretLister
	configMapLister corelisters.ConfigMapLister
	eventRecorder   events.Recorder
	getResourceID   func(resourceName, accountID, apiKey string) (string, error)
}

const (
	// Name of key with ibmcloud_api_key in Secret provided by cloud-credentials-operator.
	cloudSecretKey = "ibmcloud_api_key"
	// Name of key with cloud.conf in ConfigMap provided by cloud-credentials-operator.
	CloudConfigmapKey = "cloud.conf"
	// Name of the key in secret storage-secret-store creating on operator namespace
	StorageSecretStoreKey = "slclient.toml"

	// storage-secret-store data format
	StorageSecretTomlTemplate = `[vpc]
iam_client_id = "bx"
iam_client_secret = "bx"
g2_token_exchange_endpoint_url = "https://iam.cloud.ibm.com"
g2_riaas_endpoint_url = "https://%s.iaas.cloud.ibm.com"	
g2_resource_group_id = "%s" 
g2_api_key = "%s"
provider_type = "g2"
`
)

func NewSecretSyncController(
	operatorClient v1helpers.OperatorClient,
	kubeClient kubernetes.Interface,
	informers v1helpers.KubeInformersForNamespaces,
	resync time.Duration,
	eventRecorder events.Recorder) factory.Controller {

	// Read secret from operator namespace and save the translated one to the operand namespace
	secretInformer := informers.InformersFor(util.OperatorNamespace)
	configMapInformer := informers.InformersFor(util.ConfigMapNamespace)
	c := &SecretSyncController{
		operatorClient:  operatorClient,
		kubeClient:      kubeClient,
		secretLister:    secretInformer.Core().V1().Secrets().Lister(),
		configMapLister: configMapInformer.Core().V1().ConfigMaps().Lister(),
		eventRecorder:   eventRecorder.WithComponentSuffix("SecretSync"),
		getResourceID:   defaultGetResourceID,
	}
	return factory.New().WithSync(c.sync).ResyncEvery(resync).WithSyncDegradedOnError(operatorClient).WithInformers(
		operatorClient.Informer(),
		secretInformer.Core().V1().Secrets().Informer(),
		configMapInformer.Core().V1().Secrets().Informer(),
	).ToController("SecretSync", eventRecorder)
}

func (c *SecretSyncController) sync(ctx context.Context, syncCtx factory.SyncContext) error {
	opSpec, _, _, err := c.operatorClient.GetOperatorState()
	if err != nil {
		klog.V(2).ErrorS(err, "Error while getting operator state")
		return err
	}
	if opSpec.ManagementState != operatorv1.Managed {
		klog.V(2).Info("Operator management state is not managed")
		return nil
	}

	cloudSecret, err := c.secretLister.Secrets(util.OperatorNamespace).Get(util.CloudCredentialSecretName)
	if err != nil {
		if errors.IsNotFound(err) {
			// TODO: report error after some while?
			klog.V(2).Infof("Waiting for secret %s from %s", util.CloudCredentialSecretName, util.OperatorNamespace)
			return nil
		}
		klog.V(2).ErrorS(err, "Secret listener failed to get secret details")
		return err
	}

	cloudConfConfigMap, err := c.configMapLister.ConfigMaps(util.ConfigMapNamespace).Get(util.ConfigMapName)
	if err != nil {
		if errors.IsNotFound(err) {
			// TODO: report error after some while?
			klog.V(2).Infof("Waiting for configmap %s from %s", util.ConfigMapName, util.ConfigMapNamespace)
			return nil
		}
		klog.V(2).ErrorS(err, "Configmap listener failed to get cm details")
		return err
	}

	// Get the storage-secret-store secret to be created from ibm-cloud-credential secret and clod-conf configmap
	driverSecret, err := c.translateSecret(cloudSecret, cloudConfConfigMap)
	if err != nil {
		klog.V(2).ErrorS(err, "Error while extracting data from secret/cm")
		return err
	}
	_, _, err = resourceapply.ApplySecret(ctx, c.kubeClient.CoreV1(), c.eventRecorder, driverSecret)
	if err != nil {
		klog.V(2).ErrorS(err, "Error while creating the secret")
		return err
	}
	klog.V(2).Infof("%s secret created successfully", util.IBMCSIDriverSecretName)
	return nil
}

func (c *SecretSyncController) translateSecret(cloudSecret *v1.Secret, cloudConf *v1.ConfigMap) (*v1.Secret, error) {
	apiKey, ok := cloudSecret.Data[cloudSecretKey]
	if !ok {
		return nil, fmt.Errorf("cloud-credential-operator secret %s did not contain key %s", util.CloudCredentialSecretName, cloudSecretKey)
	}
	conf, ok := cloudConf.Data[CloudConfigmapKey]
	if !ok {
		return nil, fmt.Errorf("cloud-credential-operator configmap %s did not contain key %s", util.ConfigMapName, cloudSecretKey)
	}

	var re *regexp.Regexp
	var match []string

	// Extracting the region from configmap
	re = regexp.MustCompile("region = (.*?)\n")
	match = re.FindStringSubmatch(conf)
	if len(match) <= 0 {
		return nil, fmt.Errorf("cloud-credential-operator configmap %s did not contain region", util.ConfigMapName)
	}
	region := match[1]

	re = regexp.MustCompile("g2ResourceGroupName = (.*?)\n")
	match = re.FindStringSubmatch(conf)
	if len(match) <= 1 {
		return nil, fmt.Errorf("cloud-credential-operator configmap %s did not contain resourcegroupname", util.ConfigMapName)
	}
	resourceGroupName := match[1]

	re = regexp.MustCompile("accountID = (.*?)\n")
	match = re.FindStringSubmatch(conf)
	if len(match) <= 1 {
		return nil, fmt.Errorf("cloud-credential-operator configmap %s did not contain accountID", util.ConfigMapName)
	}
	accountID := match[1]

	resourceId, err := c.getResourceID(resourceGroupName, accountID, string(apiKey))
	if err != nil {
		return nil, err
	}

	// Creating secret data storage-secret-store
	tomlData := fmt.Sprintf(StorageSecretTomlTemplate, region, resourceId, apiKey)
	data := make(map[string][]byte)
	data[StorageSecretStoreKey] = []byte(tomlData)
	secret := v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      util.IBMCSIDriverSecretName,
			Namespace: util.OperatorNamespace,
		},
		Type: v1.SecretTypeOpaque,
		Data: data,
	}

	return &secret, nil
}

func defaultGetResourceID(resourceName, accountID, apiKey string) (string, error) {
	serviceClientOptions := &resourcemanagerv2.ResourceManagerV2Options{
		URL:           "https://resource-controller.cloud.ibm.com",
		Authenticator: &core.IamAuthenticator{ApiKey: apiKey},
	}
	serviceClient, err := resourcemanagerv2.NewResourceManagerV2UsingExternalConfig(serviceClientOptions)
	if err != nil {
		return "", err
	}
	listResourceGroupsOptions := serviceClient.NewListResourceGroupsOptions()
	listResourceGroupsOptions.SetAccountID(accountID)
	resourceGroupList, _, err := serviceClient.ListResourceGroups(listResourceGroupsOptions)

	if err != nil {
		return "", err
	}
	if len(resourceGroupList.Resources) > 0 {
		for _, v := range resourceGroupList.Resources {
			if *v.Name == resourceName {
				resourceID := *v.ID
				return resourceID, nil
			}
		}
	}
	return "", fmt.Errorf("Resource %s not found for given g2Credentials", resourceName)
}
