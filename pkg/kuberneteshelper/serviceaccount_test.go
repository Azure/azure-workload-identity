package kuberneteshelper

import (
	"context"
	"testing"
	"time"

	"github.com/Azure/azure-workload-identity/pkg/webhook"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const (
	testNamespace          = "test-namespace"
	testServiceAccountName = "test-service-account"
)

func TestCreateOrUpdateServiceAccount(t *testing.T) {
	// create fake client
	k8sClient := fake.NewClientBuilder().Build()

	if err := CreateOrUpdateServiceAccount(context.TODO(), k8sClient, testNamespace, testServiceAccountName, "client-id", "tenant-id", 3600*time.Second+500*time.Millisecond); err != nil {
		t.Errorf("CreateServiceAccount() error = %v, wantErr %v", err, false)
	}
	sa := &corev1.ServiceAccount{}
	// check if service account was created and has correct annotations
	err := k8sClient.Get(context.TODO(), types.NamespacedName{Name: testServiceAccountName, Namespace: testNamespace}, sa)
	if err != nil {
		t.Errorf("CreateServiceAccount() error = %v, wantErr %v", err, false)
	}
	if sa.Annotations[webhook.ClientIDAnnotation] != "client-id" {
		t.Errorf("CreateServiceAccount() clientID annotation = %v, want %v", sa.Annotations[webhook.ClientIDAnnotation], "client-id")
	}
	if sa.Annotations[webhook.TenantIDAnnotation] != "tenant-id" {
		t.Errorf("CreateServiceAccount() tenantID annotation = %v, want %v", sa.Annotations[webhook.TenantIDAnnotation], "tenant-id")
	}
	// also test for rounding (i.e. 3600.5s -> 3601s)
	if sa.Annotations[webhook.ServiceAccountTokenExpiryAnnotation] != "3601" {
		t.Errorf("CreateServiceAccount() token expiry annotation = %v, want %v", sa.Annotations[webhook.ServiceAccountTokenExpiryAnnotation], "3601")
	}
	if sa.Labels[webhook.UseWorkloadIdentityLabel] != "true" {
		t.Errorf("CreateServiceAccount() useWorkloadIdentity label = %v, want %v", sa.Labels[webhook.UseWorkloadIdentityLabel], "true")
	}
}

func TestCreateOrUpdateServiceAccountDefaultTokenExpiration(t *testing.T) {
	// create fake client
	k8sClient := fake.NewClientBuilder().Build()

	if err := CreateOrUpdateServiceAccount(context.TODO(), k8sClient, testNamespace, testServiceAccountName, "client-id", "tenant-id", time.Duration(webhook.DefaultServiceAccountTokenExpiration)*time.Second); err != nil {
		t.Errorf("CreateServiceAccount() error = %v, wantErr %v", err, false)
	}
	sa := &corev1.ServiceAccount{}
	// check if service account was created and has correct annotations
	err := k8sClient.Get(context.TODO(), types.NamespacedName{Name: testServiceAccountName, Namespace: testNamespace}, sa)
	if err != nil {
		t.Errorf("CreateServiceAccount() error = %v, wantErr %v", err, false)
	}
	if sa.Annotations[webhook.ClientIDAnnotation] != "client-id" {
		t.Errorf("CreateServiceAccount() clientID annotation = %v, want %v", sa.Annotations[webhook.ClientIDAnnotation], "client-id")
	}
	if sa.Annotations[webhook.TenantIDAnnotation] != "tenant-id" {
		t.Errorf("CreateServiceAccount() tenantID annotation = %v, want %v", sa.Annotations[webhook.TenantIDAnnotation], "tenant-id")
	}
	if _, ok := sa.Annotations[webhook.ServiceAccountTokenExpiryAnnotation]; ok {
		t.Errorf("CreateServiceAccount() token expiry annotation should not be set")
	}
	if sa.Labels[webhook.UseWorkloadIdentityLabel] != "true" {
		t.Errorf("CreateServiceAccount() useWorkloadIdentity label = %v, want %v", sa.Labels[webhook.UseWorkloadIdentityLabel], "true")
	}
}

func TestDeleteServiceAccount(t *testing.T) {
	tests := []struct {
		name        string
		initObjects []client.Object
		wantErr     bool
	}{
		{
			name:        "service account does not exist",
			initObjects: []client.Object{},
			wantErr:     true,
		},
		{
			name: "no error",
			initObjects: []client.Object{
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
			k8sClient := fake.NewClientBuilder().WithObjects(tt.initObjects...).Build()

			if err := DeleteServiceAccount(context.TODO(), k8sClient, testNamespace, testServiceAccountName); (err != nil) != tt.wantErr {
				t.Errorf("DeleteService Account() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
