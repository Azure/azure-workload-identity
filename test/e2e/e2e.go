//go:build e2e

package e2e

import (
	"context"
	"os"
	"path"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilversion "k8s.io/apimachinery/pkg/util/version"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
	e2edebug "k8s.io/kubernetes/test/e2e/framework/debug"
	e2ekubectl "k8s.io/kubernetes/test/e2e/framework/kubectl"
	e2enode "k8s.io/kubernetes/test/e2e/framework/node"
	e2epod "k8s.io/kubernetes/test/e2e/framework/pod"
)

var (
	tokenExchangeE2EImage string

	c              *kubernetes.Clientset
	coreNamespaces = []string{
		metav1.NamespaceSystem,
		"azure-workload-identity-system",
	}

	useNativeSidecar bool
)

var _ = ginkgo.SynchronizedBeforeSuite(func(ctx context.Context) []byte {
	var err error
	c, err = framework.LoadClientset()
	if err != nil {
		framework.Failf("error loading clientset: %v", err)
	}

	// Delete any namespaces except those created by the system. This ensures no
	// lingering resources are left over from a previous test run.
	if framework.TestContext.CleanStart {
		deleted, err := framework.DeleteNamespaces(ctx, c, nil, /* deleteFilter */
			[]string{
				metav1.NamespaceSystem,
				metav1.NamespaceDefault,
				metav1.NamespacePublic,
				corev1.NamespaceNodeLease,
			})
		if err != nil {
			framework.Failf("error deleting orphaned namespaces: %v", err)
		}

		if err := framework.WaitForNamespacesDeleted(ctx, c, deleted, 5*time.Minute); err != nil {
			framework.Failf("error deleting orphaned namespaces %v: %v", deleted, err)
		}
	}

	timeouts := framework.NewTimeoutContext()
	// ensure all nodes are schedulable
	framework.ExpectNoError(e2enode.WaitForAllNodesSchedulable(ctx, c, timeouts.NodeSchedulable))

	// Ensure all pods are running and ready before starting tests
	podStartupTimeout := timeouts.SystemPodsStartup
	for _, namespace := range coreNamespaces {
		if err := e2epod.WaitForPodsRunningReady(ctx, c, namespace, framework.TestContext.MinStartupPods, podStartupTimeout); err != nil {
			e2edebug.DumpAllNamespaceInfo(ctx, c, namespace)
			e2ekubectl.LogFailedContainers(ctx, c, namespace, framework.Logf)
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
	sv, err := utilversion.ParseSemantic(serverVersion.GitVersion)
	if err != nil {
		framework.Failf("unexpected server error parsing version: %v", err)
	}
	// "SidecarContainers" went beta in 1.29. With the 3 version skew policy,
	// between API server and kubelet, 1.32 is the earliest version this can be
	// safely used.
	useNativeSidecar = sv.AtLeast(utilversion.MajorMinor(1, 32))
	framework.Logf("proxy should use native sidecar: %t", useNativeSidecar)
	return nil
}, func(data []byte) {})

var _ = ginkgo.SynchronizedAfterSuite(func(ctx context.Context) {
	framework.Logf("Running AfterSuite actions on all node")
}, func(ctx context.Context) {
	collectPodLogs(ctx)
})

func collectPodLogs(ctx context.Context) {
	var wg sync.WaitGroup
	var since time.Time
	if os.Getenv("SOAK_CLUSTER") == "true" {
		// get logs for the last 24h since e2e is run against soak clusters every 24h
		since = time.Now().Add(-24 * time.Hour)
	}

	for _, namespace := range coreNamespaces {
		pods, err := c.CoreV1().Pods(namespace).List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			framework.Logf("failed to list pods from %s: %v", namespace, err)
			continue
		}

		for _, pod := range pods.Items {
			for _, container := range pod.Spec.Containers {
				wg.Add(1)
				go func(namespace string, pod corev1.Pod, container corev1.Container) {
					defer ginkgo.GinkgoRecover()
					defer wg.Done()

					framework.Logf("fetching logs from pod %s/%s, container %s", namespace, pod.Name, container.Name)

					logFile := path.Join(framework.TestContext.ReportDir, namespace, pod.Name, container.Name+".log")
					gomega.Expect(os.MkdirAll(filepath.Dir(logFile), 0755)).To(gomega.Succeed())

					f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
					if err != nil {
						// Failing to fetch logs should not cause the test to fail
						framework.Logf("error opening file to write pod logs: %v", err)
						return
					}
					defer f.Close()

					var log string
					if since.IsZero() {
						log, err = e2epod.GetPodLogs(ctx, c, namespace, pod.Name, container.Name)
					} else {
						log, err = e2epod.GetPodLogsSince(ctx, c, namespace, pod.Name, container.Name, since)
					}
					if err != nil {
						framework.Logf("error when getting logs from pod %s/%s, container %s: %v", namespace, pod.Name, container.Name, err)
						return
					}

					_, err = f.Write([]byte(log))
					if err != nil {
						framework.Logf("error when writing logs to %s: %v", logFile, err)
					}
				}(namespace, pod, container)
			}
		}
	}
	wg.Wait()
}

// RunE2ETests checks configuration parameters (specified through flags) and then runs
// E2E tests using the Ginkgo runner.
func RunE2ETests(t *testing.T) {
	gomega.RegisterFailHandler(framework.Fail)

	// NOTE: junit report can be simply created by executing your tests with the new --junit-report flags instead.
	if err := os.MkdirAll(framework.TestContext.ReportDir, 0755); err != nil {
		framework.Failf("failed creating report directory: %v", err)
	}
	suiteConfig, reporterConfig := framework.CreateGinkgoConfig()
	ginkgo.RunSpecs(t, "Azure AD Workload Identity E2E Test Suite", suiteConfig, reporterConfig)
}
