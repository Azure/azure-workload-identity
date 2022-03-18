package kuberneteshelper

import (
	"context"

	aadpodv1 "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ListAzureIdentityBinding(ctx context.Context, kubeClient client.Client, namespace string) map[string]aadpodv1.AzureIdentityBinding {
	return nil
}
