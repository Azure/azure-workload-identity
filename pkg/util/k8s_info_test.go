package util

import (
	"testing"

	"k8s.io/apimachinery/pkg/version"
	fakediscovery "k8s.io/client-go/discovery/fake"

	"k8s.io/client-go/discovery"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/rest"
)

func TestAssertK8sMinVersion(t *testing.T) {
	orig := newDiscoveryClientForConfig
	defer func() { newDiscoveryClientForConfig = orig }()

	config := rest.Config{}
	client := fake.NewClientset().Discovery()

	newDiscoveryClientForConfig = func(c *rest.Config) (discovery.DiscoveryInterface, error) {
		return client, nil
	}

	tests := []struct {
		major     string
		minor     string
		expected  bool
		expectErr bool
	}{
		{
			major:     "1",
			minor:     "32",
			expected:  true,
			expectErr: false,
		},
		{
			major:     "2",
			minor:     "1",
			expected:  true,
			expectErr: false,
		},
		{
			major:     "1",
			minor:     "31",
			expected:  false,
			expectErr: false,
		},
		{
			major:     "3",
			minor:     "foo",
			expected:  false,
			expectErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.major+"."+tt.minor, func(t *testing.T) {
			client.(*fakediscovery.FakeDiscovery).FakedServerVersion = &version.Info{
				Major: tt.major,
				Minor: tt.minor,
			}

			major := 1
			minor := 32

			client.ServerVersion()
			result, err := AssertK8sMinVersion(&config, major, minor)
			if err != nil && !tt.expectErr {
				t.Errorf("AssertK8sMinVersion(%v, %d, %d) errored", config, major, minor)
			}
			if result != tt.expected {
				t.Errorf("AssertK8sMinVersion(%v, %d, %d) = %t, want %t", config, major, minor, result, tt.expected)
			}
		})
	}
}
