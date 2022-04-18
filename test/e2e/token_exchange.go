//go:build e2e

package e2e

import (
	"context"
	"os"
	"strings"

	"github.com/Azure/azure-workload-identity/pkg/webhook"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/test/e2e/framework"
	e2epod "k8s.io/kubernetes/test/e2e/framework/pod"
)

// Only kind cluster supports custom service account issuer for now.
// Run this test in nightly jobs only because we can't establish federated
// identity under the Microsoft tenant at runtime at the moment.
// This test also can't be run on Arc-enabled clusters because it requires
// the projected service account token to be stored as a Kubernetes secret.
var _ = ginkgo.Describe("TokenExchange [AKSSoakOnly] [Exclude:Arc]", func() {
	f := framework.NewDefaultFramework("token-exchange")

	// E2E scenario from https://github.com/Azure/azure-workload-identity/tree/main/examples/msal-go
	ginkgo.It("should exchange the service account token for a valid AAD token", func() {
		clientID, ok := os.LookupEnv("APPLICATION_CLIENT_ID")
		gomega.Expect(ok).To(gomega.BeTrue(), "APPLICATION_CLIENT_ID must be set")
		keyvaultName, ok := os.LookupEnv("KEYVAULT_NAME")
		gomega.Expect(ok).To(gomega.BeTrue(), "KEYVAULT_NAME must be set")
		keyvaultSecretName, ok := os.LookupEnv("KEYVAULT_SECRET_NAME")
		gomega.Expect(ok).To(gomega.BeTrue(), "KEYVAULT_SECRET_NAME must be set")

		// trust is only set up for 'pod-identity-sa' service account in the default namespace for now
		const namespace = "default"
		serviceAccount := createServiceAccount(f.ClientSet, namespace, "pod-identity-sa", map[string]string{webhook.UseWorkloadIdentityLabel: "true"}, map[string]string{webhook.ClientIDAnnotation: clientID})
		defer f.ClientSet.CoreV1().ServiceAccounts(namespace).Delete(context.TODO(), serviceAccount, metav1.DeleteOptions{})

		pod, err := createPodWithServiceAccount(
			f.ClientSet,
			namespace,
			serviceAccount,
			tokenExchangeE2EImage,
			nil,
			nil,
			[]corev1.EnvVar{{
				Name:  "KEYVAULT_NAME",
				Value: keyvaultName,
			}, {
				Name:  "SECRET_NAME",
				Value: keyvaultSecretName,
			}},
			nil,
		)
		framework.ExpectNoError(err, "failed to create pod %s in %s", pod.Name, namespace)
		defer f.ClientSet.CoreV1().Pods(namespace).Delete(context.TODO(), pod.Name, metav1.DeleteOptions{})

		for _, container := range []string{busybox1, busybox2} {
			framework.Logf("validating that %s in %s has exchanged its service account token for a valid AAD token", container, pod.Name)
			gomega.Eventually(func() bool {
				stdout, err := e2epod.GetPodLogs(f.ClientSet, namespace, pod.Name, container)
				if err != nil {
					framework.Logf("failed to get logs from container %s in %s/%s: %v. Retrying...", container, namespace, pod.Name, err)
					return false
				}
				framework.Logf("stdout: %s", stdout)
				return strings.Contains(stdout, `"successfully got secret" secret="Hello!"`)
			}, framework.PollShortTimeout, framework.Poll).Should(gomega.BeTrue())
		}
	})
})
