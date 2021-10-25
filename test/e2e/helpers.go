//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/Azure/azure-workload-identity/pkg/webhook"

	"github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
	e2edeploy "k8s.io/kubernetes/test/e2e/framework/deployment"
	e2epod "k8s.io/kubernetes/test/e2e/framework/pod"
)

const (
	busybox1  = "busybox-1"
	busybox2  = "busybox-2"
	proxyInit = "proxy-init"
	proxy     = "proxy"
)

// createServiceAccount creates a service account with customizable name, namespace, labels and annotations.
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

	// make sure the service account is created
	// ref: https://github.com/Azure/azure-workload-identity/issues/114
	gomega.Eventually(func() bool {
		_, err := c.CoreV1().ServiceAccounts(namespace).Get(context.TODO(), name, metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			framework.Logf("service account %s/%s is not found", namespace, name)
		}
		return err == nil
	}, framework.PollShortTimeout, framework.Poll).Should(gomega.BeTrue())

	framework.Logf("created service account %s", name)
	return name
}

// createPodWithServiceAccount creates a pod with two containers, busybox-1 and busybox-2 with customizable
// namespace, service account, image, command, arguments, environment variables, and annotations.
func createPodWithServiceAccount(c kubernetes.Interface, namespace, serviceAccount, image string, command, args []string, env []corev1.EnvVar, annotations map[string]string) (*corev1.Pod, error) {
	if arcCluster {
		createSecretForArcCluster(c, namespace, serviceAccount)
	}

	pod := generatePodWithServiceAccount(c, namespace, serviceAccount, image, command, args, env, annotations)
	return createPod(c, pod)
}

// generatePodWithServiceAccount generates a pod with two containers, busybox-1 and busybox-2 with customizable
// namespace, service account, image, command, arguments, environment variables, and annotations.
func generatePodWithServiceAccount(c kubernetes.Interface, namespace, serviceAccount, image string, command, args []string, env []corev1.EnvVar, annotations map[string]string) *corev1.Pod {
	zero := int64(0)
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: namespace + "-",
			Namespace:    namespace,
			Annotations:  annotations,
		},
		Spec: corev1.PodSpec{
			TerminationGracePeriodSeconds: &zero,
			Containers: []corev1.Container{{
				Name:            busybox1,
				Image:           image, // this image should support both Linux and Windows
				Command:         command,
				Args:            args,
				Env:             env,
				ImagePullPolicy: corev1.PullIfNotPresent,
			}, {
				Name:            busybox2,
				Image:           image, // this image should support both Linux and Windows
				Command:         command,
				Args:            args,
				Env:             env,
				ImagePullPolicy: corev1.PullIfNotPresent,
			}},
			RestartPolicy:      corev1.RestartPolicyNever,
			ServiceAccountName: serviceAccount,
		},
	}

	nodeOSDistro := "linux"
	if framework.NodeOSDistroIs("windows") {
		nodeOSDistro = "windows"
	}
	e2epod.SetNodeSelection(&pod.Spec, e2epod.NodeSelection{
		Selector: map[string]string{
			"kubernetes.io/os": nodeOSDistro,
		},
	})

	return pod
}

// createPod creates the given pod
func createPod(c kubernetes.Interface, pod *corev1.Pod) (*corev1.Pod, error) {
	framework.Logf("creating a pod in %s namespace with service account %s", pod.Namespace, pod.Spec.ServiceAccountName)
	return c.CoreV1().Pods(pod.Namespace).Create(context.TODO(), pod, metav1.CreateOptions{})
}

// createPodUsingDeploymentWithServiceAccount creates a deployment containing one pod with customizable service account.
func createPodUsingDeploymentWithServiceAccount(f *framework.Framework, serviceAccount string) *corev1.Pod {
	framework.Logf("creating a deployment in %s namespace with service account %s", f.Namespace.Name, serviceAccount)

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
							Name:            "busybox",
							Image:           "k8s.gcr.io/e2e-test-images/busybox:1.29-1", // this image supports both Linux and Windows
							Command:         []string{"sleep"},
							Args:            []string{"3600"},
							ImagePullPolicy: corev1.PullIfNotPresent,
						},
					},
					ServiceAccountName: serviceAccount,
				},
			},
		},
	}

	nodeOSDistro := "linux"
	if framework.NodeOSDistroIs("windows") {
		nodeOSDistro = "windows"
	}
	e2epod.SetNodeSelection(&d.Spec.Template.Spec, e2epod.NodeSelection{
		Selector: map[string]string{
			"kubernetes.io/os": nodeOSDistro,
		},
	})

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

// createSecretForArcCluster creates a secret called localtoken-<sa> with dummy data.
func createSecretForArcCluster(c kubernetes.Interface, namespace, serviceAccount string) {
	// TODO(chewong): remove this secret creation process once we stopped using fake arc cluster
	secretName := fmt.Sprintf("localtoken-%s", serviceAccount)
	framework.Logf("creating secret %s in %s namespace", secretName, namespace)

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

// validateMutatedPod validates the following properties of the mutated pod in order:
// 1. verify that all containers except the one in skipContainers have expected environment variables injected;
// 2. verify that all containers except the one in skipContainers have azure-identity-token mounted;
// 3. verify that the pod has a service account token volume projected;
// 4. verify that the pod has access to token file via `cat /var/run/secrets/tokens/azure-identity-token`.
func validateMutatedPod(f *framework.Framework, pod *corev1.Pod, skipContainers []string) {
	withoutSkipContainers := []corev1.Container{}
	// consider init containers as well
	allContainers := append(pod.Spec.Containers, pod.Spec.InitContainers...)
	for _, c := range allContainers {
		keepContainer := true
		for _, skip := range skipContainers {
			if c.Name == skip {
				keepContainer = false
				break
			}
		}
		if keepContainer {
			withoutSkipContainers = append(withoutSkipContainers, c)
		}
	}

	for _, container := range withoutSkipContainers {
		m := make(map[string]struct{})
		for _, env := range container.Env {
			m[env.Name] = struct{}{}
		}

		framework.Logf("ensuring that the correct environment variables are injected to %s in %s", container.Name, pod.Name)
		for _, injected := range []string{
			webhook.AzureClientIDEnvVar,
			webhook.AzureTenantIDEnvVar,
			webhook.AzureAuthorityHostEnvVar,
			webhook.AzureFederatedTokenFileEnvVar,
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
						Sources:     getVolumeProjectionSources(pod.Spec.ServiceAccountName),
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

	if len(withoutSkipContainers) > 0 {
		err := e2epod.WaitForPodNameRunningInNamespace(f.ClientSet, pod.Name, pod.Namespace)
		framework.ExpectNoError(err, "failed to start pod %s", pod.Name)
		_ = f.ExecCommandInContainer(pod.Name, withoutSkipContainers[0].Name, "cat", filepath.Join(webhook.TokenFileMountPath, webhook.TokenFilePathName))
	}
}

// validateUnmutatedContainers validates that the environment variables and the volume mounts
// are not injected to the skip containers of the pod.
func validateUnmutatedContainers(f *framework.Framework, pod *corev1.Pod, skipContainers []string) {
	framework.Logf("validating that %v in %s are unmutated", skipContainers, pod.Name)
	noEnv := func(c corev1.Container) {
		gomega.Expect(c.Env).To(gomega.BeEmpty())
	}
	noVolumeMount := func(c corev1.Container) {
		for _, volumeMount := range c.VolumeMounts {
			gomega.Expect(volumeMount.Name).NotTo(gomega.Equal(webhook.TokenFilePathName))
		}
	}
	for _, c := range pod.Spec.Containers {
		for _, skip := range skipContainers {
			if c.Name == skip {
				noEnv(c)
				noVolumeMount(c)
			}
		}
	}
}

func getVolumeProjectionSources(serviceAccountName string) []corev1.VolumeProjection {
	// This is only required because webhook v0.6.0 uses 86400 for default token expiration
	// and we are running upgrade tests.
	// TODO(aramase): remove this after next release
	expirationSeconds := int64(serviceAccountTokenExpiration.Seconds())

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
