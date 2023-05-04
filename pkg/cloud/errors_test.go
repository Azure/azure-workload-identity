package cloud

import (
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore"
	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/microsoftgraph/msgraph-sdk-go/models/odataerrors"
	"github.com/pkg/errors"
)

func TestIsNotFound(t *testing.T) {
	tests := []struct {
		name      string
		actualErr error
		want      bool
	}{
		{
			name:      "not found error",
			actualErr: errors.New("resource not found"),
			want:      true,
		},
		{
			name:      "not not found error",
			actualErr: errors.New("something else"),
			want:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNotFound(tt.actualErr); got != tt.want {
				t.Errorf("IsNotFound() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsRoleAssignmentAlreadyDeleted(t *testing.T) {
	tests := []struct {
		name      string
		actualErr error
		want      bool
	}{
		{
			name:      "not azcore response error",
			actualErr: errors.New("role assignment already deleted"),
			want:      false,
		},
		{
			name:      "status code doesn't match",
			actualErr: &azcore.ResponseError{StatusCode: 404, ErrorCode: "role assignment not found"},
			want:      false,
		},
		{
			name:      "role assignment already deleted error",
			actualErr: &azcore.ResponseError{StatusCode: 204, ErrorCode: "role assignment already deleted"},
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRoleAssignmentAlreadyDeleted(tt.actualErr); got != tt.want {
				t.Errorf("IsRoleAssignmentAlreadyDeleted() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsRoleAssignmentExists(t *testing.T) {
	tests := []struct {
		name      string
		actualErr error
		want      bool
	}{
		{
			name:      "not azcore response error",
			actualErr: errors.New("resource already exists"),
			want:      false,
		},
		{
			name:      "status code doesn't match",
			actualErr: &azcore.ResponseError{StatusCode: 401, ErrorCode: "authorization failed"},
			want:      false,
		},
		{
			name:      "resource already exists error",
			actualErr: &azcore.ResponseError{StatusCode: 409, ErrorCode: "resource already exists"},
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRoleAssignmentExists(tt.actualErr); got != tt.want {
				t.Errorf("IsAlreadyExists() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsFederatedCredentialNotFound(t *testing.T) {
	tests := []struct {
		name      string
		actualErr func() error
		want      bool
	}{
		{
			name:      "not graph error",
			actualErr: func() error { return errors.New("resource not found") },
			want:      false,
		},
		{
			name: "graph error code doesn't match",
			actualErr: func() error {
				err := GraphError{Errorable: odataerrors.NewMainError()}
				err.Errorable.SetCode(to.Ptr("random_error_code"))
				return err
			},
			want: false,
		},
		{
			name: "graph error resource not found",
			actualErr: func() error {
				err := GraphError{Errorable: odataerrors.NewMainError()}
				err.Errorable.SetCode(to.Ptr(GraphErrorCodeResourceNotFound))
				return err
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsFederatedCredentialNotFound(tt.actualErr()); got != tt.want {
				t.Errorf("IsFederatedCredentialNotFound() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsFederatedCredentialAlreadyExists(t *testing.T) {
	tests := []struct {
		name      string
		actualErr func() error
		want      bool
	}{
		{
			name:      "not graph error",
			actualErr: func() error { return errors.New("resource already exists") },
			want:      false,
		},
		{
			name: "graph error code doesn't match",
			actualErr: func() error {
				err := GraphError{Errorable: odataerrors.NewMainError()}
				err.Errorable.SetCode(to.Ptr("random_error_code"))
				return err
			},
			want: false,
		},
		{
			name: "graph error resource already exists",
			actualErr: func() error {
				err := GraphError{Errorable: odataerrors.NewMainError()}
				err.Errorable.SetCode(to.Ptr(GraphErrorCodeMultipleObjectsWithSameKeyValue))
				return err
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsFederatedCredentialAlreadyExists(tt.actualErr()); got != tt.want {
				t.Errorf("IsFederatedCredentialAlreadyExists() = %v, want %v", got, tt.want)
			}
		})
	}
}
