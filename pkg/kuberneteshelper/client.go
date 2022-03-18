package kuberneteshelper

import (
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
