package cloud

import (
	"testing"

	"github.com/Azure/go-autorest/autorest"
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
			name:      "not autorest detailed error",
			actualErr: errors.New("role assignment already deleted"),
			want:      false,
		},
		{
			name:      "status code doesn't match",
			actualErr: &autorest.DetailedError{StatusCode: 404, Message: "role assignment not found"},
			want:      false,
		},
		{
			name:      "role assignment already deleted error",
			actualErr: autorest.DetailedError{StatusCode: 204, Message: "role assignment already deleted"},
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

func TestIsResourceNotFound(t *testing.T) {
	tests := []struct {
		name      string
		actualErr error
		want      bool
	}{
		{
			name:      "not autorest detailed error",
			actualErr: errors.New("resource not found"),
			want:      false,
		},
		{
			name:      "status code doesn't match",
			actualErr: &autorest.DetailedError{StatusCode: 401, Message: "authorization failed"},
			want:      false,
		},
		{
			name:      "resource not found error",
			actualErr: autorest.DetailedError{StatusCode: 404, Message: "resource not found"},
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsResourceNotFound(tt.actualErr); got != tt.want {
				t.Errorf("IsResourceNotFound() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIsAlreadyExists(t *testing.T) {
	tests := []struct {
		name      string
		actualErr error
		want      bool
	}{
		{
			name:      "not autorest detailed error",
			actualErr: errors.New("resource already exists"),
			want:      false,
		},
		{
			name:      "status code doesn't match",
			actualErr: &autorest.DetailedError{StatusCode: 401, Message: "authorization failed"},
			want:      false,
		},
		{
			name:      "resource already exists error",
			actualErr: autorest.DetailedError{StatusCode: 409, Message: "resource already exists"},
			want:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsAlreadyExists(tt.actualErr); got != tt.want {
				t.Errorf("IsAlreadyExists() = %v, want %v", got, tt.want)
			}
		})
	}
}
