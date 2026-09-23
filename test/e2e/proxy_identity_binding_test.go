package e2e

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/test/e2e/framework"
	e2epod "k8s.io/kubernetes/test/e2e/framework/pod"
)

// AKS identity binding trust must be configured for 'proxy-test-sa' in the
// default namespace ahead of time.
var _ = ginkgo.Describe("Proxy identity bindings [LinuxOnly] [AKSSoakOnly]", func() {
	f := framework.NewDefaultFramework("proxy-identity-binding")

	for _, test := range []struct {
		name            string
		includeClientID bool
	}{
		{name: "with client_id in request", includeClientID: true},
		{name: "with no client_id in request"},
	} {
		ginkgo.It("should get a valid AAD token via AKS identity bindings "+test.name, func(ctx context.Context) {
			clientID, ok := os.LookupEnv("APPLICATION_CLIENT_ID")
			gomega.Expect(ok).To(gomega.BeTrue(), "APPLICATION_CLIENT_ID must be set")
			const namespace = "default"
			serviceAccount := createServiceAccount(f.ClientSet, namespace, "proxy-test-sa", map[string]string{clientIDAnnotation: clientID})
			defer f.ClientSet.CoreV1().ServiceAccounts(namespace).Delete(context.TODO(), serviceAccount, metav1.DeleteOptions{})

			command := "az login -i --allow-no-subscriptions --debug; sleep 3600"
			if test.includeClientID {
				command = fmt.Sprintf("az login -i --client-id %s --allow-no-subscriptions --debug; sleep 3600", clientID)
			}
			pod, err := createPodWithServiceAccount(
				f.ClientSet,
				namespace,
				serviceAccount,
				"mcr.microsoft.com/azure-cli",
				nil,
				[]string{"/bin/sh", "-c", command},
				nil,
				map[string]string{
					injectProxySidecarAnnotation:                   "true",
					proxySidecarPortAnnotation:                     "8080",
					"azure.workload.identity/use-identity-binding": "true",
				},
				map[string]string{useWorkloadIdentityLabel: "true"},
				true,
			)
			framework.ExpectNoError(err, "failed to create pod in %s", namespace)
			defer f.ClientSet.CoreV1().Pods(namespace).Delete(context.TODO(), pod.Name, metav1.DeleteOptions{})

			// output proxy and proxy init logs for debugging
			defer func() {
				for _, container := range []string{"azwi-proxy-init", "azwi-proxy"} {
					stdout, _ := e2epod.GetPodLogs(ctx, f.ClientSet, namespace, pod.Name, container)
					framework.Logf("%s logs: %s", container, stdout)
				}
			}()

			validateProxySideCarInMutatedPod(pod)
			validateProxyIdentityBindingInMutatedPod(pod)

			for _, container := range []string{busybox1, busybox2} {
				framework.Logf("validating that %s in %s has acquired a valid AAD token via the proxy using AKS identity bindings", container, pod.Name)
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
	}
})

func validateProxyIdentityBindingInMutatedPod(pod *corev1.Pod) {
	containers := pod.Spec.Containers
	if useNativeSidecar {
		containers = pod.Spec.InitContainers
	}
	proxySidecar := getProxySidecarContainer(containers)
	gomega.Expect(proxySidecar).NotTo(gomega.BeNil(), "proxy sidecar is not injected to pod %s", pod.Name)

	framework.Logf("validating that the proxy sidecar in %s is configured for AKS identity bindings", pod.Name)
	envVars := map[string]string{}
	for _, env := range proxySidecar.Env {
		envVars[env.Name] = env.Value
	}
	for _, name := range []string{"AZURE_KUBERNETES_TOKEN_PROXY", "AZURE_KUBERNETES_SNI_NAME", "AZURE_KUBERNETES_CA_FILE"} {
		gomega.Expect(envVars).To(gomega.HaveKeyWithValue(name, gomega.Not(gomega.BeEmpty())))
	}

	found := false
	for _, volume := range pod.Spec.Volumes {
		if strings.HasPrefix(volume.Name, projectedVolumeNamePrefix) {
			found = true
			gomega.Expect(volume.Projected).NotTo(gomega.BeNil())
			gomega.Expect(volume.Projected.Sources).To(gomega.Equal(getVolumeProjectionSources(pod.Spec.ServiceAccountName, true)))
			mounted := false
			for _, mount := range proxySidecar.VolumeMounts {
				if mount.Name == volume.Name {
					mounted = true
					gomega.Expect(mount.ReadOnly).To(gomega.BeTrue())
					gomega.Expect(envVars["AZURE_FEDERATED_TOKEN_FILE"]).To(gomega.Equal(filepath.Join(mount.MountPath, tokenFilePath)))
					gomega.Expect(envVars["AZURE_KUBERNETES_CA_FILE"]).To(gomega.Equal(filepath.Join(mount.MountPath, "ca-cert/ca.crt")))
				}
			}
			gomega.Expect(mounted).To(gomega.BeTrue(), "identity binding token and CA volume is not mounted in the proxy sidecar")
		}
	}
	gomega.Expect(found).To(gomega.BeTrue(), "identity binding token and CA volume is not projected to pod %s", pod.Name)
}
