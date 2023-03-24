//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"strings"

	"github.com/onsi/ginkgo/v2"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/test/e2e/framework"
	"k8s.io/utils/pointer"

	"github.com/Azure/azure-workload-identity/pkg/webhook"
)

var _ = ginkgo.Describe("Webhook", func() {
	f := framework.NewDefaultFramework("webhook")

	ginkgo.It("should mutate a labeled pod", func() {
		serviceAccount := createServiceAccount(f.ClientSet, f.Namespace.Name, f.Namespace.Name+"-sa", nil, nil)
		pod, err := createPodWithServiceAccount(
			f.ClientSet,
			f.Namespace.Name,
			serviceAccount,
			"registry.k8s.io/e2e-test-images/busybox:1.29-4",
			[]string{"sleep"},
			[]string{"3600"},
			nil,
			nil,
			map[string]string{webhook.UseWorkloadIdentityLabel: "true"},
			false,
		)
		framework.ExpectNoError(err, "failed to create pod %s in %s", pod.Name, f.Namespace.Name)
		validateMutatedPod(f, pod, nil)
	})

	ginkgo.It("should mutate the init containers within a pod", func() {
		serviceAccount := createServiceAccount(f.ClientSet, f.Namespace.Name, f.Namespace.Name+"-sa", map[string]string{webhook.UseWorkloadIdentityLabel: "true"}, nil)

		pod := generatePodWithServiceAccount(
			f.ClientSet,
			f.Namespace.Name,
			serviceAccount,
			"registry.k8s.io/e2e-test-images/busybox:1.29-4",
			[]string{"sleep"},
			[]string{"3600"},
			nil,
			nil,
			map[string]string{webhook.UseWorkloadIdentityLabel: "true"},
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

		validateMutatedPod(f, pod, nil)
	})

	ginkgo.It("should mutate a deployment pod with a labeled pod spec", func() {
		serviceAccount := createServiceAccount(f.ClientSet, f.Namespace.Name, f.Namespace.Name+"-sa", map[string]string{webhook.UseWorkloadIdentityLabel: "true"}, nil)
		pod := createPodUsingDeploymentWithServiceAccount(f, serviceAccount)
		validateMutatedPod(f, pod, nil)
	})

	ginkgo.It("should mutate a deployment pod with an annotated service account", func() {
		serviceAccount := createServiceAccount(f.ClientSet, f.Namespace.Name, f.Namespace.Name+"-sa", nil, map[string]string{webhook.UseWorkloadIdentityLabel: "true"})
		pod := createPodUsingDeploymentWithServiceAccount(f, serviceAccount)
		validateMutatedPod(f, pod, nil)
	})

	ginkgo.It(fmt.Sprintf("should not mutate selected containers if the pod has %s annotated", webhook.SkipContainersAnnotation), func() {
		const skipContainers = busybox1 + ";"
		serviceAccount := createServiceAccount(f.ClientSet, f.Namespace.Name, f.Namespace.Name+"-sa", map[string]string{webhook.UseWorkloadIdentityLabel: "true"}, nil)
		pod, err := createPodWithServiceAccount(
			f.ClientSet,
			f.Namespace.Name,
			serviceAccount,
			"registry.k8s.io/e2e-test-images/busybox:1.29-4",
			[]string{"sleep"},
			[]string{"3600"},
			nil,
			map[string]string{webhook.SkipContainersAnnotation: skipContainers},
			map[string]string{webhook.UseWorkloadIdentityLabel: "true"},
			false,
		)
		framework.ExpectNoError(err, "failed to create pod %s in %s", pod.Name, f.Namespace.Name)
		validateMutatedPod(f, pod, strings.Split(skipContainers, ";"))
		validateUnmutatedContainers(f, pod, strings.Split(skipContainers, ";"))
	})

	for _, annotations := range []map[string]string{
		{webhook.ServiceAccountTokenExpiryAnnotation: "100"},     // less than 3600 (the minimum expiry)
		{webhook.ServiceAccountTokenExpiryAnnotation: "invalid"}, // non-numeric value
	} {
		ginkgo.It(fmt.Sprintf("should not mutate a pod if '%s: \"%s\"' is annotated to the service account", webhook.ServiceAccountTokenExpiryAnnotation, annotations[webhook.ServiceAccountTokenExpiryAnnotation]), func() {
			serviceAccount := createServiceAccount(f.ClientSet, f.Namespace.Name, f.Namespace.Name+"-sa", map[string]string{webhook.UseWorkloadIdentityLabel: "true"}, annotations)
			_, err := createPodWithServiceAccount(
				f.ClientSet,
				f.Namespace.Name,
				serviceAccount,
				"registry.k8s.io/e2e-test-images/busybox:1.29-4",
				[]string{"sleep"},
				[]string{"3600"},
				nil,
				nil,
				map[string]string{webhook.UseWorkloadIdentityLabel: "true"},
				false,
			)
			framework.Logf("ensuring that the creation of pod is denied by the webhook")
			framework.ExpectError(err, "creation of pod should be denied by the webhook")
		})

		ginkgo.It(fmt.Sprintf("should not mutate a pod if '%s: \"%s\"' is annotated to the pod", webhook.ServiceAccountTokenExpiryAnnotation, annotations[webhook.ServiceAccountTokenExpiryAnnotation]), func() {
			serviceAccount := createServiceAccount(f.ClientSet, f.Namespace.Name, f.Namespace.Name+"-sa", map[string]string{webhook.UseWorkloadIdentityLabel: "true"}, nil)
			_, err := createPodWithServiceAccount(
				f.ClientSet,
				f.Namespace.Name,
				serviceAccount,
				"registry.k8s.io/e2e-test-images/busybox:1.29-4",
				[]string{"sleep"},
				[]string{"3600"},
				nil,
				annotations,
				map[string]string{webhook.UseWorkloadIdentityLabel: "true"},
				false,
			)
			framework.Logf("ensuring that the creation of pod is denied by the webhook")
			framework.ExpectError(err, "creation of pod should be denied by the webhook")
		})
	}
})
