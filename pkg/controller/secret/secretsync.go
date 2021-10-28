package secret

import (
	"context"
	b64 "encoding/base64"
	"fmt"
	operatorv1 "github.com/openshift/api/operator/v1"
	"github.com/IBM/ibm-vpc-block-csi-driver-operator/pkg/util"
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
	"regexp"
	"time"
)

// This SecretSyncController translates Secret provided by cloud-credential-operator into
// format required by the CSI driver.
type SecretSyncController struct {
	operatorClient  v1helpers.OperatorClient
	kubeClient      kubernetes.Interface
	secretLister    corelisters.SecretLister
	configMapLister corelisters.ConfigMapLister
	eventRecorder   events.Recorder
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

g2_token_exchange_endpoint_url = "https://iam.bluemix.net"
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
		return err
	}
	if opSpec.ManagementState != operatorv1.Managed {
		return nil
	}

	cloudSecret, err := c.secretLister.Secrets(util.OperatorNamespace).Get(util.CloudCredentialSecretName)
	if err != nil {
		if errors.IsNotFound(err) {
			// TODO: report error after some while?
			klog.V(2).Infof("Waiting for secret %s from %s", util.CloudCredentialSecretName, util.OperatorNamespace)
			return nil
		}
		return err
	}

	cloudConfConfigMap, err := c.configMapLister.ConfigMaps(util.ConfigMapNamespace).Get(util.ConfigMapName)
	if err != nil {
		if errors.IsNotFound(err) {
			// TODO: report error after some while?
			klog.V(2).Infof("Waiting for configmap %s from %s", util.CloudCredentialSecretName, util.ConfigMapNamespace)
			return nil
		}
		return err
	}

	// Get the storage-secret-store secret to be created from ibm-cloud-credential secret and clod-conf configmap
	driverSecret, err := c.translateSecret(cloudSecret, cloudConfConfigMap)
	if err != nil {
		return err
	}

	_, _, err = resourceapply.ApplySecret(c.kubeClient.CoreV1(), c.eventRecorder, driverSecret)
	if err != nil {
		klog.V(2).Infof("Error while creating the secret %s", err)
		return err
	}
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

	// Extracting the region from configmap
	re := regexp.MustCompile("region = (.*?)\n")
	match := re.FindStringSubmatch(conf)
	if len(match) <= 0 {
		return nil, fmt.Errorf("cloud-credential-operator configmap %s did not contain region", util.ConfigMapName)
	}
	region := match[0]

	resourceId := "test-id" // TODO add resource id

	// Creating secret data storage-secret-store
	tomlData := fmt.Sprintf(StorageSecretTomlTemplate, region, resourceId, apiKey)
	//tomlDataEncoded := b64.StdEncoding.EncodeToString([]byte(tomlData))
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
