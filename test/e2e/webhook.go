// +build e2e

package e2e

import (
	"fmt"
	"strings"

	"github.com/Azure/azure-workload-identity/pkg/webhook"

	"github.com/onsi/ginkgo"
	"k8s.io/kubernetes/test/e2e/framework"
)

var _ = ginkgo.Describe("Webhook", func() {
	f := framework.NewDefaultFramework("webhook")

	ginkgo.It("should mutate a pod with a labeled service account", func() {
		serviceAccount := createServiceAccount(f.ClientSet, f.Namespace.Name, f.Namespace.Name+"-sa", map[string]string{webhook.UsePodIdentityLabel: "true"}, nil)
		pod, err := createPodWithServiceAccount(
			f.ClientSet,
			f.Namespace.Name,
			serviceAccount,
			"k8s.gcr.io/e2e-test-images/busybox:1.29-1",
			[]string{"sleep"},
			[]string{"3600"},
			nil,
			nil,
		)
		framework.ExpectNoError(err, "failed to create pod %s in %s", pod.Name, f.Namespace.Name)
		validateMutatedPod(f, pod, nil)
	})

	ginkgo.It("should mutate a deployment pod with a labeled service account", func() {
		serviceAccount := createServiceAccount(f.ClientSet, f.Namespace.Name, f.Namespace.Name+"-sa", map[string]string{webhook.UsePodIdentityLabel: "true"}, nil)
		pod := createPodUsingDeploymentWithServiceAccount(f, serviceAccount)
		validateMutatedPod(f, pod, nil)
	})

	ginkgo.It(fmt.Sprintf("should not mutate selected containers if the pod has %s annotated", webhook.SkipContainersAnnotation), func() {
		const skipContainers = busybox1 + ";"
		serviceAccount := createServiceAccount(f.ClientSet, f.Namespace.Name, f.Namespace.Name+"-sa", map[string]string{webhook.UsePodIdentityLabel: "true"}, nil)
		pod, err := createPodWithServiceAccount(
			f.ClientSet,
			f.Namespace.Name,
			serviceAccount,
			"k8s.gcr.io/e2e-test-images/busybox:1.29-1",
			[]string{"sleep"},
			[]string{"3600"},
			nil,
			map[string]string{webhook.SkipContainersAnnotation: skipContainers},
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
			serviceAccount := createServiceAccount(f.ClientSet, f.Namespace.Name, f.Namespace.Name+"-sa", map[string]string{webhook.UsePodIdentityLabel: "true"}, annotations)
			_, err := createPodWithServiceAccount(
				f.ClientSet,
				f.Namespace.Name,
				serviceAccount,
				"k8s.gcr.io/e2e-test-images/busybox:1.29-1",
				[]string{"sleep"},
				[]string{"3600"},
				nil,
				nil,
			)
			framework.Logf("ensuring that the creation of pod is denied by the webhook")
			framework.ExpectError(err, "creation of pod should be denied by the webhook")
		})

		ginkgo.It(fmt.Sprintf("should not mutate a pod if '%s: \"%s\"' is annotated to the pod", webhook.ServiceAccountTokenExpiryAnnotation, annotations[webhook.ServiceAccountTokenExpiryAnnotation]), func() {
			serviceAccount := createServiceAccount(f.ClientSet, f.Namespace.Name, f.Namespace.Name+"-sa", map[string]string{webhook.UsePodIdentityLabel: "true"}, nil)
			_, err := createPodWithServiceAccount(
				f.ClientSet,
				f.Namespace.Name,
				serviceAccount,
				"k8s.gcr.io/e2e-test-images/busybox:1.29-1",
				[]string{"sleep"},
				[]string{"3600"},
				nil,
				annotations,
			)
			framework.Logf("ensuring that the creation of pod is denied by the webhook")
			framework.ExpectError(err, "creation of pod should be denied by the webhook")
		})
	}
})
