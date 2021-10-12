package kuberneteshelper

import (
	"context"
	"fmt"
	"time"

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
func CreateOrUpdateServiceAccount(ctx context.Context, kubeClient kubernetes.Interface, namespace, name, clientID, tenantID string, tokenExpiration time.Duration) error {
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

	if tokenExpiration != time.Duration(webhook.DefaultServiceAccountTokenExpiration)*time.Second {
		// Round to the nearest second before converting to a string
		sa.ObjectMeta.Annotations[webhook.ServiceAccountTokenExpiryAnnotation] = fmt.Sprintf("%.0f", tokenExpiration.Round(time.Second).Seconds())
	}

	serviceAccount, err := kubeClient.CoreV1().ServiceAccounts(namespace).Create(ctx, sa, metav1.CreateOptions{})
	if apierrors.IsAlreadyExists(err) {
		// Update the existing service account
		sa.ObjectMeta.ResourceVersion = serviceAccount.ObjectMeta.ResourceVersion
		_, err = kubeClient.CoreV1().ServiceAccounts(namespace).Update(ctx, sa, metav1.UpdateOptions{})
	}
	return err
}

// Delete ServiceAccount in the cluster
func DeleteServiceAccount(ctx context.Context, kubeClient kubernetes.Interface, namespace, name string) error {
	return kubeClient.CoreV1().ServiceAccounts(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}
