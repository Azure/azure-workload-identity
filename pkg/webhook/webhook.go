package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/Azure/aad-pod-managed-identity/pkg/config"

	"github.com/Azure/go-autorest/autorest/azure"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// +kubebuilder:webhook:path=/mutate-v1-pod,mutating=true,failurePolicy=ignore,groups="",resources=pods,verbs=create;update,versions=v1,name=mpod.aad-pod-identity.io,sideEffects=None,admissionReviewVersions=v1;v1beta1,matchPolicy=Equivalent
// +kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;create;update;patch

// podMutator mutates pod objects to add project service account token volume
type podMutator struct {
	client  client.Client
	config  *config.Config
	decoder *admission.Decoder
}

// NewPodMutator returns a pod mutation handler
func NewPodMutator(client client.Client) (*podMutator, error) {
	c, err := config.ParseConfig("/etc/kubernetes/azure.json")
	if err != nil {
		return nil, err
	}
	return &podMutator{
		client: client,
		config: c,
	}, nil
}

// PodMutator adds projected service account volume for incoming pods if service account is annotated
func (m *podMutator) Handle(ctx context.Context, req admission.Request) admission.Response {
	pod := &corev1.Pod{}

	err := m.decoder.Decode(req, pod)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	namespace := pod.GetNamespace()
	// use default namespace if not explicitly defined
	if namespace == "" {
		namespace = corev1.NamespaceDefault
	}

	logger := log.Log.WithName("handler").WithValues("pod", pod.GetName(), "namespace", namespace, "serviceAccount", pod.Spec.ServiceAccountName)
	// get service account associated with the pod
	serviceAccount := &corev1.ServiceAccount{}
	err = m.client.Get(ctx, types.NamespacedName{Name: pod.Spec.ServiceAccountName, Namespace: namespace}, serviceAccount)
	if err != nil {
		logger.Error(err, "failed to get service account")
		return admission.Errored(http.StatusBadRequest, err)
	}
	// check if the service account has the annotation
	if !isServiceAccountAnnotated(serviceAccount) {
		logger.Info("service account not annotated")
		return admission.Allowed("service account not annotated")
	}
	// get service account token expiration
	serviceAccountTokenExpiration, err := getServiceAccountTokenExpiration(pod, serviceAccount)
	if err != nil {
		logger.Error(err, "failed to get service account token expiration")
		return admission.Errored(http.StatusBadRequest, err)
	}
	// get the clientID
	clientID := getClientID(serviceAccount)
	// get the tenantID
	tenantID := getTenantID(serviceAccount, m.config)
	// get containers to skip
	skipContainers := getSkipContainers(pod)
	for i := range pod.Spec.Containers {
		// container is in the skip list
		if _, ok := skipContainers[pod.Spec.Containers[i].Name]; ok {
			continue
		}
		// add environment variables to container if not exists
		pod.Spec.Containers[i] = addEnvironmentVariables(pod.Spec.Containers[i], clientID, tenantID)
		// add the volume mount if not exists
		pod.Spec.Containers[i] = addProjectServiceAccountTokenVolumeMount(pod.Spec.Containers[i])
	}

	// add the projected service account token volume to the pod if not exists
	err = addProjectedServiceAccountTokenVolume(pod, m.config, serviceAccountTokenExpiration)
	if err != nil {
		logger.Error(err, "failed to add projected service account volume")
		return admission.Errored(http.StatusBadRequest, err)
	}

	marshaledPod, err := json.Marshal(pod)
	if err != nil {
		logger.Error(err, "failed to marshal pod object")
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}

// PodMutator implements admission.DecoderInjector
// A decoder will be automatically injected

// InjectDecoder injects the decoder
func (m *podMutator) InjectDecoder(d *admission.Decoder) error {
	m.decoder = d
	return nil
}

// isServiceAccountAnnotated checks if the service account has been annotated
// to use with pod identity
func isServiceAccountAnnotated(sa *corev1.ServiceAccount) bool {
	if len(sa.Labels) == 0 {
		return false
	}
	_, ok := sa.Labels[UsePodIdentityLabel]
	return ok
}

// getSkipContainers gets the list of containers to skip based on the annotation
func getSkipContainers(pod *corev1.Pod) map[string]struct{} {
	skipContainers := pod.Annotations[SkipContainersAnnotation]
	if len(skipContainers) == 0 {
		return nil
	}
	skipContainersList := strings.Split(skipContainers, ";")
	m := make(map[string]struct{})
	for _, skipContainer := range skipContainersList {
		m[strings.TrimSpace(skipContainer)] = struct{}{}
	}
	return m
}

// getServiceAccountTokenExpiration returns the expiration seconds for the project service account token volume
// Order of preference:
// 	1. annotation in the pod
// 	2. annotation in the service account
//	default expiration if no annotation specified
func getServiceAccountTokenExpiration(pod *corev1.Pod, sa *corev1.ServiceAccount) (int64, error) {
	serviceAccountTokenExpiration := DefaultServiceAccountTokenExpiration
	var err error
	// check if expiry defined in the pod with annotation
	if pod.Annotations != nil && pod.Annotations[ServiceAccountTokenExpiryAnnotation] != "" {
		if serviceAccountTokenExpiration, err = strconv.ParseInt(pod.Annotations[ServiceAccountTokenExpiryAnnotation], 10, 64); err != nil {
			return 0, err
		}
	} else if sa.Annotations != nil && sa.Annotations[ServiceAccountTokenExpiryAnnotation] != "" {
		if serviceAccountTokenExpiration, err = strconv.ParseInt(sa.Annotations[ServiceAccountTokenExpiryAnnotation], 10, 64); err != nil {
			return 0, err
		}
	}
	// validate expiration time
	if !validServiceAccountTokenExpiry(serviceAccountTokenExpiration) {
		return 0, fmt.Errorf("token expiration %d not valid. Expected value to be between 3600 and 86400", serviceAccountTokenExpiration)
	}
	return serviceAccountTokenExpiration, nil
}

func validServiceAccountTokenExpiry(tokenExpiry int64) bool {
	return tokenExpiry <= DefaultServiceAccountTokenExpiration && tokenExpiry >= MinServiceAccountTokenExpiration
}

// getClientID returns the clientID to be configured
func getClientID(sa *corev1.ServiceAccount) string {
	return sa.Annotations[ClientIDAnnotation]
}

// getTenantID returns the tenantID to be configured
func getTenantID(sa *corev1.ServiceAccount, c *config.Config) string {
	// use tenantID if provided in the annotation
	if tenantID, ok := sa.Annotations[TenantIDAnnotation]; ok {
		return tenantID
	}
	// use the cluster tenantID as default value
	return c.TenantID
}

// addEnvironmentVariables adds the clientID, tenantID and token file path environment variables needed for SDK
func addEnvironmentVariables(container corev1.Container, clientID, tenantID string) corev1.Container {
	m := make(map[string]string)
	for _, env := range container.Env {
		m[env.Name] = env.Value
	}
	// add the clientID env var
	if _, ok := m[AzureClientIDEnvVar]; !ok {
		container.Env = append(container.Env, corev1.EnvVar{Name: AzureClientIDEnvVar, Value: clientID})
	}
	// add the tenantID env var
	if _, ok := m[AzureTenantIDEnvVar]; !ok {
		container.Env = append(container.Env, corev1.EnvVar{Name: AzureTenantIDEnvVar, Value: tenantID})
	}
	// add the token file path env var
	if _, ok := m[TokenFilePathEnvVar]; !ok {
		container.Env = append(container.Env, corev1.EnvVar{Name: TokenFilePathEnvVar, Value: "/var/run/secrets/tokens/azure-identity-token"})
	}

	return container
}

// addProjectServiceAccountTokenVolumeMount adds the projected service account token volume mount for the container
func addProjectServiceAccountTokenVolumeMount(container corev1.Container) corev1.Container {
	for _, volume := range container.VolumeMounts {
		if volume.Name == "azure-identity-token" {
			return container
		}
	}
	container.VolumeMounts = append(container.VolumeMounts,
		corev1.VolumeMount{
			Name:      "azure-identity-token",
			MountPath: "/var/run/secrets/tokens",
			ReadOnly:  true,
		})

	return container
}

func addProjectedServiceAccountTokenVolume(pod *corev1.Pod, config *config.Config, serviceAccountTokenExpiration int64) error {
	// add the projected service account token volume to the pod if not exists
	for _, volume := range pod.Spec.Volumes {
		if volume.Projected == nil {
			continue
		}
		for _, pvs := range volume.Projected.Sources {
			if pvs.ServiceAccountToken == nil {
				continue
			}
			if pvs.ServiceAccountToken.Path == "azure-identity-token" {
				return nil
			}
		}
	}

	// get aad endpoint to configure as audience
	aadEndpoint, err := getAADEndpoint(config)
	if err != nil {
		return fmt.Errorf("failed to get AAD endpoint: %v", err)
	}
	aadEndpoint = strings.TrimRight(aadEndpoint, "/")

	// add the projected service account token volume
	// the path for this volume will always be set to "azure-identity-token"
	pod.Spec.Volumes = append(
		pod.Spec.Volumes,
		corev1.Volume{
			Name: "azure-identity-token",
			VolumeSource: corev1.VolumeSource{
				Projected: &corev1.ProjectedVolumeSource{
					Sources: []corev1.VolumeProjection{
						{
							ServiceAccountToken: &corev1.ServiceAccountTokenProjection{
								Path:              "azure-identity-token",
								ExpirationSeconds: &serviceAccountTokenExpiration,
								Audience:          fmt.Sprintf("%s/federatedidentity", aadEndpoint),
							},
						},
					},
				},
			},
		})

	return nil
}

// TODO use https://login.microsoftonline.com/federatedidentity as audience
func getAADEndpoint(c *config.Config) (string, error) {
	var env azure.Environment
	var err error
	if c.Cloud == "" {
		env = azure.PublicCloud
	} else {
		env, err = azure.EnvironmentFromName(c.Cloud)
	}
	return env.ActiveDirectoryEndpoint, err
}
