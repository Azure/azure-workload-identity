package kuberneteshelper

import (
	"context"
	"fmt"
	"time"

	"github.com/Azure/azure-workload-identity/pkg/webhook"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Create ServiceAccount in the cluster
// If the ServiceAccount already exists, error is returned
func CreateOrUpdateServiceAccount(ctx context.Context, kubeClient client.Client, namespace, name, clientID, tenantID string, tokenExpiration time.Duration) error {
	sa := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				webhook.UseWorkloadIdentityLabel: "true",
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

	err := kubeClient.Create(ctx, sa)
	if apierrors.IsAlreadyExists(err) {
		err = kubeClient.Update(ctx, sa)
	}
	return err
}

// Delete ServiceAccount in the cluster
func DeleteServiceAccount(ctx context.Context, kubeClient client.Client, namespace, name string) error {
	sa := &corev1.ServiceAccount{}
	if err := kubeClient.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, sa); err != nil {
		return err
	}
	return kubeClient.Delete(ctx, sa)
}
