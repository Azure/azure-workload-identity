package serviceaccount

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Azure/azure-workload-identity/pkg/cloud/mock_cloud"
	"github.com/Azure/azure-workload-identity/pkg/webhook"

	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/azure-sdk-for-go/services/preview/authorization/mgmt/2018-01-01-preview/authorization"
	"github.com/Azure/go-autorest/autorest/to"
	"github.com/golang/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

const (
	testTenantID = "test-tenant-id"
	testAppID    = "test-app-id"

	trueValue = "true"
)

func TestCreateCmdCalidate(t *testing.T) {
	tests := []struct {
		name      string
		createCmd createCmd
		wantErr   bool
	}{
		{
			name: "token expiration >= minimum token expiration",
			createCmd: createCmd{
				authProvider: &mockAuthProvider{
					authArgs: &authArgs{},
				},
				tokenExpiration: 1 * time.Hour,
			},
			wantErr: false,
		},
		{
			name: "token expiration < minimum token expiration",
			createCmd: createCmd{
				authProvider: &mockAuthProvider{
					authArgs: &authArgs{},
				},
				tokenExpiration: 1 * time.Minute,
			},
			wantErr: true,
		},
		{
			name: "token expiration > maximum token expiration",
			createCmd: createCmd{
				authProvider: &mockAuthProvider{
					authArgs: &authArgs{},
				},
				tokenExpiration: 25 * time.Hour,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.createCmd.validate()
			if (err == nil && tt.wantErr) || (err != nil && !tt.wantErr) {
				t.Errorf("validate() got err = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCreateCmdRun(t *testing.T) {
	tests := []struct {
		name      string
		createCmd createCmd
		expect    func(m *mock_cloud.MockInterfaceMockRecorder)
		verify    func(t *testing.T, createCmd createCmd, err error)
	}{
		{
			name: "application not found",
			createCmd: createCmd{
				authProvider: &mockAuthProvider{
					authArgs: &authArgs{},
				},

				name:       "foo",
				namespace:  "bar",
				issuer:     "https://issuer-url",
				azureRole:  "role",
				azureScope: "scope",
				kubeClient: fake.NewSimpleClientset(),
			},
			expect: func(m *mock_cloud.MockInterfaceMockRecorder) {
				m.GetApplication(gomock.Any(), gomock.Any()).Return(graphrbac.Application{}, errors.New("app not found"))
				m.CreateApplication(gomock.Any(), gomock.Any()).Return(graphrbac.Application{}, errors.New("failed to create application"))
			},
			verify: func(t *testing.T, createCmd createCmd, err error) {
				if err == nil {
					t.Errorf("run() error is nil, expected error")
				}
			},
		},
		{
			name: "service principal not found",
			createCmd: createCmd{
				authProvider: &mockAuthProvider{
					authArgs: &authArgs{},
				},

				name:       "foo",
				namespace:  "bar",
				issuer:     "https://issuer-url",
				azureRole:  "role",
				azureScope: "scope",
				kubeClient: fake.NewSimpleClientset(),
			},
			expect: func(m *mock_cloud.MockInterfaceMockRecorder) {
				m.GetApplication(gomock.Any(), gomock.Any()).Return(graphrbac.Application{}, errors.New("app not found"))
				m.CreateApplication(gomock.Any(), gomock.Any()).Return(graphrbac.Application{
					DisplayName: to.StringPtr("app"),
					ObjectID:    to.StringPtr("app-object-id"),
					AppID:       to.StringPtr(testAppID),
				}, nil)
				m.GetServicePrincipal(gomock.Any(), gomock.Any()).Return(graphrbac.ServicePrincipal{}, errors.New("service principal not found"))
				m.CreateServicePrincipal(gomock.Any(), gomock.Any(), gomock.Any()).Return(graphrbac.ServicePrincipal{
					ObjectID:    to.StringPtr("sp-object-id"),
					DisplayName: to.StringPtr("app"),
				}, errors.New("failed to create service principal"))
			},
			verify: func(t *testing.T, createCmd createCmd, err error) {
				if err == nil {
					t.Errorf("run() error is nil, expected error")
				}
			},
		},
		{
			name: "service account already exists and is updated",
			createCmd: createCmd{
				authProvider: &mockAuthProvider{
					authArgs: &authArgs{
						tenantID: testTenantID,
					},
				},

				name:       "foo",
				namespace:  "bar",
				issuer:     "https://issuer-url",
				azureRole:  "role",
				azureScope: "scope",
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
				m.GetApplication(gomock.Any(), gomock.Any()).Return(graphrbac.Application{}, errors.New("app not found"))
				m.CreateApplication(gomock.Any(), gomock.Any()).Return(graphrbac.Application{
					DisplayName: to.StringPtr("app"),
					ObjectID:    to.StringPtr("app-object-id"),
					AppID:       to.StringPtr(testAppID),
				}, nil)
				m.GetServicePrincipal(gomock.Any(), gomock.Any()).Return(graphrbac.ServicePrincipal{}, errors.New("service principal not found"))
				m.CreateServicePrincipal(gomock.Any(), gomock.Any(), gomock.Any()).Return(graphrbac.ServicePrincipal{
					DisplayName: to.StringPtr("app"),
					ObjectID:    to.StringPtr("sp-object-id"),
				}, nil)
				m.AddFederatedCredential(gomock.Any(), gomock.Any(), gomock.Any()).Return(errors.New("failed to add federated credential"))
			},
			verify: func(t *testing.T, createCmd createCmd, err error) {
				if err == nil {
					t.Errorf("run() error is nil, expected error")
				}
				var sa *corev1.ServiceAccount
				if sa, err = createCmd.kubeClient.CoreV1().ServiceAccounts(createCmd.namespace).Get(context.TODO(), createCmd.name, metav1.GetOptions{}); err != nil {
					t.Errorf("GetServiceAccount() error = %v, wantErr %v", err, nil)
				}
				if sa.Annotations[webhook.ClientIDAnnotation] != testAppID {
					t.Errorf("%s annotation = %s, want %s", webhook.ClientIDAnnotation, sa.Annotations[webhook.ClientIDAnnotation], testAppID)
				}
				if sa.Annotations[webhook.TenantIDAnnotation] != testTenantID {
					t.Errorf("%s annotation = %s, want %s", webhook.TenantIDAnnotation, sa.Annotations[webhook.TenantIDAnnotation], testTenantID)
				}
				if sa.Labels[webhook.UsePodIdentityLabel] != trueValue {
					t.Errorf("%s label = %s, want %s", webhook.UsePodIdentityLabel, sa.Labels[webhook.UsePodIdentityLabel], trueValue)
				}
			},
		},
		{
			name: "failed to add role assignment",
			createCmd: createCmd{
				authProvider: &mockAuthProvider{
					authArgs: &authArgs{
						tenantID: testTenantID,
					},
				},

				name:       "foo",
				namespace:  "bar",
				issuer:     "https://issuer-url",
				azureRole:  "role",
				azureScope: "scope",
				kubeClient: fake.NewSimpleClientset(),
			},
			expect: func(m *mock_cloud.MockInterfaceMockRecorder) {
				m.GetApplication(gomock.Any(), gomock.Any()).Return(graphrbac.Application{}, errors.New("app not found"))
				m.CreateApplication(gomock.Any(), gomock.Any()).Return(graphrbac.Application{
					DisplayName: to.StringPtr("app"),
					ObjectID:    to.StringPtr("app-object-id"),
					AppID:       to.StringPtr(testAppID),
				}, nil)
				m.GetServicePrincipal(gomock.Any(), gomock.Any()).Return(graphrbac.ServicePrincipal{}, errors.New("service principal not found"))
				m.CreateServicePrincipal(gomock.Any(), gomock.Any(), gomock.Any()).Return(graphrbac.ServicePrincipal{
					DisplayName: to.StringPtr("app"),
					ObjectID:    to.StringPtr("sp-object-id"),
				}, nil)
				m.AddFederatedCredential(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				m.CreateRoleAssignment(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(authorization.RoleAssignment{}, errors.New("failed to create role assignment"))
			},
			verify: func(t *testing.T, createCmd createCmd, err error) {
				if err == nil {
					t.Errorf("run() error is nil, expected error")
				}
				var sa *corev1.ServiceAccount
				if sa, err = createCmd.kubeClient.CoreV1().ServiceAccounts(createCmd.namespace).Get(context.TODO(), createCmd.name, metav1.GetOptions{}); err != nil {
					t.Errorf("GetServiceAccount() error = %v, wantErr %v", err, nil)
				}
				if sa.Annotations[webhook.ClientIDAnnotation] != testAppID {
					t.Errorf("%s annotation = %s, want %s", webhook.ClientIDAnnotation, sa.Annotations[webhook.ClientIDAnnotation], testAppID)
				}
				if sa.Annotations[webhook.TenantIDAnnotation] != testTenantID {
					t.Errorf("%s annotation = %s, want %s", webhook.TenantIDAnnotation, sa.Annotations[webhook.TenantIDAnnotation], testTenantID)
				}
				if sa.Labels[webhook.UsePodIdentityLabel] != trueValue {
					t.Errorf("%s label = %s, want %s", webhook.UsePodIdentityLabel, sa.Labels[webhook.UsePodIdentityLabel], trueValue)
				}
			},
		},
		{
			name: "successful create cmd run",
			createCmd: createCmd{
				authProvider: &mockAuthProvider{
					authArgs: &authArgs{
						tenantID: testTenantID,
					},
				},

				name:       "foo",
				namespace:  "bar",
				issuer:     "https://issuer-url",
				azureRole:  "role",
				azureScope: "scope",
				kubeClient: fake.NewSimpleClientset(),
			},
			expect: func(m *mock_cloud.MockInterfaceMockRecorder) {
				m.GetApplication(gomock.Any(), gomock.Any()).Return(graphrbac.Application{}, errors.New("app not found"))
				m.CreateApplication(gomock.Any(), gomock.Any()).Return(graphrbac.Application{
					DisplayName: to.StringPtr("app"),
					ObjectID:    to.StringPtr("app-object-id"),
					AppID:       to.StringPtr(testAppID),
				}, nil)
				m.GetServicePrincipal(gomock.Any(), gomock.Any()).Return(graphrbac.ServicePrincipal{}, errors.New("service principal not found"))
				m.CreateServicePrincipal(gomock.Any(), gomock.Any(), gomock.Any()).Return(graphrbac.ServicePrincipal{
					DisplayName: to.StringPtr("app"),
					ObjectID:    to.StringPtr("sp-object-id"),
				}, nil)
				m.AddFederatedCredential(gomock.Any(), gomock.Any(), gomock.Any()).Return(nil)
				m.CreateRoleAssignment(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(authorization.RoleAssignment{
					ID: to.StringPtr("role-assignment-id"),
				}, nil)
			},
			verify: func(t *testing.T, createCmd createCmd, err error) {
				if err != nil {
					t.Errorf("run() error = %v, want nil error", err)
				}
				var sa *corev1.ServiceAccount
				if sa, err = createCmd.kubeClient.CoreV1().ServiceAccounts(createCmd.namespace).Get(context.TODO(), createCmd.name, metav1.GetOptions{}); err != nil {
					t.Errorf("GetServiceAccount() error = %v, wantErr %v", err, nil)
				}
				if sa.Annotations[webhook.ClientIDAnnotation] != testAppID {
					t.Errorf("%s annotation = %s, want %s", webhook.ClientIDAnnotation, sa.Annotations[webhook.ClientIDAnnotation], testAppID)
				}
				if sa.Annotations[webhook.TenantIDAnnotation] != testTenantID {
					t.Errorf("%s annotation = %s, want %s", webhook.TenantIDAnnotation, sa.Annotations[webhook.TenantIDAnnotation], testTenantID)
				}
				if sa.Labels[webhook.UsePodIdentityLabel] != trueValue {
					t.Errorf("%s label = %s, want %s", webhook.UsePodIdentityLabel, sa.Labels[webhook.UsePodIdentityLabel], trueValue)
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

			tt.createCmd.azureClient = clientMock
			tt.createCmd.kubeClient = fake.NewSimpleClientset()

			err := tt.createCmd.run()
			tt.verify(t, tt.createCmd, err)
		})
	}
}
