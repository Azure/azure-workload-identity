package kuberneteshelper

import (
	"context"
	"testing"

	"github.com/Azure/azure-workload-identity/pkg/webhook"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

func TestCreateServiceAccount(t *testing.T) {
	testNamespace := "test-namespace"
	testServiceAccountName := "test-service-account"

	tests := []struct {
		name        string
		initObjects []runtime.Object
		wantErr     bool
	}{
		{
			name: "service account already exists",
			initObjects: []runtime.Object{
				&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      testServiceAccountName,
						Namespace: testNamespace,
					},
				},
			},
			wantErr: true,
		},
		{
			name:        "no error",
			initObjects: []runtime.Object{},
			wantErr:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create fake client
			k8sClient := fake.NewSimpleClientset(tt.initObjects...)

			if err := CreateServiceAccount(k8sClient, testNamespace, testServiceAccountName, "client-id", "tenant-id"); (err != nil) != tt.wantErr {
				t.Errorf("CreateServiceAccount() error = %v, wantErr %v", err, tt.wantErr)
			}
			if !tt.wantErr {
				// check if service account was created and has correct annotations
				sa, err := k8sClient.CoreV1().ServiceAccounts(testNamespace).Get(context.TODO(), testServiceAccountName, metav1.GetOptions{})
				if err != nil {
					t.Errorf("CreateServiceAccount() error = %v, wantErr %v", err, tt.wantErr)
				}
				if sa.Annotations[webhook.ClientIDAnnotation] != "client-id" {
					t.Errorf("CreateServiceAccount() clientID annotation = %v, want %v", sa.Annotations[webhook.ClientIDAnnotation], "client-id")
				}
				if sa.Annotations[webhook.TenantIDAnnotation] != "tenant-id" {
					t.Errorf("CreateServiceAccount() tenantID annotation = %v, want %v", sa.Annotations[webhook.TenantIDAnnotation], "tenant-id")
				}
				if sa.Labels[webhook.UsePodIdentityLabel] != "true" {
					t.Errorf("CreateServiceAccount() usePodIdentity label = %v, want %v", sa.Labels[webhook.UsePodIdentityLabel], "true")
				}
			}
		})
	}
}

func TestDeleteServiceAccount(t *testing.T) {
	testNamespace := "test-namespace"
	testServiceAccountName := "test-service-account"

	tests := []struct {
		name        string
		initObjects []runtime.Object
		wantErr     bool
	}{
		{
			name:        "service account does not exist",
			initObjects: []runtime.Object{},
			wantErr:     false,
		},
		{
			name: "no error",
			initObjects: []runtime.Object{
				&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      testServiceAccountName,
						Namespace: testNamespace,
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// create fake client
			k8sClient := fake.NewSimpleClientset(tt.initObjects...)

			if err := DeleteServiceAccount(k8sClient, testNamespace, testServiceAccountName); (err != nil) != tt.wantErr {
				t.Errorf("DeleteService Account() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
