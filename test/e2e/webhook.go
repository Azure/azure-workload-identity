// +build e2e

package e2e

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
	e2edeploy "k8s.io/kubernetes/test/e2e/framework/deployment"
	e2epod "k8s.io/kubernetes/test/e2e/framework/pod"

	"github.com/Azure/aad-pod-managed-identity/pkg/webhook"
)

var _ = ginkgo.Describe("Webhook", func() {
	f := framework.NewDefaultFramework("webhook")

	ginkgo.It("should mutate a pod with a labeled service account", func() {
		serviceAccount := createServiceAccount(f.ClientSet, f.Namespace.Name, f.Namespace.Name+"-sa", map[string]string{webhook.UsePodIdentityLabel: "true"}, nil)
		pod := createPodWithServiceAccount(f.ClientSet, f.Namespace.Name, serviceAccount, "k8s.gcr.io/e2e-test-images/busybox:1.29-1", []string{"sleep"}, []string{"3600"}, nil)
		validateMutatedPod(f, pod)
	})

	ginkgo.It("should mutate a deployment pod with a labeled service account", func() {
		serviceAccount := createServiceAccount(f.ClientSet, f.Namespace.Name, f.Namespace.Name+"-sa", map[string]string{webhook.UsePodIdentityLabel: "true"}, nil)
		pod := createPodUsingDeploymentWithServiceAccount(f, serviceAccount)
		validateMutatedPod(f, pod)
	})

	// E2E scenario from https://github.com/Azure/aad-pod-managed-identity/tree/main/examples/msal-net/akvdotnet
	ginkgo.It("should exchange the service account token for an valid AAD token [KindOnly]", func() {
		// trust is only set up for 'pod-identity-sa' service account in the default namespace for now
		const namespace = "default"
		serviceAccount := createServiceAccount(f.ClientSet, namespace, "pod-identity-sa", map[string]string{webhook.UsePodIdentityLabel: "true"}, map[string]string{webhook.ClientIDAnnotation: os.Getenv("SERVICE_PRINCIPAL_CLIENT_ID")})
		defer f.ClientSet.CoreV1().ServiceAccounts(namespace).Delete(context.TODO(), serviceAccount, metav1.DeleteOptions{})

		keyvaultName, ok := os.LookupEnv("KEYVAULT_NAME")
		gomega.Expect(ok).To(gomega.BeTrue(), "KEYVAULT_NAME must be set")
		keyvaultSecretName, ok := os.LookupEnv("KEYVAULT_SECRET_NAME")
		gomega.Expect(ok).To(gomega.BeTrue(), "KEYVAULT_SECRET_NAME must be set")

		pod := createPodWithServiceAccount(f.ClientSet, namespace, serviceAccount, "aramase/dotnet:v0.4", nil, nil, []corev1.EnvVar{{
			Name:  "KEYVAULT_NAME",
			Value: keyvaultName,
		}, {
			Name:  "SECRET_NAME",
			Value: keyvaultSecretName,
		}})
		defer f.ClientSet.CoreV1().Pods(namespace).Delete(context.TODO(), pod.Name, metav1.DeleteOptions{})

		ginkgo.By("validating that the application has exchanged its service account token for an valid AAD token")
		time.Sleep(3 * time.Second)
		stdout, err := e2epod.GetPodLogs(f.ClientSet, namespace, pod.Name, pod.Spec.Containers[0].Name)
		gomega.Expect(err).To(gomega.BeNil())

		framework.Logf("stdout: %s", stdout)
		gomega.Expect(strings.Contains(stdout, "Your secret is Hello!")).To(gomega.BeTrue())
	})
})

func createServiceAccount(c kubernetes.Interface, namespace, name string, labels, annotations map[string]string) string {
	account := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Labels:      labels,
			Annotations: annotations,
		},
	}
	_, err := c.CoreV1().ServiceAccounts(namespace).Create(context.TODO(), account, metav1.CreateOptions{})
	framework.ExpectNoError(err, "failed to create service account %s", name)
	framework.Logf("created service account %s", name)
	return name
}

func createPodWithServiceAccount(c kubernetes.Interface, namespace, serviceAccount, image string, command, args []string, env []corev1.EnvVar) *corev1.Pod {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: namespace + "-",
			Namespace:    namespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{
				Name: namespace + "-test-container",
				// this image supports both Linux and Windows
				Image:   image,
				Command: command,
				Args:    args,
				Env:     env,
			}},
			RestartPolicy:      corev1.RestartPolicyNever,
			ServiceAccountName: serviceAccount,
		},
	}

	if framework.NodeOSDistroIs("windows") {
		e2epod.SetNodeSelection(&pod.Spec, e2epod.NodeSelection{
			Selector: map[string]string{
				"kubernetes.io/os": "windows",
			},
		})
	}

	if arcCluster {
		createSecretForArcCluster(c, namespace, serviceAccount)
	}

	pod, err := c.CoreV1().Pods(namespace).Create(context.TODO(), pod, metav1.CreateOptions{})
	framework.ExpectNoError(err, "failed to create pod %s", pod.Name)

	err = e2epod.WaitForPodNameRunningInNamespace(c, pod.Name, pod.Namespace)
	framework.ExpectNoError(err, "failed to start pod %s", pod.Name)

	framework.Logf("created pod %s", pod.Name)
	return pod
}

func createPodUsingDeploymentWithServiceAccount(f *framework.Framework, serviceAccount string) *corev1.Pod {
	replicas := int32(1)
	zero := int64(0)
	podLabels := map[string]string{"app": "busybox"}

	d := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: f.Namespace.Name + "-",
			Namespace:    f.Namespace.Name,
			Labels:       podLabels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{MatchLabels: podLabels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: podLabels,
				},
				Spec: corev1.PodSpec{
					TerminationGracePeriodSeconds: &zero,
					Containers: []corev1.Container{
						{
							Name: "busybox",
							// this image supports both Linux and Windows
							Image:   "k8s.gcr.io/e2e-test-images/busybox:1.29-1",
							Command: []string{"sleep"},
							Args:    []string{"3600"},
						},
					},
					ServiceAccountName: serviceAccount,
				},
			},
		},
	}

	if framework.NodeOSDistroIs("windows") {
		e2epod.SetNodeSelection(&d.Spec.Template.Spec, e2epod.NodeSelection{
			Selector: map[string]string{
				"kubernetes.io/os": "windows",
			},
		})
	}

	if arcCluster {
		createSecretForArcCluster(f.ClientSet, f.Namespace.Name, serviceAccount)
	}

	d, err := f.ClientSet.AppsV1().Deployments(f.Namespace.Name).Create(context.TODO(), d, metav1.CreateOptions{})
	framework.ExpectNoError(err, "failed to create deployment %s", d.Name)

	err = e2edeploy.WaitForDeploymentComplete(f.ClientSet, d)
	framework.ExpectNoError(err, "failed to complete deployment %s", d.Name)

	podList, err := e2edeploy.GetPodsForDeployment(f.ClientSet, d)
	framework.ExpectNoError(err, "failed to get pods for deployment %s", d.Name)
	pod := &podList.Items[0]

	framework.Logf("created pod %s with deployment %s", pod.Name, d.Name)
	return pod
}

func validateMutatedPod(f *framework.Framework, pod *corev1.Pod) {
	for _, container := range pod.Spec.Containers {
		m := make(map[string]struct{})
		for _, env := range container.Env {
			m[env.Name] = struct{}{}
		}

		framework.Logf("ensuring that the correct environment variables are injected to %s in %s", container.Name, pod.Name)
		for _, injected := range []string{
			webhook.AzureClientIDEnvVar,
			webhook.AzureTenantIDEnvVar,
			webhook.TokenFilePathEnvVar,
			webhook.AzureAuthorityHostEnvVar,
		} {
			if _, ok := m[injected]; !ok {
				framework.Failf("container %s in pod %s does not have env var %s injected", container.Name, pod.Name, injected)
			}
		}

		framework.Logf("ensuring that azure-identity-token is mounted to %s", container.Name)
		found := false
		for _, volumeMount := range container.VolumeMounts {
			if volumeMount.Name == "azure-identity-token" {
				found = true
				gomega.Expect(volumeMount).To(gomega.Equal(corev1.VolumeMount{
					Name:      webhook.TokenFilePathName,
					MountPath: webhook.TokenFileMountPath,
					ReadOnly:  true,
				}))
				break
			}
		}
		if !found {
			framework.Failf("container %s in pod %s does not have azure-identity-token volume mount", container.Name, pod.Name)
		}
	}

	framework.Logf("ensuring that the token volume is projected to %s as azure-identity-token", pod.Name)
	defaultMode := int32(420)
	found := false
	for _, volume := range pod.Spec.Volumes {
		if volume.Name == webhook.TokenFilePathName {
			found = true
			gomega.Expect(volume).To(gomega.Equal(corev1.Volume{
				Name: webhook.TokenFilePathName,
				VolumeSource: corev1.VolumeSource{
					Projected: &corev1.ProjectedVolumeSource{
						Sources:     getVolumeProjectionSources(f, pod.Spec.ServiceAccountName),
						DefaultMode: &defaultMode,
					},
				},
			}))
			break
		}
	}
	if !found {
		framework.Failf("pod %s does not have azure-identity-token as a projected token volume", pod.Name)
	}

	_ = f.ExecCommandInContainer(pod.Name, pod.Spec.Containers[0].Name, "cat", filepath.Join(webhook.TokenFileMountPath, webhook.TokenFilePathName))
}

func getVolumeProjectionSources(f *framework.Framework, serviceAccountName string) []corev1.VolumeProjection {
	expirationSeconds := webhook.DefaultServiceAccountTokenExpiration
	if arcCluster {
		return []corev1.VolumeProjection{{
			Secret: &corev1.SecretProjection{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: fmt.Sprintf("localtoken-%s", serviceAccountName),
				},
				Items: []corev1.KeyToPath{
					{
						Key:  "token",
						Path: webhook.TokenFilePathName,
					},
				},
			},
		}}
	}
	return []corev1.VolumeProjection{{
		ServiceAccountToken: &corev1.ServiceAccountTokenProjection{
			Path:              webhook.TokenFilePathName,
			ExpirationSeconds: &expirationSeconds,
			Audience:          webhook.DefaultAudience,
		}},
	}
}

func createSecretForArcCluster(c kubernetes.Interface, namespace, serviceAccount string) {
	// TODO(chewong): remove this secret creation process once we stopped using fake arc cluster
	secretName := fmt.Sprintf("localtoken-%s", serviceAccount)
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{
			"token": []byte("fake token"),
		},
	}
	_, err := c.CoreV1().Secrets(namespace).Create(context.TODO(), secret, metav1.CreateOptions{})
	framework.ExpectNoError(err, "failed to create secret %s", secretName)
}
