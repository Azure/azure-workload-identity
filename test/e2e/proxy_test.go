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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/test/e2e/framework"
	e2epod "k8s.io/kubernetes/test/e2e/framework/pod"
)

// The proxy implementation is only for Linux.
// Run this test in nightly jobs only because we can't establish federated
// identity under the Microsoft tenant at runtime at the moment.
var _ = ginkgo.Describe("Proxy [LinuxOnly] [AKSSoakOnly] [Exclude:Arc]", func() {
	f := framework.NewDefaultFramework("proxy")

	ginkgo.It("should get a valid AAD token with the proxy sidecar", func() {
		clientID, ok := os.LookupEnv("APPLICATION_CLIENT_ID")
		gomega.Expect(ok).To(gomega.BeTrue(), "APPLICATION_CLIENT_ID must be set")
		// trust is only set up for 'proxy-test-sa' service account in the default namespace for now
		const namespace = "default"
		serviceAccount := createServiceAccount(f.ClientSet, namespace, "proxy-test-sa", map[string]string{webhook.UseWorkloadIdentityLabel: "true"}, map[string]string{webhook.ClientIDAnnotation: clientID})
		defer f.ClientSet.CoreV1().ServiceAccounts(namespace).Delete(context.TODO(), serviceAccount, metav1.DeleteOptions{})

		pod := generatePodWithServiceAccount(
			f.ClientSet,
			namespace,
			serviceAccount,
			"mcr.microsoft.com/azure-cli",
			nil,
			[]string{"/bin/sh", "-c", fmt.Sprintf("az login -i -u %s --allow-no-subscriptions --debug; sleep 3600", clientID)},
			nil,
			nil,
		)

		trueVal := true
		// proxy-init needs to be run as root
		runAsRoot := int64(0)
		pod.Spec.InitContainers = []corev1.Container{
			{
				Name:            proxyInit,
				Image:           proxyInitImage,
				ImagePullPolicy: corev1.PullIfNotPresent,
				SecurityContext: &corev1.SecurityContext{
					Privileged: &trueVal,
					RunAsUser:  &runAsRoot,
					Capabilities: &corev1.Capabilities{
						Add:  []corev1.Capability{"NET_ADMIN"},
						Drop: []corev1.Capability{"ALL"},
					},
				},
				Env: []corev1.EnvVar{
					{
						Name:  "PROXY_PORT",
						Value: "8000",
					},
				},
			},
		}

		pod.Spec.Containers = append(pod.Spec.Containers,
			[]corev1.Container{
				{
					Name:            proxy,
					Image:           proxyImage,
					ImagePullPolicy: corev1.PullIfNotPresent,
					Args:            []string{"--log-encoder=json"},
					Ports: []corev1.ContainerPort{
						{
							Name:          "http",
							ContainerPort: 8000,
						},
					},
				},
			}...)

		pod, err := createPod(f.ClientSet, pod)
		framework.ExpectNoError(err, "failed to create pod %s in %s", pod.Name, namespace)
		defer f.ClientSet.CoreV1().Pods(namespace).Delete(context.TODO(), pod.Name, metav1.DeleteOptions{})

		// output proxy and proxy init logs for debugging
		defer func() {
			for _, container := range []string{proxy, proxyInit} {
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
