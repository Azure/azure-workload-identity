package kuberneteshelper

import (
	"context"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GetKubeConfig returns the kubeconfig
func GetKubeConfig() (*rest.Config, error) {
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{}).ClientConfig()
}

// GetKubeClient returns a Kubernetes clientset.
func GetKubeClient() (client.Client, error) {
	kubeConfig, err := GetKubeConfig()
	if err != nil {
		return nil, err
	}

	return client.New(kubeConfig, client.Options{})
}

// GetObject returns an object from the Kubernetes cluster.
func GetObject(ctx context.Context, kubeClient client.Client, namespace string, name string, obj client.Object) (client.Object, error) {
	err := kubeClient.Get(ctx, client.ObjectKey{
		Namespace: namespace,
		Name:      name,
	}, obj)

	return obj, err
}
