//go:build e2e

package e2e

import (
	"context"
	"path/filepath"

	"github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/kubernetes"
	"k8s.io/kubernetes/test/e2e/framework"
	e2edeploy "k8s.io/kubernetes/test/e2e/framework/deployment"
	e2epod "k8s.io/kubernetes/test/e2e/framework/pod"
	"k8s.io/utils/pointer"
)

const (
	busybox1 = "busybox-1"
	busybox2 = "busybox-2"
)

// createServiceAccount creates a service account with customizable name, namespace, labels and annotations.
func createServiceAccount(c kubernetes.Interface, namespace, name string, annotations map[string]string) string {
	account := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
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
func createPodWithServiceAccount(c kubernetes.Interface, namespace, serviceAccount, image string, command, args []string, env []corev1.EnvVar, annotations, labels map[string]string, runAsRoot bool) (*corev1.Pod, error) {
	pod := generatePodWithServiceAccount(c, namespace, serviceAccount, image, command, args, env, annotations, labels, runAsRoot)
	return createPod(c, pod)
}

// generatePodWithServiceAccount generates a pod with two containers, busybox-1 and busybox-2 with customizable
// namespace, service account, image, command, arguments, environment variables, and annotations.
func generatePodWithServiceAccount(c kubernetes.Interface, namespace, serviceAccount, image string, command, args []string, env []corev1.EnvVar, annotations, labels map[string]string, runAsRoot bool) *corev1.Pod {
	// this is required for pod to be admitted in kubernetes 1.24+
	contSecurityContext := &corev1.SecurityContext{
		AllowPrivilegeEscalation: pointer.Bool(false),
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{"ALL"},
		},
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
		RunAsNonRoot: pointer.Bool(true),
		RunAsUser:    pointer.Int64(1000),
	}
	if runAsRoot {
		contSecurityContext.RunAsNonRoot = pointer.Bool(false)
		contSecurityContext.RunAsUser = pointer.Int64(0)
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: namespace + "-",
			Namespace:    namespace,
			Annotations:  annotations,
			Labels:       labels,
		},
		Spec: corev1.PodSpec{
			TerminationGracePeriodSeconds: pointer.Int64(0),
			Containers: []corev1.Container{{
				Name:            busybox1,
				Image:           image, // this image should support both Linux and Windows
				Command:         command,
				Args:            args,
				Env:             env,
				ImagePullPolicy: corev1.PullIfNotPresent,
				SecurityContext: contSecurityContext,
			}, {
				Name:            busybox2,
				Image:           image, // this image should support both Linux and Windows
				Command:         command,
				Args:            args,
				Env:             env,
				ImagePullPolicy: corev1.PullIfNotPresent,
				SecurityContext: contSecurityContext,
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
func createPodUsingDeploymentWithServiceAccount(ctx context.Context, f *framework.Framework, serviceAccount string) *corev1.Pod {
	framework.Logf("creating a deployment in %s namespace with service account %s", f.Namespace.Name, serviceAccount)

	podLabels := map[string]string{
		"app":                    "busybox",
		useWorkloadIdentityLabel: "true",
	}
	nonRootUser := int64(1000)

	d := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: f.Namespace.Name + "-",
			Namespace:    f.Namespace.Name,
			Labels:       podLabels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: pointer.Int32Ptr(1),
			Selector: &metav1.LabelSelector{MatchLabels: podLabels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: podLabels,
				},
				Spec: corev1.PodSpec{
					TerminationGracePeriodSeconds: pointer.Int64(0),
					Containers: []corev1.Container{
						{
							Name:            "busybox",
							Image:           "registry.k8s.io/e2e-test-images/busybox:1.29-4", // this image supports both Linux and Windows
							Command:         []string{"sleep"},
							Args:            []string{"3600"},
							ImagePullPolicy: corev1.PullIfNotPresent,
							SecurityContext: &corev1.SecurityContext{
								AllowPrivilegeEscalation: pointer.Bool(false),
								Capabilities: &corev1.Capabilities{
									Drop: []corev1.Capability{"ALL"},
								},
								RunAsNonRoot: pointer.Bool(true),
								SeccompProfile: &corev1.SeccompProfile{
									Type: corev1.SeccompProfileTypeRuntimeDefault,
								},
								RunAsUser: &nonRootUser,
							},
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

	d, err := f.ClientSet.AppsV1().Deployments(f.Namespace.Name).Create(ctx, d, metav1.CreateOptions{})
	framework.ExpectNoError(err, "failed to create deployment %s", d.Name)

	err = e2edeploy.WaitForDeploymentComplete(f.ClientSet, d)
	framework.ExpectNoError(err, "failed to complete deployment %s", d.Name)

	podList, err := e2edeploy.GetPodsForDeployment(ctx, f.ClientSet, d)
	framework.ExpectNoError(err, "failed to get pods for deployment %s", d.Name)
	pod := &podList.Items[0]

	framework.Logf("created pod %s with deployment %s", pod.Name, d.Name)
	return pod
}

// validateMutatedPod validates the following properties of the mutated pod in order:
// 1. verify that all containers except the one in skipContainers have expected environment variables injected;
// 2. verify that all containers except the one in skipContainers have azure-identity-token mounted;
// 3. verify that the pod has a service account token volume projected;
// 4. verify that the pod has access to token file via `cat /var/run/secrets/azure/tokens/azure-identity-token`.
func validateMutatedPod(ctx context.Context, f *framework.Framework, pod *corev1.Pod, skipContainers []string) {
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
		m := sets.New[string]()
		for _, env := range container.Env {
			m.Insert(env.Name)
		}

		framework.Logf("ensuring that the correct environment variables are injected to %s in %s", container.Name, pod.Name)
		for _, injected := range []string{
			"AZURE_CLIENT_ID",
			"AZURE_TENANT_ID",
			"AZURE_AUTHORITY_HOST",
			"AZURE_FEDERATED_TOKEN_FILE",
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
					Name:      tokenFilePathName,
					MountPath: tokenFileMountPath,
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
		if volume.Name == tokenFilePathName {
			found = true
			gomega.Expect(volume).To(gomega.Equal(corev1.Volume{
				Name: tokenFilePathName,
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
		err := e2epod.WaitForPodNameRunningInNamespace(ctx, f.ClientSet, pod.Name, pod.Namespace)
		framework.ExpectNoError(err, "failed to start pod %s", pod.Name)
		_ = e2epod.ExecCommandInContainer(f, pod.Name, withoutSkipContainers[0].Name, "cat", filepath.Join(tokenFileMountPath, tokenFilePathName))
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
			gomega.Expect(volumeMount.Name).NotTo(gomega.Equal(tokenFilePathName))
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
	return []corev1.VolumeProjection{{
		ServiceAccountToken: &corev1.ServiceAccountTokenProjection{
			Path:              tokenFilePathName,
			ExpirationSeconds: pointer.Int64(3600),
			Audience:          "api://AzureADTokenExchange",
		}},
	}
}

func validateProxySideCarInMutatedPod(pod *corev1.Pod) {
	framework.Logf("validating that the proxy sidecar is injected to %s", pod.Name)
	containers := pod.Spec.Containers
	if useNativeSidecar {
		framework.Logf("validating that the proxy init container is injected as native sidecar to %s", pod.Name)
		containers = pod.Spec.InitContainers
	}

	proxySidecar := getProxySidecarContainer(containers)
	gomega.Expect(proxySidecar).NotTo(gomega.BeNil(), "proxy sidecar is not injected to pod %s", pod.Name)

	if useNativeSidecar {
		gomega.Expect(proxySidecar.RestartPolicy).ToNot(gomega.BeNil(), "proxy sidecar in pod %s should have a restart policy", pod.Name)
		gomega.Expect(*proxySidecar.RestartPolicy).To(gomega.Equal(corev1.ContainerRestartPolicyAlways), "proxy sidecar in pod %s should have restart policy 'Always'", pod.Name)
	} else {
		gomega.Expect(proxySidecar.RestartPolicy).To(gomega.BeNil(), "proxy sidecar in pod %s should not have a restart policy", pod.Name)
	}
}

func getProxySidecarContainer(containers []corev1.Container) *corev1.Container {
	for _, container := range containers {
		if container.Name == "azwi-proxy" {
			return &container
		}
	}
	return nil
}
