package phases

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/Azure/azure-workload-identity/pkg/cmd/serviceaccount/phases/workflow"
)

func TestServiceAccountPreRun(t *testing.T) {
	tests := []struct {
		name     string
		phase    workflow.Phase
		data     interface{}
		errorMsg string
	}{
		{
			name:     "invalid data type",
			data:     "test",
			errorMsg: "invalid data type string",
		},
		{
			name:     "missing --service-account-namespace",
			data:     &mockDeleteData{},
			errorMsg: "--service-account-namespace is required",
		},
		{
			name:     "missing --service-account-name",
			data:     &mockDeleteData{serviceAccountNamespace: "test"},
			errorMsg: "--service-account-name is required",
		},
		{
			name: "valid data",
			data: &mockDeleteData{
				serviceAccountNamespace: "test",
				serviceAccountName:      "test",
				kubeClient:              fake.NewClientBuilder().Build(),
			},
			errorMsg: "",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := NewServiceAccountPhase().PreRun(test.data)
			if err == nil {
				if test.errorMsg != "" {
					t.Errorf("expected error but got nil")
				}
			} else if err.Error() != test.errorMsg {
				t.Errorf("expected error message: %s, but got: %s", test.errorMsg, err.Error())
			}
		})
	}
}

func TestServiceAccountRun(t *testing.T) {
	phase := NewServiceAccountPhase()
	data := &mockDeleteData{
		serviceAccountNamespace: "service-account-namespace",
		serviceAccountName:      "service-account-name",
	}

	kubeClient := fake.NewClientBuilder().WithObjects([]client.Object{
		&corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      data.serviceAccountName,
				Namespace: data.serviceAccountNamespace,
			},
		},
	}...).Build()
	data.kubeClient = kubeClient

	err := phase.PreRun(data)
	if err != nil {
		t.Errorf("expected no error but got: %s", err.Error())
	}
	err = phase.Run(context.TODO(), data)
	if err != nil {
		t.Errorf("expected no error but got: %s", err.Error())
	}
	sa := &corev1.ServiceAccount{}
	if err := kubeClient.Get(context.TODO(), types.NamespacedName{Namespace: data.serviceAccountNamespace, Name: data.serviceAccountName}, sa); err == nil {
		t.Errorf("expected service account to be deleted")
	}

	// Test for service account not found
	phase = NewServiceAccountPhase()
	kubeClient = fake.NewClientBuilder().Build()
	data.kubeClient = kubeClient
	err = phase.PreRun(data)
	if err != nil {
		t.Errorf("expected no error but got: %s", err.Error())
	}
	err = phase.Run(context.Background(), data)
	if err != nil {
		t.Errorf("expected no error but got: %s", err.Error())
	}
}
