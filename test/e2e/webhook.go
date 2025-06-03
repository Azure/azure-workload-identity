//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"strings"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/utils/pointer"
)

const (
	useWorkloadIdentityLabel            = "azure.workload.identity/use"
	clientIDAnnotation                  = "azure.workload.identity/client-id"
	skipContainersAnnotation            = "azure.workload.identity/skip-containers"
	serviceAccountTokenExpiryAnnotation = "azure.workload.identity/service-account-token-expiration"
	injectProxySidecarAnnotation        = "azure.workload.identity/inject-proxy-sidecar"
	proxySidecarPortAnnotation          = "azure.workload.identity/proxy-sidecar-port"
	tokenFilePathName                   = "azure-identity-token"
	tokenFileMountPath                  = "/var/run/secrets/azure/tokens" // #nosec
)

var _ = ginkgo.Describe("Webhook", func() {
	f := framework.NewDefaultFramework("webhook")

	ginkgo.It("should mutate a labeled pod", func(ctx context.Context) {
		serviceAccount := createServiceAccount(f.ClientSet, f.Namespace.Name, f.Namespace.Name+"-sa", map[string]string{clientIDAnnotation: "000-0000-0000-0000"})
		pod, err := createPodWithServiceAccount(
			f.ClientSet,
			f.Namespace.Name,
			serviceAccount,
			"registry.k8s.io/e2e-test-images/busybox:1.29-4",
			[]string{"sleep"},
			[]string{"3600"},
			nil,
			nil,
			map[string]string{useWorkloadIdentityLabel: "true"},
			false,
		)
		framework.ExpectNoError(err, "failed to create pod %s in %s", pod.Name, f.Namespace.Name)
		validateMutatedPod(ctx, f, pod, nil)
	})

	ginkgo.It("should mutate the init containers within a pod", func(ctx context.Context) {
		serviceAccount := createServiceAccount(f.ClientSet, f.Namespace.Name, f.Namespace.Name+"-sa", map[string]string{clientIDAnnotation: "000-0000-0000-0000"})

		pod := generatePodWithServiceAccount(
			f.ClientSet,
			f.Namespace.Name,
			serviceAccount,
			"registry.k8s.io/e2e-test-images/busybox:1.29-4",
			[]string{"sleep"},
			[]string{"3600"},
			nil,
			nil,
			map[string]string{useWorkloadIdentityLabel: "true"},
			false,
		)
		pod.Spec.InitContainers = []corev1.Container{{
			Name:            "init-container",
			Image:           "registry.k8s.io/e2e-test-images/busybox:1.29-4",
			ImagePullPolicy: corev1.PullIfNotPresent,
			Command:         []string{"sleep"},
			Args:            []string{"5"},
			SecurityContext: &corev1.SecurityContext{
				AllowPrivilegeEscalation: pointer.Bool(false),
				Capabilities: &corev1.Capabilities{
					Drop: []corev1.Capability{"ALL"},
				},
				RunAsNonRoot: pointer.Bool(true),
				SeccompProfile: &corev1.SeccompProfile{
					Type: corev1.SeccompProfileTypeRuntimeDefault,
				},
				RunAsUser: pointer.Int64(1000),
			},
		}}
		pod, err := createPod(f.ClientSet, pod)
		framework.ExpectNoError(err, "failed to create pod %s in %s", pod.Name, f.Namespace.Name)
		defer f.ClientSet.CoreV1().Pods(f.Namespace.Name).Delete(context.TODO(), pod.Name, metav1.DeleteOptions{})

		validateMutatedPod(ctx, f, pod, nil)
	})

	ginkgo.It("should mutate a deployment pod with a labeled pod spec", func(ctx context.Context) {
		serviceAccount := createServiceAccount(f.ClientSet, f.Namespace.Name, f.Namespace.Name+"-sa", map[string]string{clientIDAnnotation: "000-0000-0000-0000"})
		pod := createPodUsingDeploymentWithServiceAccount(ctx, f, serviceAccount)
		validateMutatedPod(ctx, f, pod, nil)
	})

	ginkgo.It("should mutate a deployment pod with an annotated service account", func(ctx context.Context) {
		serviceAccount := createServiceAccount(f.ClientSet, f.Namespace.Name, f.Namespace.Name+"-sa", map[string]string{clientIDAnnotation: "000-0000-0000-0000"})
		pod := createPodUsingDeploymentWithServiceAccount(ctx, f, serviceAccount)
		validateMutatedPod(ctx, f, pod, nil)
	})

	ginkgo.It(fmt.Sprintf("should not mutate selected containers if the pod has %s annotated", skipContainersAnnotation), func(ctx context.Context) {
		const skipContainers = busybox1 + ";"
		serviceAccount := createServiceAccount(f.ClientSet, f.Namespace.Name, f.Namespace.Name+"-sa", map[string]string{clientIDAnnotation: "000-0000-0000-0000"})
		pod, err := createPodWithServiceAccount(
			f.ClientSet,
			f.Namespace.Name,
			serviceAccount,
			"registry.k8s.io/e2e-test-images/busybox:1.29-4",
			[]string{"sleep"},
			[]string{"3600"},
			nil,
			map[string]string{skipContainersAnnotation: skipContainers},
			map[string]string{useWorkloadIdentityLabel: "true"},
			false,
		)
		framework.ExpectNoError(err, "failed to create pod %s in %s", pod.Name, f.Namespace.Name)
		validateMutatedPod(ctx, f, pod, strings.Split(skipContainers, ";"))
		validateUnmutatedContainers(f, pod, strings.Split(skipContainers, ";"))
	})

	for _, annotations := range []map[string]string{
		{serviceAccountTokenExpiryAnnotation: "100"},     // less than 3600 (the minimum expiry)
		{serviceAccountTokenExpiryAnnotation: "invalid"}, // non-numeric value
	} {
		ginkgo.It(fmt.Sprintf("should not mutate a pod if '%s: \"%s\"' is annotated to the service account", serviceAccountTokenExpiryAnnotation, annotations[serviceAccountTokenExpiryAnnotation]), func() {
			serviceAccount := createServiceAccount(f.ClientSet, f.Namespace.Name, f.Namespace.Name+"-sa", annotations)
			_, err := createPodWithServiceAccount(
				f.ClientSet,
				f.Namespace.Name,
				serviceAccount,
				"registry.k8s.io/e2e-test-images/busybox:1.29-4",
				[]string{"sleep"},
				[]string{"3600"},
				nil,
				nil,
				map[string]string{useWorkloadIdentityLabel: "true"},
				false,
			)
			framework.Logf("ensuring that the creation of pod is denied by the webhook")
			gomega.ExpectWithOffset(1, err).To(gomega.HaveOccurred(), "creation of pod should be denied by the webhook")
		})

		ginkgo.It(fmt.Sprintf("should not mutate a pod if '%s: \"%s\"' is annotated to the pod", serviceAccountTokenExpiryAnnotation, annotations[serviceAccountTokenExpiryAnnotation]), func() {
			serviceAccount := createServiceAccount(f.ClientSet, f.Namespace.Name, f.Namespace.Name+"-sa", nil)
			_, err := createPodWithServiceAccount(
				f.ClientSet,
				f.Namespace.Name,
				serviceAccount,
				"registry.k8s.io/e2e-test-images/busybox:1.29-4",
				[]string{"sleep"},
				[]string{"3600"},
				nil,
				annotations,
				map[string]string{useWorkloadIdentityLabel: "true"},
				false,
			)
			framework.Logf("ensuring that the creation of pod is denied by the webhook")
			gomega.ExpectWithOffset(1, err).To(gomega.HaveOccurred(), "creation of pod should be denied by the webhook")
		})
	}
})
