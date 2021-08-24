package util

import (
	"os"
	"testing"
)

func TestGetNamespace(t *testing.T) {
	tests := []struct {
		name         string
		podNamespace string
		want         string
	}{
		{
			name:         "default webhook namespace",
			podNamespace: "",
			want:         "azure-workload-identity-system",
		},
		{
			name:         "namespace set",
			podNamespace: "kube-system",
			want:         "kube-system",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.podNamespace != "" {
				os.Setenv("POD_NAMESPACE", tt.podNamespace)
				defer os.Unsetenv("POD_NAMESPACE")
			}

			if got := GetNamespace(); got != tt.want {
				t.Errorf("GetNamespace() = %v, want %v", got, tt.want)
			}
		})
	}
}
