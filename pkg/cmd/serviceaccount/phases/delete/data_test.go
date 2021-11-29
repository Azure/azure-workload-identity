package phases

import (
	"fmt"

	"github.com/Azure/azure-workload-identity/pkg/cloud"
	"github.com/Azure/azure-workload-identity/pkg/cmd/serviceaccount/util"

	"github.com/microsoftgraph/msgraph-beta-sdk-go/models/microsoft/graph"
	"github.com/pkg/errors"
	"k8s.io/client-go/kubernetes"
)

type mockDeleteData struct {
	serviceAccountName      string
	serviceAccountNamespace string
	serviceAccountIssuerURL string
	aadApplication          *graph.Application // cache
	aadApplicationName      string
	aadApplicationObjectID  string
	roleAssignmentID        string
	azureClient             cloud.Interface
	kubeClient              kubernetes.Interface
}

var _ DeleteData = &mockDeleteData{}

func (d *mockDeleteData) ServiceAccountName() string {
	return d.serviceAccountName
}

func (d *mockDeleteData) ServiceAccountNamespace() string {
	return d.serviceAccountNamespace
}

func (d *mockDeleteData) ServiceAccountIssuerURL() string {
	return d.serviceAccountIssuerURL
}

func (d *mockDeleteData) AADApplication() (*graph.Application, error) {
	if d.aadApplication == nil {
		return nil, errors.New("not found")
	}
	return d.aadApplication, nil
}

func (d *mockDeleteData) AADApplicationName() string {
	if d.aadApplicationName == "" && d.ServiceAccountNamespace() != "" && d.ServiceAccountName() != "" && d.ServiceAccountIssuerURL() != "" {
		return fmt.Sprintf("%s-%s-%s", d.ServiceAccountNamespace(), d.serviceAccountName, util.GetIssuerHash(d.ServiceAccountIssuerURL()))
	}
	return d.aadApplicationName
}

func (d *mockDeleteData) AADApplicationObjectID() string {
	return d.aadApplicationObjectID
}

func (d *mockDeleteData) RoleAssignmentID() string {
	return d.roleAssignmentID
}

func (d *mockDeleteData) AzureClient() cloud.Interface {
	return d.azureClient
}

func (d *mockDeleteData) KubeClient() (kubernetes.Interface, error) {
	if d.kubeClient == nil {
		return nil, errors.New("not found")
	}
	return d.kubeClient, nil
}
