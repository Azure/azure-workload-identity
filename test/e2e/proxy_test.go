//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/Azure/azure-workload-identity/pkg/webhook"
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/test/e2e/framework"
	e2epod "k8s.io/kubernetes/test/e2e/framework/pod"
)

// The proxy implementation is only for Linux.
// Run this test in nightly jobs only because we can't establish federated
// identity under the Microsoft tenant at runtime at the moment.
var _ = ginkgo.Describe("Proxy [LinuxOnly] [AKSSoakOnly] [Exclude:Arc]", func() {
	f := framework.NewDefaultFramework("proxy")

	ginkgo.It("should get a valid AAD token after injecting proxy init container and sidecar", func() {
		clientID, ok := os.LookupEnv("APPLICATION_CLIENT_ID")
		gomega.Expect(ok).To(gomega.BeTrue(), "APPLICATION_CLIENT_ID must be set")
		// trust is only set up for 'proxy-test-sa' service account in the default namespace for now
		const namespace = "default"
		serviceAccount := createServiceAccount(f.ClientSet, namespace, "proxy-test-sa", map[string]string{webhook.UseWorkloadIdentityLabel: "true"}, map[string]string{webhook.ClientIDAnnotation: clientID})
		defer f.ClientSet.CoreV1().ServiceAccounts(namespace).Delete(context.TODO(), serviceAccount, metav1.DeleteOptions{})

		proxyAnnotations := map[string]string{
			webhook.InjectProxySidecarAnnotation: "true",
			webhook.ProxySidecarPortAnnotation:   "8080",
		}

		pod := generatePodWithServiceAccount(
			f.ClientSet,
			namespace,
			serviceAccount,
			"mcr.microsoft.com/azure-cli",
			nil,
			// az login -i reuses the connection, so we need to make sure the token request is made after the proxy sidecar is started
			// otherwise the token request will fail. The sleep 15 is a workaround for the issue.
			// TODO(aramase): remove the sleep after https://github.com/Azure/azure-workload-identity/issues/486 is fixed and v0.12.0 is released.
			[]string{"/bin/sh", "-c", fmt.Sprintf("sleep 15; az login -i -u %s --allow-no-subscriptions --debug; sleep 3600", clientID)},
			nil,
			proxyAnnotations,
			true,
		)

		pod, err := createPod(f.ClientSet, pod)
		framework.ExpectNoError(err, "failed to create pod %s in %s", pod.Name, namespace)
		defer f.ClientSet.CoreV1().Pods(namespace).Delete(context.TODO(), pod.Name, metav1.DeleteOptions{})

		// output proxy and proxy init logs for debugging
		defer func() {
			for _, container := range []string{webhook.ProxyInitContainerName, webhook.ProxySidecarContainerName} {
				stdout, _ := e2epod.GetPodLogs(f.ClientSet, namespace, pod.Name, container)
				framework.Logf("%s logs: %s", container, stdout)
			}
		}()

		for _, container := range []string{busybox1, busybox2} {
			framework.Logf("validating that %s in %s has acquired a valid AAD token via the proxy", container, pod.Name)
			gomega.Eventually(func() bool {
				stdout, err := e2epod.GetPodLogs(f.ClientSet, namespace, pod.Name, container)
				if err != nil {
					framework.Logf("failed to get logs from container %s in %s/%s: %v. Retrying...", container, namespace, pod.Name, err)
					return false
				}
				framework.Logf("stdout: %s", stdout)
				return strings.Contains(stdout, `"environmentName": "AzureCloud"`)
			}, framework.PollShortTimeout, framework.Poll).Should(gomega.BeTrue())
		}
	})
})
