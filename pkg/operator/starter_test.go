package operator

import (
	"context"
	"testing"
	"time"

	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/record"

	"github.com/stretchr/testify/assert"

	"github.com/openshift/library-go/pkg/controller/controllercmd"
	"github.com/openshift/library-go/pkg/operator/events"
)

const (
	Namespace string = "openshift-cluster-csi-drivers"
)

func fakeKubeConfig() *rest.Config {
	config := "apiVersion: v1\nclusters:\n- cluster:\n    server: http://localhost\n  name: fake\ncontexts:\n- context:\n    cluster: fake\n    user: \"\"\n  name: fake\ncurrent-context: fake\nkind: Config\npreferences: {}"
	kubeConfig, _ := clientcmd.NewClientConfigFromBytes([]byte(config))
	clientConfig, _ := kubeConfig.ClientConfig()
	return clientConfig
}

func fakeEventRecorder() events.Recorder {
	clientset := fake.NewSimpleClientset()
	controllerRef, _ := events.GetControllerReferenceForCurrentPod(clientset, Namespace, nil)
	eventRecorder := events.NewKubeRecorderWithOptions(
		clientset.CoreV1().Events(Namespace),
		record.CorrelatorOptions{},
		"",
		controllerRef)
	return eventRecorder
}

func fakeControllerConfig() controllercmd.ControllerContext {
	var NewFakeControllerConfig = controllercmd.ControllerContext{}
	kubeConfig := fakeKubeConfig()
	eventRecorder := fakeEventRecorder()
	NewFakeControllerConfig.KubeConfig = kubeConfig
	NewFakeControllerConfig.EventRecorder = eventRecorder
	return NewFakeControllerConfig
}

func TestRunOperatorConfigNull(t *testing.T) {
	newFakeControllerConfig := fakeControllerConfig()
	ctx, cancel := context.WithCancel(context.Background())
	go RunOperator(ctx, &newFakeControllerConfig)
	time.Sleep(5 * time.Second)
	cancel()
	assert.NoError(t, nil)
}
