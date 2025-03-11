package util

import (
	"strconv"

	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
)

var newDiscoveryClientForConfig = func(c *rest.Config) (discovery.DiscoveryInterface, error) {
	dc, err := discovery.NewDiscoveryClientForConfig(c)
	var di discovery.DiscoveryInterface = dc
	return di, err
}

// Return true if the server's version is greater or equal than the specified version
func AssertK8sMinVersion(config *rest.Config, minMajor int, minMinor int) (bool, error) {
	discoveryClient, err := newDiscoveryClientForConfig(config)
	if err != nil {
		return false, err
	}
	serverVersion, err := discoveryClient.ServerVersion()
	if err != nil {
		return false, err
	}
	major, err := strconv.Atoi(serverVersion.Major)
	if err != nil {
		return false, err
	}
	minor, err := strconv.Atoi(serverVersion.Minor)
	if err != nil {
		return false, err
	}
	return major > minMajor || (major == minMajor && minor >= minMinor), nil
}
