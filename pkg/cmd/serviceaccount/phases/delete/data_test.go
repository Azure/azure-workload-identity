package phases

import (
	"fmt"

	"github.com/microsoftgraph/msgraph-sdk-go/models"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/Azure/azure-workload-identity/pkg/cloud"
	"github.com/Azure/azure-workload-identity/pkg/cmd/serviceaccount/util"
)

type mockDeleteData struct {
	serviceAccountName      string
	serviceAccountNamespace string
	serviceAccountIssuerURL string
	aadApplication          models.Applicationable // cache
	aadApplicationName      string
	aadApplicationObjectID  string
	roleAssignmentID        string
	azureClient             cloud.Interface
	kubeClient              client.Client
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

func (d *mockDeleteData) AADApplication() (models.Applicationable, error) {
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

func (d *mockDeleteData) KubeClient() (client.Client, error) {
	if d.kubeClient == nil {
		return nil, errors.New("not found")
	}
	return d.kubeClient, nil
}
