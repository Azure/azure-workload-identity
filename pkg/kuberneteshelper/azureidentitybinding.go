package kuberneteshelper

import (
	"context"
	"sort"

	aadpodv1 "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type azureIdentityBindings []aadpodv1.AzureIdentityBinding

func (a azureIdentityBindings) Len() int {
	return len(a)
}

func (a azureIdentityBindings) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a azureIdentityBindings) Less(i, j int) bool {
	if a[i].Namespace == a[j].Namespace {
		return a[i].Name < a[j].Name
	}
	return a[i].Namespace < a[j].Namespace
}

// ListAzureIdentityBinding returns a list of AzureIdentityBinding
func ListAzureIdentityBinding(ctx context.Context, kubeClient client.Client, namespace string) (map[string]aadpodv1.AzureIdentityBinding, error) {
	list := &aadpodv1.AzureIdentityBindingList{}
	if err := kubeClient.List(ctx, list, client.InNamespace(namespace)); err != nil {
		return nil, err
	}

	sort.Sort(azureIdentityBindings(list.Items))
	azureIdentityBindingMap := make(map[string]aadpodv1.AzureIdentityBinding)
	for _, binding := range list.Items {
		azureIdentityBindingMap[binding.Name] = binding
	}

	return azureIdentityBindingMap, nil
}
