// +build e2e

package e2e

import (
	"fmt"
	"path"
	"testing"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/ginkgo/config"
	"github.com/onsi/ginkgo/reporters"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/test/e2e/framework"
	e2ekubectl "k8s.io/kubernetes/test/e2e/framework/kubectl"
	e2epod "k8s.io/kubernetes/test/e2e/framework/pod"

	_ "github.com/Azure/aad-pod-managed-identity/test/e2e/webhook"
)

var _ = ginkgo.SynchronizedBeforeSuite(func() []byte {
	c, err := framework.LoadClientset()
	if err != nil {
		framework.Failf("error loading clientset: %v", err)
	}

	// Delete any namespaces except those created by the system. This ensures no
	// lingering resources are left over from a previous test run.
	if framework.TestContext.CleanStart {
		deleted, err := framework.DeleteNamespaces(c, nil, /* deleteFilter */
			[]string{
				metav1.NamespaceSystem,
				metav1.NamespaceDefault,
				metav1.NamespacePublic,
				corev1.NamespaceNodeLease,
			})
		if err != nil {
			framework.Failf("error deleting orphaned namespaces: %v", err)
		}

		if err := framework.WaitForNamespacesDeleted(c, deleted, 5*time.Minute); err != nil {
			framework.Failf("error deleting orphaned namespaces %v: %v", deleted, err)
		}
	}

	// ensure all nodes are schedulable
	framework.ExpectNoError(framework.WaitForAllNodesSchedulable(c, framework.TestContext.NodeSchedulableTimeout))

	// Ensure all pods are running and ready before starting tests
	podStartupTimeout := framework.TestContext.SystemPodsStartupTimeout
	for _, namespace := range []string{metav1.NamespaceSystem, "aad-pi-webhook-system", "cert-manager"} {
		if err := e2epod.WaitForPodsRunningReady(c, namespace, int32(framework.TestContext.MinStartupPods), int32(framework.TestContext.AllowedNotReadyNodes), podStartupTimeout, map[string]string{}); err != nil {
			framework.DumpAllNamespaceInfo(c, namespace)
			e2ekubectl.LogFailedContainers(c, namespace, framework.Logf)
			framework.Failf("error waiting for all pods to be running and ready: %v", err)
		}
	}

	serverVersion, err := c.DiscoveryClient.ServerVersion()
	if err != nil {
		framework.Logf("unexpected server error retrieving version: %v", err)
	}
	if serverVersion != nil {
		framework.Logf("kube-apiserver version: %s", serverVersion.GitVersion)
	}

	return nil
}, func(data []byte) {})

var _ = ginkgo.SynchronizedAfterSuite(func() {
	framework.Logf("Running AfterSuite actions on all node")
	framework.RunCleanupActions()
}, func() {})

// RunE2ETests checks configuration parameters (specified through flags) and then runs
// E2E tests using the Ginkgo runner.
func RunE2ETests(t *testing.T) {
	gomega.RegisterFailHandler(framework.Fail)

	// Run tests through the Ginkgo runner with output to console + JUnit
	var r []ginkgo.Reporter
	if framework.TestContext.ReportDir != "" {
		r = append(r, reporters.NewJUnitReporter(path.Join(framework.TestContext.ReportDir, fmt.Sprintf("junit_%v%02d.xml", framework.TestContext.ReportPrefix, config.GinkgoConfig.ParallelNode))))
	}
	ginkgo.RunSpecsWithDefaultAndCustomReporters(t, "AAD Pod Managed Identity E2E Test Suite", r)
}
