package secret

import (
	"bou.ke/monkey"
	"fmt"
	"github.com/IBM/ibm-vpc-block-csi-driver-operator/pkg/util"
	k8v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
	"testing"
)

func TestSecretSyncController_translateSecret(t *testing.T) {
	secretNamespace := "test-ns-operator"
	secretName := "ibm-cloud-credential"
	cmNamespace := "test-ns-cco"
	cmName := "cloud-conf"
	c := &SecretSyncController{}
	type args struct {
		cloudSecret *k8v1.Secret
		cloudConf   *k8v1.ConfigMap
	}
	tests := []struct {
		name    string
		args    args
		want    *k8v1.Secret
		wantErr bool
	}{
		{
			name: "Empty secret",
			want: nil,
			wantErr: true,
			args: args{
				cloudSecret: &k8v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:  		secretName,
						Namespace: secretNamespace,
					},
				},
				cloudConf:	&k8v1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      cmName,
						Namespace: cmNamespace,
					},
				},
			},
		}, {
			name: "Empty configmap",
			want: nil,
			wantErr: true,
			args: args{
				cloudSecret: &k8v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:  		secretName,
						Namespace: secretNamespace,
					},
					Data: map[string][]byte{cloudSecretKey: []byte("test")},
				},
				cloudConf:	&k8v1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      cmName,
						Namespace: cmNamespace,
					},
				},
			},
		}, {
			name: "Empty ResourceGroupName",
			want: nil,
			wantErr: true,
			args: args{
				cloudSecret: &k8v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:  		secretName,
						Namespace: secretNamespace,
					},
					Data: map[string][]byte{cloudSecretKey: []byte("test")},
				},
				cloudConf:	&k8v1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      cmName,
						Namespace: cmNamespace,
					},
					Data: map[string]string{CloudConfigmapKey: "cm-data"},
				},
			},
		}, {
			name: "Empty accountID",
			want: nil,
			wantErr: true,
			args: args{
				cloudSecret: &k8v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:  		secretName,
						Namespace: secretNamespace,
					},
					Data: map[string][]byte{cloudSecretKey: []byte("test")},
				},
				cloudConf:	&k8v1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						Name:      cmName,
						Namespace: cmNamespace,
					},
					Data: map[string]string{CloudConfigmapKey: "region = region1\ng2ResourceGroupName = testresource\n"},
				},
			},
		}, {
			name: "Error getting resource ID",
			want: nil,
			wantErr: true,
			args: args{
				cloudSecret: &k8v1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:  		secretName,
						Namespace: secretNamespace,
					},
					Data: map[string][]byte{cloudSecretKey: []byte("test")},
				},
				cloudConf:	&k8v1.ConfigMap{
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
			if (err != nil) != tt.wantErr {
				t.Errorf("translateSecret() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("translateSecret() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSecretSyncController_translateSecret1(t *testing.T) {
	secretNamespace := "test-ns-operator"
	secretName := "ibm-cloud-credential"
	cmNamespace := "test-ns-cco"
	cmName := "cloud-conf"
	resourceId := "fakeid"
	apiKey := "testapikey"

	tomlData := fmt.Sprintf(StorageSecretTomlTemplate, resourceId, apiKey)
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

	c := &SecretSyncController{}
	cloudSecret := &k8v1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:  		secretName,
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
	monkey.Patch(getResourceID, func(resourceName, accountID, apiKey string) (string, error) {
		return resourceId, nil
	})
	got, err := c.translateSecret(cloudSecret, cloudConf)

	if !reflect.DeepEqual(got, want) {
		t.Errorf("translateSecret() got = %v, want %v", *got, *want)
		t.Errorf("translateSecret() error = %v, wantErr %v", err, nil)
	}
}
