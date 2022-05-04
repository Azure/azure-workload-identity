package kuberneteshelper

import (
	"context"

	aadpodv1 "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ListAzureIdentity returns a list of AzureIdentity
func ListAzureIdentity(ctx context.Context, kubeClient client.Client, namespace string) (map[string]aadpodv1.AzureIdentity, error) {
	list := &aadpodv1.AzureIdentityList{}
	if err := kubeClient.List(ctx, list, client.InNamespace(namespace)); err != nil {
		return nil, err
	}

	azureIdentityMap := make(map[string]aadpodv1.AzureIdentity)
	for _, identity := range list.Items {
		azureIdentityMap[identity.Name] = identity
	}

	return azureIdentityMap, nil
}
