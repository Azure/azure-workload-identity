// +build e2e

package e2e

import (
	"context"
	"fmt"
	"strings"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/kubernetes/test/e2e/framework"

	"github.com/Azure/aad-pod-managed-identity/pkg/webhook"
)

var _ = ginkgo.Describe("Webhook", func() {
	f := framework.NewDefaultFramework("webhook")

	ginkgo.It("should mutate a pod with a labeled service account", func() {
		serviceAccount := createServiceAccount(f, map[string]string{webhook.UsePodIdentityLabel: "true"}, nil)
		pod := createPodWithServiceAccount(f, serviceAccount)
		validateMutatedPod(f, pod)
	})
})

func createServiceAccount(f *framework.Framework, labels, annotations map[string]string) string {
	accountName := f.Namespace.Name + "-sa"
	account := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:        accountName,
			Namespace:   f.Namespace.Name,
			Labels:      labels,
			Annotations: annotations,
		},
	}
	_, err := f.ClientSet.CoreV1().ServiceAccounts(f.Namespace.Name).Create(context.TODO(), account, metav1.CreateOptions{})
	framework.ExpectNoError(err, "failed to create service account %s", accountName)
	framework.Logf("created service account %s", accountName)
	return accountName
}

func createPodWithServiceAccount(f *framework.Framework, serviceAccount string) *corev1.Pod {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: f.Namespace.Name + "-",
			Namespace:    f.Namespace.Name,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "busybox",
					Image:   "busybox",
					Command: []string{"sleep"},
					Args:    []string{"3600"},
				},
				{
					Name:    "nginx",
					Image:   "nginx",
					Command: []string{"sleep"},
					Args:    []string{"3600"},
				},
			},
			RestartPolicy:      corev1.RestartPolicyNever,
			ServiceAccountName: serviceAccount,
		},
	}
	createdPod, err := f.ClientSet.CoreV1().Pods(f.Namespace.Name).Create(context.TODO(), pod, metav1.CreateOptions{})
	framework.ExpectNoError(err, "failed to create pod %s", createdPod.Name)
	framework.Logf("created pod %s", createdPod.Name)
	return createdPod
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
}

func getVolumeProjectionSources(f *framework.Framework, serviceAccountName string) []corev1.VolumeProjection {
	expirationSeconds := webhook.DefaultServiceAccountTokenExpiration
	if arcCluster {
		// TODO(chewong): remove this secret creation process once we stopped using fake arc cluster
		secretName := fmt.Sprintf("localtoken-%s", serviceAccountName)
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: f.Namespace.Name,
			},
			Type: corev1.SecretTypeOpaque,
			Data: map[string][]byte{
				"token": []byte("fake token"),
			},
		}
		_, err := f.ClientSet.CoreV1().Secrets(f.Namespace.Name).Create(context.TODO(), secret, metav1.CreateOptions{})
		framework.ExpectNoError(err, "failed to create secret %s", secretName)

		return []corev1.VolumeProjection{{
			Secret: &corev1.SecretProjection{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: secretName,
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
			Audience:          fmt.Sprintf("%s/federatedidentity", strings.TrimRight(azure.PublicCloud.ActiveDirectoryEndpoint, "/")),
		}},
	}
}
