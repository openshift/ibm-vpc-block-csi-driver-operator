package secret

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/openshift/ibm-vpc-block-csi-driver-operator/pkg/util"
	k8v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func fakeGetResourceID(resourceName, accountID, apiKey string) (string, error) {
	return "fakeid", nil
}

func TestTranslateSecretError(t *testing.T) {
	secretNamespace := "test-ns-operator"
	secretName := "ibm-cloud-credential"
	cmNamespace := "test-ns-cco"
	cmName := "cloud-conf"
	c := &SecretSyncController{
		getResourceID: defaultGetResourceID,
	}
	type args struct {
		cloudSecret *k8v1.Secret
		cloudConf   *k8v1.ConfigMap
	}
	tests := []struct {
		name string
		args args
	}{
		{
			name: "Empty secret",
			args: args{
				cloudSecret: &k8v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      secretName,
						Namespace: secretNamespace,
					},
				},
				cloudConf: &k8v1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      cmName,
						Namespace: cmNamespace,
					},
				},
			},
		}, {
			name: "Empty configmap",
			args: args{
				cloudSecret: &k8v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      secretName,
						Namespace: secretNamespace,
					},
					Data: map[string][]byte{cloudSecretKey: []byte("test")},
				},
				cloudConf: &k8v1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      cmName,
						Namespace: cmNamespace,
					},
				},
			},
		}, {
			name: "Empty region",
			args: args{
				cloudSecret: &k8v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      secretName,
						Namespace: secretNamespace,
					},
					Data: map[string][]byte{cloudSecretKey: []byte("test")},
				},
				cloudConf: &k8v1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      cmName,
						Namespace: cmNamespace,
					},
					Data: map[string]string{CloudConfigmapKey: "cm-data"},
				},
			},
		}, {
			name: "Empty ResourceGroupName",
			args: args{
				cloudSecret: &k8v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      secretName,
						Namespace: secretNamespace,
					},
					Data: map[string][]byte{cloudSecretKey: []byte("test")},
				},
				cloudConf: &k8v1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      cmName,
						Namespace: cmNamespace,
					},
					Data: map[string]string{CloudConfigmapKey: "region = region1\n"},
				},
			},
		}, {
			name: "Empty accountID",
			args: args{
				cloudSecret: &k8v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      secretName,
						Namespace: secretNamespace,
					},
					Data: map[string][]byte{cloudSecretKey: []byte("test")},
				},
				cloudConf: &k8v1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      cmName,
						Namespace: cmNamespace,
					},
					Data: map[string]string{CloudConfigmapKey: "region = region1\ng2ResourceGroupName = testresource\n"},
				},
			},
		}, {
			name: "Error getting resource ID",
			args: args{
				cloudSecret: &k8v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      secretName,
						Namespace: secretNamespace,
					},
					Data: map[string][]byte{cloudSecretKey: []byte("test")},
				},
				cloudConf: &k8v1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      cmName,
						Namespace: cmNamespace,
					},
					Data: map[string]string{CloudConfigmapKey: "region = region1\ng2ResourceGroupName = testresource\naccountID = testaccount\n"},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := c.translateSecret(tt.args.cloudSecret, tt.args.cloudConf)
			if err == nil {
				t.Errorf("translateSecret() no error returned %v", err)
				return
			}
			if got != nil {
				t.Errorf("translateSecret() got = %v is not nil", got)
			}
		})
	}
}

func TestTranslateSecretSuccess(t *testing.T) {
	secretNamespace := "test-ns-operator"
	secretName := "ibm-cloud-credential"
	cmNamespace := "test-ns-cco"
	cmName := "cloud-conf"
	resourceId := "fakeid"
	apiKey := "testapikey"
	region := "region1"

	tomlData := fmt.Sprintf(StorageSecretTomlTemplate, region, resourceId, apiKey)
	data := make(map[string][]byte)
	data[StorageSecretStoreKey] = []byte(tomlData)
	want := &k8v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      util.IBMCSIDriverSecretName,
			Namespace: util.OperatorNamespace,
		},
		Type: k8v1.SecretTypeOpaque,
		Data: data,
	}

	c := &SecretSyncController{
		getResourceID: fakeGetResourceID,
	}
	cloudSecret := &k8v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: secretNamespace,
		},
		Data: map[string][]byte{cloudSecretKey: []byte(apiKey)},
	}
	cloudConf := &k8v1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cmName,
			Namespace: cmNamespace,
		},
		Data: map[string]string{CloudConfigmapKey: "region = region1\ng2ResourceGroupName = testresource\naccountID = testaccount\n"},
	}
	got, err := c.translateSecret(cloudSecret, cloudConf)
	if err != nil {
		t.Errorf("translateSecret() error: %v", err)
	} else if !reflect.DeepEqual(got, want) {
		t.Errorf("translateSecret() got = %v, want %v", *got, *want)
	}
}
