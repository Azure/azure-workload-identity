//go:build e2e

package e2e

import (
	"context"
	"os"
	"strings"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/test/e2e/framework"
	e2epod "k8s.io/kubernetes/test/e2e/framework/pod"
)

// Only kind cluster supports custom service account issuer for now.
var _ = ginkgo.Describe("TokenExchange [KindOnly]", func() {
	f := framework.NewDefaultFramework("token-exchange")

	// E2E scenario from https://github.com/Azure/azure-workload-identity/tree/main/examples/msal-go
	ginkgo.FIt("should exchange the service account token for a valid AAD token", func() {
		applicationName, ok := os.LookupEnv("APPLICATION_NAME")
		gomega.Expect(ok, gomega.BeTrue(), "APPLICATION_NAME must be set")
		serviceAccountIsser, ok := os.LookupEnv("SERVICE_ACCOUNT_ISSUER")
		gomega.Expect(ok, gomega.BeTrue(), "SERVICE_ACCOUNT_ISSUER must be set")
		keyvaultName, ok := os.LookupEnv("KEYVAULT_NAME")
		gomega.Expect(ok).To(gomega.BeTrue(), "KEYVAULT_NAME must be set")
		keyvaultSecretName, ok := os.LookupEnv("KEYVAULT_SECRET_NAME")
		gomega.Expect(ok).To(gomega.BeTrue(), "KEYVAULT_SECRET_NAME must be set")

		// create a service account and federated identity
		serviceAccount := f.Namespace.Name + "-sa"
		err := runAzwiSerivceAccount("create",
			"--aad-application-name", applicationName,
			"--service-account-namespace", f.Namespace.Name,
			"--service-account-name", serviceAccount,
			"--service-account-issuer-url", serviceAccountIsser,
			"--skip-phases", "aad-application,role-assignment",
		)
		framework.ExpectNoError(err, "failed to create service account and federated identity")

		pod, err := createPodWithServiceAccount(
			f.ClientSet,
			f.Namespace.Name,
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
		framework.ExpectNoError(err, "failed to create pod %s in %s", pod.Name, f.Namespace.Name)
		defer f.ClientSet.CoreV1().Pods(f.Namespace.Name).Delete(context.TODO(), pod.Name, metav1.DeleteOptions{})

		for _, container := range []string{busybox1, busybox2} {
			framework.Logf("validating that %s in %s has exchanged its service account token for a valid AAD token", container, pod.Name)
			gomega.Eventually(func() bool {
				stdout, err := e2epod.GetPodLogs(f.ClientSet, f.Namespace.Name, pod.Name, container)
				if err != nil {
					framework.Logf("failed to get logs from container %s in %s/%s: %v. Retrying...", container, f.Namespace.Name, pod.Name, err)
					return false
				}
				framework.Logf("stdout: %s", stdout)
				return strings.Contains(stdout, `"successfully got secret" secret="Hello!"`)
			}, framework.PollShortTimeout, framework.Poll).Should(gomega.BeTrue())
		}
	})
})
