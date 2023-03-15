package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"
	"monis.app/mlog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/Azure/azure-workload-identity/pkg/config"
)

var (
	// ProxyImageRegistry is the image registry for the proxy init and sidecar.
	// This is injected via LDFLAGS in the Makefile during the build.
	ProxyImageRegistry string
	// ProxyImageVersion is the image version of the proxy init and sidecar.
	// This is injected via LDFLAGS in the Makefile during the build.
	ProxyImageVersion string
)

// +kubebuilder:webhook:path=/mutate-v1-pod,mutating=true,failurePolicy=fail,groups="",resources=pods,verbs=create,versions=v1,name=mutation.azure-workload-identity.io,sideEffects=None,admissionReviewVersions=v1;v1beta1,matchPolicy=Equivalent,reinvocationPolicy=IfNeeded
// +kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch

// this is required for the webhook server certs generated and rotated as part of cert-controller rotator
// +kubebuilder:rbac:groups="",namespace=azure-workload-identity-system,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=admissionregistration.k8s.io,resources=mutatingwebhookconfigurations,verbs=get;list;watch;update

// podMutator mutates pod objects to add project service account token volume
type podMutator struct {
	client client.Client
	// reader is an instance of mgr.GetAPIReader that is configured to use the API server.
	// This should be used sparingly and only when the client does not fit the use case.
	reader             client.Reader
	config             *config.Config
	decoder            *admission.Decoder
	audience           string
	azureAuthorityHost string
	reporter           StatsReporter
}

// NewPodMutator returns a pod mutation handler
func NewPodMutator(client client.Client, reader client.Reader, audience string) (admission.Handler, error) {
	c, err := config.ParseConfig()
	if err != nil {
		return nil, err
	}
	if audience == "" {
		audience = DefaultAudience
	}
	// this is used to configure the AZURE_AUTHORITY_HOST env var that's
	// used by the azure sdk
	azureAuthorityHost, err := getAzureAuthorityHost(c)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get AAD endpoint")
	}

	return &podMutator{
		client:             client,
		reader:             reader,
		config:             c,
		audience:           audience,
		azureAuthorityHost: azureAuthorityHost,
		reporter:           newStatsReporter(),
	}, nil
}

// PodMutator adds projected service account volume for incoming pods if service account is annotated
func (m *podMutator) Handle(ctx context.Context, req admission.Request) (response admission.Response) {
	timeStart := time.Now()
	defer func() {
		if m.reporter != nil {
			m.reporter.ReportRequest(ctx, req.Namespace, time.Since(timeStart))
		}
	}()

	pod := &corev1.Pod{}
	err := m.decoder.Decode(req, pod)
	if err != nil {
		return admission.Errored(http.StatusBadRequest, err)
	}

	podName := pod.GetName()
	if podName == "" {
		podName = pod.GetGenerateName() + " (prefix)"
	}
	// for daemonset/deployment pods the namespace field is not set in objectMeta
	// explicitly set the namespace to request namespace
	pod.Namespace = req.Namespace
	serviceAccountName := pod.Spec.ServiceAccountName
	// When you create a pod, if you do not specify a service account, it is automatically
	// assigned the default service account in the same namespace.
	// xref: https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/#use-the-default-service-account-to-access-the-api-server
	if serviceAccountName == "" {
		serviceAccountName = "default"
	}

	logger := mlog.New().WithName("handler").WithValues("pod", podName, "namespace", pod.Namespace, "service-account", serviceAccountName)
	// get service account associated with the pod
	serviceAccount := &corev1.ServiceAccount{}
	if err = m.client.Get(ctx, types.NamespacedName{Name: serviceAccountName, Namespace: pod.Namespace}, serviceAccount); err != nil {
		if !apierrors.IsNotFound(err) {
			logger.Error("failed to get service account", err)
			return admission.Errored(http.StatusBadRequest, err)
		}
		// bypass cache and get from the API server as it's not found in cache
		err = m.reader.Get(ctx, types.NamespacedName{Name: serviceAccountName, Namespace: pod.Namespace}, serviceAccount)
		if err != nil {
			logger.Error("failed to get service account", err)
			return admission.Errored(http.StatusBadRequest, err)
		}
	}

	if shouldInjectProxySidecar(pod) {
		proxyPort, err := getProxyPort(pod)
		if err != nil {
			logger.Error("failed to get proxy port", err)
			return admission.Errored(http.StatusBadRequest, err)
		}

		pod.Spec.InitContainers = m.injectProxyInitContainer(pod.Spec.InitContainers, proxyPort)
		pod.Spec.Containers = m.injectProxySidecarContainer(pod.Spec.Containers, proxyPort)
	}

	// get service account token expiration
	serviceAccountTokenExpiration, err := getServiceAccountTokenExpiration(pod, serviceAccount)
	if err != nil {
		logger.Error("failed to get service account token expiration", err)
		return admission.Errored(http.StatusBadRequest, err)
	}
	// get the clientID
	clientID := getClientID(serviceAccount)
	// get the tenantID
	tenantID := getTenantID(serviceAccount, m.config)
	// get containers to skip
	skipContainers := getSkipContainers(pod)
	pod.Spec.InitContainers = m.mutateContainers(pod.Spec.InitContainers, clientID, tenantID, skipContainers)
	pod.Spec.Containers = m.mutateContainers(pod.Spec.Containers, clientID, tenantID, skipContainers)

	// add the projected service account token volume to the pod if not exists
	if err = addProjectedServiceAccountTokenVolume(pod, serviceAccountTokenExpiration, m.audience); err != nil {
		logger.Error("failed to add projected service account volume", err)
		return admission.Errored(http.StatusBadRequest, err)
	}

	marshaledPod, err := json.Marshal(pod)
	if err != nil {
		logger.Error("failed to marshal pod object", err)
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

// mutateContainers mutates the containers by injecting the projected
// service account token volume and environment variables
func (m *podMutator) mutateContainers(containers []corev1.Container, clientID string, tenantID string, skipContainers map[string]struct{}) []corev1.Container {
	for i := range containers {
		// container is in the skip list
		if _, ok := skipContainers[containers[i].Name]; ok {
			continue
		}
		// add environment variables to container if not exists
		containers[i] = addEnvironmentVariables(containers[i], clientID, tenantID, m.azureAuthorityHost)
		// add the volume mount if not exists
		containers[i] = addProjectedTokenVolumeMount(containers[i])
	}
	return containers
}

func (m *podMutator) injectProxyInitContainer(containers []corev1.Container, proxyPort int32) []corev1.Container {
	imageRepository := strings.Join([]string{ProxyImageRegistry, ProxyInitImageName}, "/")
	for _, container := range containers {
		if strings.HasPrefix(container.Image, imageRepository) || container.Name == ProxyInitContainerName {
			return containers
		}
	}

	containers = append(containers, corev1.Container{
		Name:            ProxyInitContainerName,
		Image:           strings.Join([]string{imageRepository, ProxyImageVersion}, ":"),
		ImagePullPolicy: corev1.PullIfNotPresent,
		SecurityContext: &corev1.SecurityContext{
			Capabilities: &corev1.Capabilities{
				Add:  []corev1.Capability{"NET_ADMIN"},
				Drop: []corev1.Capability{"ALL"},
			},
			Privileged:   pointer.Bool(true),
			RunAsNonRoot: pointer.Bool(false),
			RunAsUser:    pointer.Int64(0),
		},
		Env: []corev1.EnvVar{{
			Name:  ProxyPortEnvVar,
			Value: strconv.FormatInt(int64(proxyPort), 10),
		}},
	})

	return containers
}

func (m *podMutator) injectProxySidecarContainer(containers []corev1.Container, proxyPort int32) []corev1.Container {
	imageRepository := strings.Join([]string{ProxyImageRegistry, ProxySidecarImageName}, "/")
	for _, container := range containers {
		if strings.HasPrefix(container.Image, imageRepository) || container.Name == ProxySidecarContainerName {
			return containers
		}
	}

	logLevel := currentLogLevel() // run the proxy at the same log level as the webhook
	containers = append(containers, corev1.Container{
		Name:            ProxySidecarContainerName,
		Image:           strings.Join([]string{imageRepository, ProxyImageVersion}, ":"),
		ImagePullPolicy: corev1.PullIfNotPresent,
		Args: []string{
			fmt.Sprintf("--proxy-port=%d", proxyPort),
			fmt.Sprintf("--log-level=%s", logLevel),
		},
		Ports: []corev1.ContainerPort{{
			ContainerPort: proxyPort,
		}},
		Lifecycle: &corev1.Lifecycle{
			PostStart: &corev1.LifecycleHandler{
				Exec: &corev1.ExecAction{
					Command: []string{
						"/proxy",
						fmt.Sprintf("--proxy-port=%d", proxyPort),
						"--probe",
						fmt.Sprintf("--log-level=%s", logLevel),
					},
				},
			},
		},
	})

	return containers
}

func shouldInjectProxySidecar(pod *corev1.Pod) bool {
	if len(pod.Annotations) == 0 {
		return false
	}
	_, ok := pod.Annotations[InjectProxySidecarAnnotation]
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
//  1. annotation in the pod
//  2. annotation in the service account
//     default expiration if no annotation specified
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
		return 0, errors.Errorf("token expiration %d not valid. Expected value to be between 3600 and 86400", serviceAccountTokenExpiration)
	}
	return serviceAccountTokenExpiration, nil
}

// getProxyPort returns the port for the proxy init container and the proxy sidecar container
func getProxyPort(pod *corev1.Pod) (int32, error) {
	if len(pod.Annotations) == 0 {
		return DefaultProxySidecarPort, nil
	}

	proxyPort, ok := pod.Annotations[ProxySidecarPortAnnotation]
	if !ok {
		return DefaultProxySidecarPort, nil
	}

	parsed, err := strconv.ParseInt(proxyPort, 10, 32)
	if err != nil {
		return 0, errors.Wrap(err, "failed to parse proxy sidecar port")
	}

	return int32(parsed), nil
}

func validServiceAccountTokenExpiry(tokenExpiry int64) bool {
	return tokenExpiry <= MaxServiceAccountTokenExpiration && tokenExpiry >= MinServiceAccountTokenExpiration
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
func addEnvironmentVariables(container corev1.Container, clientID, tenantID, azureAuthorityHost string) corev1.Container {
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
	// add the token file env var
	if _, ok := m[AzureFederatedTokenFileEnvVar]; !ok {
		container.Env = append(container.Env, corev1.EnvVar{Name: AzureFederatedTokenFileEnvVar, Value: filepath.Join(TokenFileMountPath, TokenFilePathName)})
	}
	// add the azure authority host env var
	if _, ok := m[AzureAuthorityHostEnvVar]; !ok {
		container.Env = append(container.Env, corev1.EnvVar{Name: AzureAuthorityHostEnvVar, Value: azureAuthorityHost})
	}

	return container
}

// addProjectedTokenVolumeMount adds the projected token volume mount for the container
func addProjectedTokenVolumeMount(container corev1.Container) corev1.Container {
	for _, volume := range container.VolumeMounts {
		if volume.Name == TokenFilePathName {
			return container
		}
	}
	container.VolumeMounts = append(container.VolumeMounts,
		corev1.VolumeMount{
			Name:      TokenFilePathName,
			MountPath: TokenFileMountPath,
			ReadOnly:  true,
		})

	return container
}

func addProjectedServiceAccountTokenVolume(pod *corev1.Pod, serviceAccountTokenExpiration int64, audience string) error {
	// add the projected service account token volume to the pod if not exists
	for _, volume := range pod.Spec.Volumes {
		if volume.Projected == nil {
			continue
		}
		for _, pvs := range volume.Projected.Sources {
			if pvs.ServiceAccountToken == nil {
				continue
			}
			if pvs.ServiceAccountToken.Path == TokenFilePathName {
				return nil
			}
		}
	}

	// add the projected service account token volume
	// the path for this volume will always be set to "azure-identity-token"
	pod.Spec.Volumes = append(
		pod.Spec.Volumes,
		corev1.Volume{
			Name: TokenFilePathName,
			VolumeSource: corev1.VolumeSource{
				Projected: &corev1.ProjectedVolumeSource{
					Sources: []corev1.VolumeProjection{
						{
							ServiceAccountToken: &corev1.ServiceAccountTokenProjection{
								Path:              TokenFilePathName,
								ExpirationSeconds: &serviceAccountTokenExpiration,
								Audience:          audience,
							},
						},
					},
				},
			},
		})

	return nil
}

// getAzureAuthorityHost returns the active directory endpoint to use for requesting
// tokens based on the azure environment the webhook is configured with.
func getAzureAuthorityHost(c *config.Config) (string, error) {
	var env azure.Environment
	var err error
	if c.Cloud == "" {
		env = azure.PublicCloud
	} else {
		env, err = azure.EnvironmentFromName(c.Cloud)
	}
	return env.ActiveDirectoryEndpoint, err
}

func currentLogLevel() string {
	for _, level := range []mlog.LogLevel{
		// iterate in reverse order
		mlog.LevelAll,
		mlog.LevelTrace,
		mlog.LevelDebug,
		mlog.LevelInfo,
		mlog.LevelWarning,
	} {
		if mlog.Enabled(level) {
			return string(level)
		}
	}
	return "" // this is unreachable
}
