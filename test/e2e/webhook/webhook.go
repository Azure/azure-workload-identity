package webhook

import (
	"github.com/onsi/ginkgo"
	clientset "k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
	e2epod "k8s.io/kubernetes/test/e2e/framework/pod"
)

var _ = ginkgo.Describe("Webhook", func() {
	f := framework.NewDefaultFramework("webhook")

	var (
		c clientset.Interface
	)

	ginkgo.BeforeEach(func() {
		c = f.ClientSet
	})

	ginkgo.It("list pods", func() {
		pods, err := e2epod.GetPodsInNamespace(c, "kube-system", nil)
		framework.ExpectNoError(err)
		framework.Logf("Number of pods in kube-system: %d", len(pods))
	})
})
