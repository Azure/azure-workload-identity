package kuberneteshelper

import (
	"context"

	"github.com/Azure/azure-workload-identity/pkg/webhook"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// GetKubeConfig returns the kubeconfig
func GetKubeConfig() (*rest.Config, error) {
	return clientcmd.NewNonInteractiveDeferredLoadingClientConfig(clientcmd.NewDefaultClientConfigLoadingRules(), &clientcmd.ConfigOverrides{}).ClientConfig()
}

// GetKubeClient returns a Kubernetes clientset.
func GetKubeClient() (kubernetes.Interface, error) {
	kubeConfig, err := GetKubeConfig()
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfigOrDie(kubeConfig), nil
}

// Create ServiceAccount in the cluster
// If the ServiceAccount already exists, error is returned
func CreateServiceAccount(kubeClient kubernetes.Interface, namespace, name, clientID, tenantID string) error {
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				webhook.UsePodIdentityLabel: "true",
			},
			Annotations: map[string]string{
				webhook.ClientIDAnnotation: clientID,
				webhook.TenantIDAnnotation: tenantID,
			},
		},
	}

	_, err := kubeClient.CoreV1().ServiceAccounts(namespace).Create(context.TODO(), sa, metav1.CreateOptions{})
	return err
}

// Delete ServiceAccount in the cluster
// If the ServiceAccount does not exist, no error is returned
func DeleteServiceAccount(kubeClient kubernetes.Interface, namespace, name string) error {
	err := kubeClient.CoreV1().ServiceAccounts(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}
