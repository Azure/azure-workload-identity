//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/test/e2e/framework"
	e2epod "k8s.io/kubernetes/test/e2e/framework/pod"
)

// The proxy implementation is only for Linux.
// Run this test in nightly jobs only because we can't establish federated
// identity under the Microsoft tenant at runtime at the moment.
var _ = ginkgo.Describe("Proxy [LinuxOnly] [AKSSoakOnly]", func() {
	f := framework.NewDefaultFramework("proxy")

	ginkgo.It("should get a valid AAD token after injecting proxy init container and sidecar", func(ctx context.Context) {
		clientID, ok := os.LookupEnv("APPLICATION_CLIENT_ID")
		gomega.Expect(ok).To(gomega.BeTrue(), "APPLICATION_CLIENT_ID must be set")
		// trust is only set up for 'proxy-test-sa' service account in the default namespace for now
		const namespace = "default"
		serviceAccount := createServiceAccount(f.ClientSet, namespace, "proxy-test-sa", map[string]string{clientIDAnnotation: clientID})
		defer f.ClientSet.CoreV1().ServiceAccounts(namespace).Delete(context.TODO(), serviceAccount, metav1.DeleteOptions{})

		proxyAnnotations := map[string]string{
			injectProxySidecarAnnotation: "true",
			proxySidecarPortAnnotation:   "8080",
		}

		pod := generatePodWithServiceAccount(
			f.ClientSet,
			namespace,
			serviceAccount,
			"mcr.microsoft.com/azure-cli",
			nil,
			[]string{"/bin/sh", "-c", fmt.Sprintf("az login -i --client-id %s --allow-no-subscriptions --debug; sleep 3600", clientID)},
			nil,
			proxyAnnotations,
			map[string]string{useWorkloadIdentityLabel: "true"},
			true,
		)

		pod, err := createPod(f.ClientSet, pod)
		framework.ExpectNoError(err, "failed to create pod %s in %s", pod.Name, namespace)
		defer f.ClientSet.CoreV1().Pods(namespace).Delete(context.TODO(), pod.Name, metav1.DeleteOptions{})

		// output proxy and proxy init logs for debugging
		defer func() {
			for _, container := range []string{"azwi-proxy-init", "azwi-proxy"} {
				stdout, _ := e2epod.GetPodLogs(ctx, f.ClientSet, namespace, pod.Name, container)
				framework.Logf("%s logs: %s", container, stdout)
			}
		}()

		validateProxySideCarInMutatedPod(pod)

		for _, container := range []string{busybox1, busybox2} {
			framework.Logf("validating that %s in %s has acquired a valid AAD token via the proxy", container, pod.Name)
			gomega.Eventually(func() bool {
				stdout, err := e2epod.GetPodLogs(ctx, f.ClientSet, namespace, pod.Name, container)
				if err != nil {
					framework.Logf("failed to get logs from container %s in %s/%s: %v. Retrying...", container, namespace, pod.Name, err)
					return false
				}
				framework.Logf("stdout: %s", stdout)
				return strings.Contains(stdout, `"environmentName": "AzureCloud"`)
			}, framework.PollShortTimeout, framework.Poll).Should(gomega.BeTrue())
		}
	})

	// This test is to validate the proxy sidecar fallback behavior to AZURE_CLIENT_ID when the client_id parameter is not part of the request.
	ginkgo.It("should get a valid AAD token after injecting proxy init container and sidecar with no client_id in request", func(ctx context.Context) {
		clientID, ok := os.LookupEnv("APPLICATION_CLIENT_ID")
		gomega.Expect(ok).To(gomega.BeTrue(), "APPLICATION_CLIENT_ID must be set")
		// trust is only set up for 'proxy-test-sa' service account in the default namespace for now
		const namespace = "default"
		serviceAccount := createServiceAccount(f.ClientSet, namespace, "proxy-test-sa", map[string]string{clientIDAnnotation: clientID})
		defer f.ClientSet.CoreV1().ServiceAccounts(namespace).Delete(context.TODO(), serviceAccount, metav1.DeleteOptions{})

		proxyAnnotations := map[string]string{
			injectProxySidecarAnnotation: "true",
			proxySidecarPortAnnotation:   "8080",
		}

		pod := generatePodWithServiceAccount(
			f.ClientSet,
			namespace,
			serviceAccount,
			"mcr.microsoft.com/azure-cli",
			nil,
			// no client_id in request
			[]string{"/bin/sh", "-c", "az login -i --allow-no-subscriptions --debug; sleep 3600"},
			nil,
			proxyAnnotations,
			map[string]string{useWorkloadIdentityLabel: "true"},
			true,
		)

		pod, err := createPod(f.ClientSet, pod)
		framework.ExpectNoError(err, "failed to create pod %s in %s", pod.Name, namespace)
		defer f.ClientSet.CoreV1().Pods(namespace).Delete(context.TODO(), pod.Name, metav1.DeleteOptions{})

		// output proxy and proxy init logs for debugging
		defer func() {
			for _, container := range []string{"azwi-proxy-init", "azwi-proxy"} {
				stdout, _ := e2epod.GetPodLogs(ctx, f.ClientSet, namespace, pod.Name, container)
				framework.Logf("%s logs: %s", container, stdout)
			}
		}()

		validateProxySideCarInMutatedPod(pod)

		for _, container := range []string{busybox1, busybox2} {
			framework.Logf("validating that %s in %s has acquired a valid AAD token via the proxy using AZURE_CLIENT_ID", container, pod.Name)
			gomega.Eventually(func() bool {
				stdout, err := e2epod.GetPodLogs(ctx, f.ClientSet, namespace, pod.Name, container)
				if err != nil {
					framework.Logf("failed to get logs from container %s in %s/%s: %v. Retrying...", container, namespace, pod.Name, err)
					return false
				}
				framework.Logf("stdout: %s", stdout)
				/*
					[
					  {
					    "environmentName": "AzureCloud",
					    "id": "72f988bf-86f1-41af-91ab-2d7cd011db47",
					    "isDefault": true,
					    "name": "N/A(tenant level account)",
					    "state": "Enabled",
					    "tenantId": "72f988bf-86f1-41af-91ab-2d7cd011db47",
					    "user": {
					      "assignedIdentityInfo": "MSIClient-3e532d33-08d4-4dea-868c-4c5c4318b6db",
					      "name": "userAssignedIdentity",
					      "type": "servicePrincipal"
					    }
					  }
					]

					// successful response on login will have the above output, so we are asserting that output
					// contains the below string to validate that the login was successful with workload identity
				*/
				return strings.Contains(stdout, `"environmentName": "AzureCloud"`)
			}, framework.PollShortTimeout, framework.Poll).Should(gomega.BeTrue())
		}
	})
})
