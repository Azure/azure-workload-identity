package webhook

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/go-autorest/autorest/azure"
	"github.com/pkg/errors"
	"monis.app/mlog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/validation"
	utilversion "k8s.io/apimachinery/pkg/util/version"
	"k8s.io/client-go/discovery"
	"k8s.io/client-go/rest"
	"k8s.io/utils/ptr"

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
	reader              client.Reader
	config              *config.Config
	decoder             admission.Decoder
	audience            string
	azureAuthorityHost  string
	proxyImage          string
	proxyInitImage      string
	useNativeSidecar    bool
	customTokenEndpoint customTokenEndpointConfig
}

// customTokenEndpointConfig holds the configuration for custom token endpoint
type customTokenEndpointConfig struct {
	annotation string
	audience   string
}

// NewPodMutator returns a pod mutation handler
func NewPodMutator(client client.Client, reader client.Reader, audience string, scheme *runtime.Scheme, restConfig *rest.Config, customTokenEndpointAnnotationSuffix, customTokenEndpointAudience string) (admission.Handler, error) {
	c, err := config.ParseConfig()
	if err != nil {
		return nil, err
	}
	if audience == "" {
		audience = DefaultAudience
	}

	if len(customTokenEndpointAnnotationSuffix) == 0 {
		return nil, fmt.Errorf("custom token endpoint annotation suffix cannot be empty")
	}
	if len(customTokenEndpointAudience) == 0 {
		return nil, fmt.Errorf("custom token endpoint audience cannot be empty")
	}

	if strings.EqualFold(customTokenEndpointAudience, DefaultAudience) {
		return nil, fmt.Errorf("custom token endpoint audience cannot be equal to default audience %q", DefaultAudience)
	}
	// Using the validation logic for keys from https://github.com/kubernetes/kubernetes/blob/69dbc74417304328a9fd3c161643dc4f0a057f41/staging/src/k8s.io/apimachinery/pkg/api/validation/objectmeta.go#L46-L51
	annotationKey := fmt.Sprintf("azure.workload.identity/use-%s", customTokenEndpointAnnotationSuffix)
	errs := validation.IsQualifiedName(strings.ToLower(annotationKey))
	if len(errs) > 0 {
		return nil, fmt.Errorf("custom token endpoint annotation key %q is not valid: %s", annotationKey, strings.Join(errs, ", "))
	}

	discoveryClient, err := discovery.NewDiscoveryClientForConfig(restConfig)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create discovery client")
	}
	// "SidecarContainers" went beta in 1.29. With the 3 version skew policy,
	// between API server and kubelet, 1.32 is the earliest version this can be
	// safely used.
	useNativeSidecar, err := serverVersionGTE(discoveryClient, utilversion.MajorMinor(1, 32))
	if err != nil {
		return nil, errors.Wrap(err, "failed to check kubernetes version")
	}

	// this is used to configure the AZURE_AUTHORITY_HOST env var that's
	// used by the azure sdk
	azureAuthorityHost, err := getAzureAuthorityHost(c)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get AAD endpoint")
	}
	proxyImage := c.ProxyImage
	if len(proxyImage) == 0 {
		proxyImage = fmt.Sprintf("%s/%s:%s", ProxyImageRegistry, ProxySidecarImageName, ProxyImageVersion)
	}
	proxyInitImage := c.ProxyInitImage
	if len(proxyInitImage) == 0 {
		proxyInitImage = fmt.Sprintf("%s/%s:%s", ProxyImageRegistry, ProxyInitImageName, ProxyImageVersion)
	}

	if err := registerMetrics(); err != nil {
		return nil, errors.Wrap(err, "failed to register metrics")
	}

	return &podMutator{
		client:             client,
		reader:             reader,
		config:             c,
		decoder:            admission.NewDecoder(scheme),
		audience:           audience,
		azureAuthorityHost: azureAuthorityHost,
		proxyImage:         proxyImage,
		proxyInitImage:     proxyInitImage,
		useNativeSidecar:   useNativeSidecar,
		customTokenEndpoint: customTokenEndpointConfig{
			annotation: fmt.Sprintf("azure.workload.identity/use-%s", customTokenEndpointAnnotationSuffix),
			audience:   customTokenEndpointAudience,
		},
	}, nil
}

// PodMutator adds projected service account volume for incoming pods if service account is annotated
func (m *podMutator) Handle(ctx context.Context, req admission.Request) (response admission.Response) {
	timeStart := time.Now()
	defer func() {
		ReportRequest(ctx, req.Namespace, time.Since(timeStart))
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
		// if the pod has hostNetwork set to true, we cannot inject the proxy sidecar
		// as it'll end up modifying the network stack of the host and affecting other pods
		if pod.Spec.HostNetwork {
			err := errors.New("hostNetwork is set to true, cannot inject proxy sidecar")
			logger.Error("failed to inject proxy sidecar", err)
			return admission.Errored(http.StatusBadRequest, err)
		}

		proxyPort, err := getProxyPort(pod)
		if err != nil {
			logger.Error("failed to get proxy port", err)
			return admission.Errored(http.StatusBadRequest, err)
		}

		pod.Spec.InitContainers = m.injectProxyInitContainer(pod.Spec.InitContainers, proxyPort)
		if m.useNativeSidecar {
			pod.Spec.InitContainers = m.injectProxySidecarContainer(pod.Spec.InitContainers, proxyPort, ptr.To(corev1.ContainerRestartPolicyAlways))
		} else {
			pod.Spec.Containers = m.injectProxySidecarContainer(pod.Spec.Containers, proxyPort, nil)
		}
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
	podUsingCustomTokenEndpoint := m.isUsingCustomTokenEndpoint(pod)
	volumeName := buildVolumeName(podName)

	pod.Spec.InitContainers = m.mutateContainers(pod.Spec.InitContainers, clientID, tenantID, skipContainers, podUsingCustomTokenEndpoint, volumeName)
	pod.Spec.Containers = m.mutateContainers(pod.Spec.Containers, clientID, tenantID, skipContainers, podUsingCustomTokenEndpoint, volumeName)

	m.addProjectedVolume(pod, serviceAccountTokenExpiration, volumeName, podUsingCustomTokenEndpoint)

	marshaledPod, err := json.Marshal(pod)
	if err != nil {
		logger.Error("failed to marshal pod object", err)
		return admission.Errored(http.StatusInternalServerError, err)
	}
	return admission.PatchResponseFromRaw(req.Object.Raw, marshaledPod)
}

// mutateContainers mutates the containers by injecting the projected
// service account token volume and environment variables
func (m *podMutator) mutateContainers(containers []corev1.Container, clientID, tenantID string, skipContainers map[string]struct{}, podUsingCustomTokenEndpoint bool, volumeName string) []corev1.Container {
	for i := range containers {
		// container is in the skip list
		if _, ok := skipContainers[containers[i].Name]; ok {
			continue
		}
		// add environment variables to container if not exists
		containers[i] = m.addEnvironmentVariables(containers[i], clientID, tenantID, m.azureAuthorityHost, podUsingCustomTokenEndpoint)
		// add the volume mount if not exists
		containers[i] = addProjectedVolumeMount(containers[i], volumeName)
	}

	return containers
}

func (m *podMutator) injectProxyInitContainer(containers []corev1.Container, proxyPort int32) []corev1.Container {
	for _, container := range containers {
		if container.Name == ProxyInitContainerName {
			return containers
		}
	}
	containers = append(containers, corev1.Container{
		Name:            ProxyInitContainerName,
		Image:           m.proxyInitImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		SecurityContext: &corev1.SecurityContext{
			Capabilities: &corev1.Capabilities{
				Add:  []corev1.Capability{"NET_ADMIN"},
				Drop: []corev1.Capability{"ALL"},
			},
			Privileged:   ptr.To(true),
			RunAsNonRoot: ptr.To(false),
			RunAsUser:    ptr.To[int64](0),
		},
		Env: []corev1.EnvVar{{
			Name:  ProxyPortEnvVar,
			Value: strconv.FormatInt(int64(proxyPort), 10),
		}},
	})

	return containers
}

func (m *podMutator) injectProxySidecarContainer(containers []corev1.Container, proxyPort int32, restartPolicy *corev1.ContainerRestartPolicy) []corev1.Container {
	for _, container := range containers {
		if container.Name == ProxySidecarContainerName {
			return containers
		}
	}
	logLevel := currentLogLevel() // run the proxy at the same log level as the webhook
	containers = append([]corev1.Container{{
		Name:            ProxySidecarContainerName,
		Image:           m.proxyImage,
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
		SecurityContext: &corev1.SecurityContext{
			AllowPrivilegeEscalation: ptr.To(false),
			Capabilities: &corev1.Capabilities{
				Drop: []corev1.Capability{"ALL"},
			},
			Privileged:             ptr.To(false),
			ReadOnlyRootFilesystem: ptr.To(true),
			RunAsNonRoot:           ptr.To(true),
		},
		RestartPolicy: restartPolicy,
	}}, containers...)

	return containers
}

func shouldInjectProxySidecar(pod *corev1.Pod) bool {
	if len(pod.Annotations) == 0 {
		return false
	}
	_, ok := pod.Annotations[InjectProxySidecarAnnotation]
	return ok
}

func (m *podMutator) isUsingCustomTokenEndpoint(pod *corev1.Pod) bool {
	if len(pod.Annotations) == 0 {
		return false
	}
	if val, ok := pod.Annotations[m.customTokenEndpoint.annotation]; ok {
		return strings.EqualFold(val, "true")
	}
	return false
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

	return int32(parsed), nil //nolint:gosec // disable G115
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
func (m *podMutator) addEnvironmentVariables(container corev1.Container, clientID, tenantID, azureAuthorityHost string, podUsingIdentityBinding bool) corev1.Container {
	envs := make(map[string]string)
	for _, env := range container.Env {
		envs[env.Name] = env.Value
	}

	desiredEnvs := []corev1.EnvVar{
		{Name: AzureClientIDEnvVar, Value: clientID},
		{Name: AzureTenantIDEnvVar, Value: tenantID},
		{Name: AzureFederatedTokenFileEnvVar, Value: filepath.Join(VolumeMountPath, TokenFilePath)},
		{Name: AzureAuthorityHostEnvVar, Value: azureAuthorityHost},
	}

	if podUsingIdentityBinding {
		desiredEnvs = append(desiredEnvs, corev1.EnvVar{Name: AzureKubernetesTokenProxyEnvVar, Value: m.config.AzureKubernetesTokenProxy})
		desiredEnvs = append(desiredEnvs, corev1.EnvVar{Name: AzureKubernetesSNINameEnvVar, Value: m.config.AzureKubernetesSNIName})
		desiredEnvs = append(desiredEnvs, corev1.EnvVar{Name: AzureKubernetesCAFileEnvVar, Value: filepath.Join(VolumeMountPath, CAFilePath)})
		desiredEnvs = append(desiredEnvs, corev1.EnvVar{Name: AzureKubernetesCADataEnvVar, Value: m.config.AzureKubernetesCAData})
	}

	// append the ones that are not already present (only if desired env contains a non-empty value)
	for _, env := range desiredEnvs {
		if _, ok := envs[env.Name]; !ok && len(env.Value) > 0 {
			container.Env = append(container.Env, env)
		}
	}

	return container
}

func addProjectedVolumeMount(container corev1.Container, volumeName string) corev1.Container {
	volumeMount := corev1.VolumeMount{
		Name:      volumeName,
		MountPath: VolumeMountPath,
		ReadOnly:  true,
	}

	for i := range container.VolumeMounts {
		if container.VolumeMounts[i].Name == volumeName {
			container.VolumeMounts[i] = volumeMount
			return container
		}
	}

	container.VolumeMounts = append(container.VolumeMounts, volumeMount)
	return container
}

func (m *podMutator) addProjectedVolume(pod *corev1.Pod, serviceAccountTokenExpiration int64, volumeName string, podUsingCustomTokenEndpoint bool) {
	aud := m.audience
	if podUsingCustomTokenEndpoint {
		aud = m.customTokenEndpoint.audience
	}

	volume := corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			Projected: &corev1.ProjectedVolumeSource{
				Sources: []corev1.VolumeProjection{
					{
						ServiceAccountToken: &corev1.ServiceAccountTokenProjection{
							Path:              TokenFilePath,
							ExpirationSeconds: &serviceAccountTokenExpiration,
							Audience:          aud,
						},
					},
				},
			},
		},
	}

	if podUsingCustomTokenEndpoint {
		if len(m.config.AzureKubernetesCAConfigMapName) > 0 {
			volume.VolumeSource.Projected.Sources = append(volume.VolumeSource.Projected.Sources,
				corev1.VolumeProjection{
					ConfigMap: &corev1.ConfigMapProjection{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: m.config.AzureKubernetesCAConfigMapName,
						},
						Items: []corev1.KeyToPath{
							{
								Key:  "ca.crt",
								Path: CAFilePath,
							},
						},
						Optional: ptr.To(false),
					},
				},
			)
		}

		if len(m.config.AzureKubernetesCACTBSignerName) > 0 {
			volume.VolumeSource.Projected.Sources = append(volume.VolumeSource.Projected.Sources,
				corev1.VolumeProjection{
					ClusterTrustBundle: &corev1.ClusterTrustBundleProjection{
						SignerName:    ptr.To(m.config.AzureKubernetesCACTBSignerName),
						LabelSelector: m.config.AzureKubernetesCACTBLabelSelector.Value,
						Path:          CAFilePath,
						Optional:      ptr.To(false),
					},
				},
			)
		}
	}

	for i := range pod.Spec.Volumes {
		if pod.Spec.Volumes[i].Name == volumeName {
			// if the volume already exists, update it
			pod.Spec.Volumes[i] = volume
			return
		}
	}

	pod.Spec.Volumes = append(pod.Spec.Volumes, volume)
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

// serverVersionGTE returns true if v is greater than or equal to the server version.
func serverVersionGTE(discoveryClient discovery.ServerVersionInterface, v *utilversion.Version) (bool, error) {
	// check if the kubernetes version is supported
	serverVersion, err := discoveryClient.ServerVersion()
	if err != nil {
		return false, err
	}
	sv, err := utilversion.ParseSemantic(serverVersion.GitVersion)
	if err != nil {
		return false, err
	}
	return sv.AtLeast(v), nil
}

func buildVolumeName(podName string) string {
	hash := sha256.Sum256([]byte(podName))
	truncatedHash := fmt.Sprintf("%x", hash)[:63-len(ProjectedVolumeNamePrefix)] // 63 is the max length for volume names
	return fmt.Sprintf("%s%s", ProjectedVolumeNamePrefix, truncatedHash)
}
