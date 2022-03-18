package kuberneteshelper

import (
	"context"

	aadpodv1 "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ListAzureIdentityBinding returns a list of AzureIdentityBinding
func ListAzureIdentityBinding(ctx context.Context, kubeClient client.Client, namespace string) (map[string]aadpodv1.AzureIdentityBinding, error) {
	list := &aadpodv1.AzureIdentityBindingList{}
	if err := kubeClient.List(ctx, list, client.InNamespace(namespace)); err != nil {
		return nil, err
	}

	azureIdentityBindingMap := make(map[string]aadpodv1.AzureIdentityBinding)
	for _, binding := range list.Items {
		azureIdentityBindingMap[binding.Name] = binding
	}

	return azureIdentityBindingMap, nil
}
