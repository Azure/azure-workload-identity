package serviceaccount

import (
	"context"
	"net/http"
	"testing"

	"github.com/Azure/azure-workload-identity/pkg/cloud"
	"github.com/Azure/azure-workload-identity/pkg/cloud/mock_cloud"

	"github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-01-01-preview/authorization"
	"github.com/Azure/go-autorest/autorest"
	"github.com/golang/mock/gomock"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestDeleteCmdRun(t *testing.T) {
	tests := []struct {
		name      string
		deleteCmd deleteCmd
		expect    func(m *mock_cloud.MockInterfaceMockRecorder)
		verify    func(t *testing.T, dc deleteCmd, err error)
	}{
		{
			name: "failed to delete role assignment",
			deleteCmd: deleteCmd{
				name:             "foo",
				namespace:        "bar",
				issuer:           "https://issuer-url",
				roleAssignmentID: "role-assignment-id",
				appObjectID:      "application-id",
				kubeClient:       fake.NewSimpleClientset(),
			},
			expect: func(m *mock_cloud.MockInterfaceMockRecorder) {
				m.DeleteRoleAssignment(gomock.Any(), gomock.Any()).Return(authorization.RoleAssignment{}, errors.New("failed to delete role assignment"))
			},
			verify: func(t *testing.T, dc deleteCmd, err error) {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			},
		},
		{
			name: "role assignment already deleted",
			deleteCmd: deleteCmd{
				name:             "foo",
				namespace:        "bar",
				issuer:           "https://issuer-url",
				roleAssignmentID: "role-assignment-id",
				appObjectID:      "application-id",
				kubeClient:       fake.NewSimpleClientset(),
			},
			expect: func(m *mock_cloud.MockInterfaceMockRecorder) {
				m.DeleteRoleAssignment(gomock.Any(), gomock.Any()).Return(authorization.RoleAssignment{}, autorest.DetailedError{StatusCode: http.StatusNoContent})
				m.GetFederatedCredential(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(cloud.FederatedCredential{}, errors.New("failed to get federated credential"))
			},
			verify: func(t *testing.T, dc deleteCmd, err error) {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			},
		},
		{
			name: "failed to delete federated identity credential",
			deleteCmd: deleteCmd{
				name:             "foo",
				namespace:        "bar",
				issuer:           "https://issuer-url",
				roleAssignmentID: "role-assignment-id",
				appObjectID:      "application-id",
				kubeClient:       fake.NewSimpleClientset(),
			},
			expect: func(m *mock_cloud.MockInterfaceMockRecorder) {
				m.DeleteRoleAssignment(gomock.Any(), gomock.Any()).Return(authorization.RoleAssignment{}, autorest.DetailedError{StatusCode: http.StatusNoContent})
				m.GetFederatedCredential(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(cloud.FederatedCredential{ID: "fic-id"}, nil)
				m.DeleteFederatedCredential(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("failed to delete federated identity credential"))
			},
			verify: func(t *testing.T, dc deleteCmd, err error) {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			},
		},
		{
			name: "failed to delete application",
			deleteCmd: deleteCmd{
				name:             "foo",
				namespace:        "bar",
				issuer:           "https://issuer-url",
				roleAssignmentID: "role-assignment-id",
				appObjectID:      "application-id",
				kubeClient: fake.NewSimpleClientset(
					&corev1.ServiceAccount{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "foo",
							Namespace: "bar",
						},
					},
				),
			},
			expect: func(m *mock_cloud.MockInterfaceMockRecorder) {
				m.DeleteRoleAssignment(gomock.Any(), gomock.Any()).Return(authorization.RoleAssignment{}, autorest.DetailedError{StatusCode: http.StatusNoContent})
				m.GetFederatedCredential(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(cloud.FederatedCredential{ID: "fic-id"}, nil)
				m.DeleteFederatedCredential(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				m.DeleteApplication(gomock.Any(), gomock.Any()).Return(autorest.Response{}, errors.New("failed to delete application"))
			},
			verify: func(t *testing.T, dc deleteCmd, err error) {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
				// check service account has been deleted
				if _, err = dc.kubeClient.CoreV1().ServiceAccounts("bar").Get(context.TODO(), "foo", metav1.GetOptions{}); err == nil {
					t.Errorf("expected service account to be deleted")
				}
				if !apierrors.IsNotFound(err) {
					t.Errorf("expected not found error, got %v", err)
				}
			},
		},
		{
			name: "successful delete cmd run",
			deleteCmd: deleteCmd{
				name:             "foo",
				namespace:        "bar",
				issuer:           "https://issuer-url",
				roleAssignmentID: "role-assignment-id",
				appObjectID:      "application-id",
				kubeClient: fake.NewSimpleClientset(
					&corev1.ServiceAccount{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "foo",
							Namespace: "bar",
						},
					},
				),
			},
			expect: func(m *mock_cloud.MockInterfaceMockRecorder) {
				m.DeleteRoleAssignment(gomock.Any(), gomock.Any()).Return(authorization.RoleAssignment{}, autorest.DetailedError{StatusCode: http.StatusNoContent})
				m.GetFederatedCredential(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(cloud.FederatedCredential{ID: "fic-id"}, nil)
				m.DeleteFederatedCredential(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				m.DeleteApplication(gomock.Any(), gomock.Any()).Return(autorest.Response{}, nil)
			},
			verify: func(t *testing.T, dc deleteCmd, err error) {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
				// check service account has been deleted
				if _, err = dc.kubeClient.CoreV1().ServiceAccounts("bar").Get(context.TODO(), "foo", metav1.GetOptions{}); err == nil {
					t.Errorf("expected service account to be deleted")
				}
				if !apierrors.IsNotFound(err) {
					t.Errorf("expected not found error, got %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			clientMock := mock_cloud.NewMockInterface(ctrl)
			tt.expect(clientMock.EXPECT())

			tt.deleteCmd.azureClient = clientMock
			tt.deleteCmd.kubeClient = fake.NewSimpleClientset()

			err := tt.deleteCmd.run()
			tt.verify(t, tt.deleteCmd, err)
		})
	}
}
