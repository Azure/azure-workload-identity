package webhook

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"monis.app/mlog"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	atypes "sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/sets"
	utilversion "k8s.io/apimachinery/pkg/util/version"
	discoveryfake "k8s.io/client-go/discovery/fake"
	kubernetesfake "k8s.io/client-go/kubernetes/fake"
	"k8s.io/utils/ptr"

	"github.com/Azure/azure-workload-identity/pkg/config"
)

var (
	serviceAccountTokenExpiry = MinServiceAccountTokenExpiration
)

func newPod(name, namespace, serviceAccountName string, labels, annotations map[string]string, hostNetwork bool) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: corev1.PodSpec{
			ServiceAccountName: serviceAccountName,
			InitContainers: []corev1.Container{
				{
					Name:  "init-container",
					Image: "init-container-image",
				},
			},
			Containers: []corev1.Container{
				{
					Name:  "container",
					Image: "image",
				},
			},
			HostNetwork: hostNetwork,
		},
	}
}

func newPodRaw(name, namespace, serviceAccountName string, labels, annotations map[string]string, hostNetwork bool) []byte {
	pod := newPod(name, namespace, serviceAccountName, labels, annotations, hostNetwork)
	raw, err := json.Marshal(pod)
	if err != nil {
		panic(err)
	}
	return raw
}

func TestGetServiceAccountTokenExpiration(t *testing.T) {
	tests := []struct {
		name               string
		pod                *corev1.Pod
		sa                 *corev1.ServiceAccount
		expectedExpiration int64
		expectedErr        bool
	}{
		{
			name: "pod token expiry annotation invalid",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "pod",
					Namespace:   "default",
					Annotations: map[string]string{ServiceAccountTokenExpiryAnnotation: "3600s"},
				},
			},
			sa: &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sa",
					Namespace: "default",
				},
			},
			expectedExpiration: 0,
			expectedErr:        true,
		},
		{
			name: "service account token expiry annotation invalid",
			pod:  &corev1.Pod{},
			sa: &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "sa",
					Namespace:   "default",
					Annotations: map[string]string{ServiceAccountTokenExpiryAnnotation: "3600s"},
				},
			},
			expectedExpiration: 0,
			expectedErr:        true,
		},
		{
			name: "invalid token expiry < 3600",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "pod",
					Namespace:   "default",
					Annotations: map[string]string{ServiceAccountTokenExpiryAnnotation: "3599"},
				},
			},
			sa: &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sa",
					Namespace: "default",
				},
			},
			expectedExpiration: 0,
			expectedErr:        true,
		},
		{
			name: "invalid token expiry > 86400",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "pod",
					Namespace:   "default",
					Annotations: map[string]string{ServiceAccountTokenExpiryAnnotation: "86401"},
				},
			},
			sa: &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sa",
					Namespace: "default",
				},
			},
			expectedExpiration: 0,
			expectedErr:        true,
		},
		{
			name: "valid token expiry defined in service account",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod",
					Namespace: "default",
				},
			},
			sa: &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "sa",
					Namespace:   "default",
					Annotations: map[string]string{ServiceAccountTokenExpiryAnnotation: "4800"},
				},
			},
			expectedExpiration: 4800,
			expectedErr:        false,
		},
		{
			name: "token expiry in pod preferred over service account token expiry",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "pod",
					Namespace:   "default",
					Annotations: map[string]string{ServiceAccountTokenExpiryAnnotation: "4000"},
				},
			},
			sa: &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "sa",
					Namespace:   "default",
					Annotations: map[string]string{ServiceAccountTokenExpiryAnnotation: "4800"},
				},
			},
			expectedExpiration: 4000,
			expectedErr:        false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			exp, err := getServiceAccountTokenExpiration(test.pod, test.sa)
			if exp != test.expectedExpiration {
				t.Fatalf("expected: %d, got: %d", test.expectedExpiration, exp)
			}
			if test.expectedErr && err == nil || !test.expectedErr && err != nil {
				t.Fatalf("expected err: %v, got: %v", test.expectedErr, err)
			}
		})
	}
}

func TestGetClientID(t *testing.T) {
	tests := []struct {
		name             string
		sa               *corev1.ServiceAccount
		expectedClientID string
	}{
		{
			name: "client id not present",
			sa: &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sa",
					Namespace: "default",
				},
			},
			expectedClientID: "",
		},
		{
			name: "client id present",
			sa: &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "sa",
					Namespace:   "default",
					Annotations: map[string]string{ClientIDAnnotation: "client-id"},
				},
			},
			expectedClientID: "client-id",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			clientID := getClientID(test.sa)
			if clientID != test.expectedClientID {
				t.Fatalf("expected: %s, got: %s", test.expectedClientID, clientID)
			}
		})
	}
}

func TestGetTenantID(t *testing.T) {
	tests := []struct {
		name             string
		sa               *corev1.ServiceAccount
		config           *config.Config
		expectedTenantID string
	}{
		{
			name: "tenant ID annotation defined",
			sa: &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "sa",
					Namespace:   "default",
					Annotations: map[string]string{TenantIDAnnotation: "tenant-id"},
				},
			},
			config:           &config.Config{},
			expectedTenantID: "tenant-id",
		},
		{
			name: "tenant ID annotation not defined, use default",
			sa: &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sa",
					Namespace: "default",
				},
			},
			config: &config.Config{
				TenantID: "tenant-id",
			},
			expectedTenantID: "tenant-id",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			tenantID := getTenantID(test.sa, test.config)
			if tenantID != test.expectedTenantID {
				t.Fatalf("expected: %s, got: %s", test.expectedTenantID, tenantID)
			}
		})
	}
}

func TestGetSkipContainers(t *testing.T) {
	tests := []struct {
		name                   string
		pod                    *corev1.Pod
		expectedSkipContainers sets.Set[string]
	}{
		{
			name: "no skip containers defined",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod",
					Namespace: "default",
				},
			},
			expectedSkipContainers: nil,
		},
		{
			name: "one skip container defined",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "pod",
					Namespace:   "default",
					Annotations: map[string]string{SkipContainersAnnotation: "container1"},
				},
			},
			expectedSkipContainers: sets.New("container1"),
		},
		{
			name: "multiple skip containers defined delimited by ;",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "pod",
					Namespace:   "default",
					Annotations: map[string]string{SkipContainersAnnotation: "container1;container2"},
				},
			},
			expectedSkipContainers: sets.New("container1", "container2"),
		},
		{
			name: "multiple skip containers defined with extra space",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "pod",
					Namespace:   "default",
					Annotations: map[string]string{SkipContainersAnnotation: "container1; container2"},
				},
			},
			expectedSkipContainers: sets.New("container1", "container2"),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			skipContainers := getSkipContainers(test.pod)
			if !reflect.DeepEqual(skipContainers, test.expectedSkipContainers) {
				t.Fatalf("expected: %v, got: %v", test.expectedSkipContainers, skipContainers)
			}
		})
	}
}

func TestAddProjectedServiceAccountTokenVolume(t *testing.T) {
	tests := []struct {
		name           string
		pod            *corev1.Pod
		expectedVolume []corev1.Volume
	}{
		{
			name: "no volumes in the pod",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod",
					Namespace: "default",
				},
			},
			expectedVolume: []corev1.Volume{
				{
					Name: TokenFilePathName,
					VolumeSource: corev1.VolumeSource{
						Projected: &corev1.ProjectedVolumeSource{
							Sources: []corev1.VolumeProjection{
								{
									ServiceAccountToken: &corev1.ServiceAccountTokenProjection{
										Path:              TokenFilePathName,
										ExpirationSeconds: &serviceAccountTokenExpiry,
										Audience:          DefaultAudience,
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "azure-identity-token projected volume already exists",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod",
					Namespace: "default",
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: TokenFilePathName,
							VolumeSource: corev1.VolumeSource{
								Projected: &corev1.ProjectedVolumeSource{
									Sources: []corev1.VolumeProjection{
										{
											ServiceAccountToken: &corev1.ServiceAccountTokenProjection{
												Path:              TokenFilePathName,
												ExpirationSeconds: &serviceAccountTokenExpiry,
												Audience:          DefaultAudience,
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expectedVolume: []corev1.Volume{
				{
					Name: TokenFilePathName,
					VolumeSource: corev1.VolumeSource{
						Projected: &corev1.ProjectedVolumeSource{
							Sources: []corev1.VolumeProjection{
								{
									ServiceAccountToken: &corev1.ServiceAccountTokenProjection{
										Path:              TokenFilePathName,
										ExpirationSeconds: &serviceAccountTokenExpiry,
										Audience:          DefaultAudience,
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "existing projected service account token volume not affected",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod",
					Namespace: "default",
				},
				Spec: corev1.PodSpec{
					Volumes: []corev1.Volume{
						{
							Name: TokenFilePathName,
							VolumeSource: corev1.VolumeSource{
								Projected: &corev1.ProjectedVolumeSource{
									Sources: []corev1.VolumeProjection{
										{
											ServiceAccountToken: &corev1.ServiceAccountTokenProjection{
												Path:              "my-projected-volume",
												ExpirationSeconds: &serviceAccountTokenExpiry,
												Audience:          "aud",
											},
										},
									},
								},
							},
						},
					},
				},
			},
			expectedVolume: []corev1.Volume{
				{
					Name: TokenFilePathName,
					VolumeSource: corev1.VolumeSource{
						Projected: &corev1.ProjectedVolumeSource{
							Sources: []corev1.VolumeProjection{
								{
									ServiceAccountToken: &corev1.ServiceAccountTokenProjection{
										Path:              "my-projected-volume",
										ExpirationSeconds: &serviceAccountTokenExpiry,
										Audience:          "aud",
									},
								},
							},
						},
					},
				},
				{
					Name: TokenFilePathName,
					VolumeSource: corev1.VolumeSource{
						Projected: &corev1.ProjectedVolumeSource{
							Sources: []corev1.VolumeProjection{
								{
									ServiceAccountToken: &corev1.ServiceAccountTokenProjection{
										Path:              TokenFilePathName,
										ExpirationSeconds: &serviceAccountTokenExpiry,
										Audience:          DefaultAudience,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			addProjectedServiceAccountTokenVolume(test.pod, serviceAccountTokenExpiry, DefaultAudience)

			if !reflect.DeepEqual(test.pod.Spec.Volumes, test.expectedVolume) {
				t.Fatalf("expected: %v, got: %v", test.pod.Spec.Volumes, test.expectedVolume)
			}
		})
	}
}

func TestAddEnvironmentVariables(t *testing.T) {
	tests := []struct {
		name              string
		container         corev1.Container
		expectedContainer corev1.Container
	}{
		{
			name: "environment variables added to container",
			container: corev1.Container{
				Name:  "cont1",
				Image: "image",
			},
			expectedContainer: corev1.Container{
				Name:  "cont1",
				Image: "image",
				// this test uses literals instead of constants for env var
				// names so that it will fail if the constant values change
				Env: []corev1.EnvVar{
					{
						Name:  "AZURE_CLIENT_ID",
						Value: "clientID",
					},
					{
						Name:  "AZURE_TENANT_ID",
						Value: "tenantID",
					},
					{
						Name:  "AZURE_FEDERATED_TOKEN_FILE",
						Value: filepath.Join(TokenFileMountPath, TokenFilePathName),
					},
					{
						Name:  "AZURE_AUTHORITY_HOST",
						Value: "https://login.microsoftonline.com/",
					},
				},
			},
		},
		{
			name: "existing environment variables not replaced",
			container: corev1.Container{
				Name:  "cont1",
				Image: "image",
				Env: []corev1.EnvVar{
					{
						Name:  AzureClientIDEnvVar,
						Value: "myClientID",
					},
					{
						Name:  AzureTenantIDEnvVar,
						Value: "myTenantID",
					},
					{
						Name:  AzureFederatedTokenFileEnvVar,
						Value: filepath.Join(TokenFileMountPath, TokenFilePathName),
					},
					{
						Name:  AzureAuthorityHostEnvVar,
						Value: "https://login.microsoftonline.com/",
					},
				},
			},
			expectedContainer: corev1.Container{
				Name:  "cont1",
				Image: "image",
				Env: []corev1.EnvVar{
					{
						Name:  AzureClientIDEnvVar,
						Value: "myClientID",
					},
					{
						Name:  AzureTenantIDEnvVar,
						Value: "myTenantID",
					},
					{
						Name:  AzureFederatedTokenFileEnvVar,
						Value: filepath.Join(TokenFileMountPath, TokenFilePathName),
					},
					{
						Name:  AzureAuthorityHostEnvVar,
						Value: "https://login.microsoftonline.com/",
					},
				},
			},
		},
		{
			name: "environment variables added to existing list",
			container: corev1.Container{
				Name:  "cont1",
				Image: "image",
				Env: []corev1.EnvVar{
					{
						Name:  "MY_ENV_VAR",
						Value: "test",
					},
				},
			},
			expectedContainer: corev1.Container{
				Name:  "cont1",
				Image: "image",
				Env: []corev1.EnvVar{
					{
						Name:  "MY_ENV_VAR",
						Value: "test",
					},
					{
						Name:  "AZURE_CLIENT_ID",
						Value: "clientID",
					},
					{
						Name:  AzureTenantIDEnvVar,
						Value: "tenantID",
					},
					{
						Name:  AzureFederatedTokenFileEnvVar,
						Value: filepath.Join(TokenFileMountPath, TokenFilePathName),
					},
					{
						Name:  AzureAuthorityHostEnvVar,
						Value: "https://login.microsoftonline.com/",
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actualContainer := addEnvironmentVariables(test.container, "clientID", "tenantID", "https://login.microsoftonline.com/")
			if !reflect.DeepEqual(actualContainer, test.expectedContainer) {
				t.Fatalf("expected: %v, got: %v", test.expectedContainer, actualContainer)
			}
		})
	}

	t.Run("environment variables are not added when empty", func(t *testing.T) {
		container := corev1.Container{
			Name:  "cont1",
			Image: "image",
		}

		expectedContainer := corev1.Container{
			Name:  "cont1",
			Image: "image",
			Env: []corev1.EnvVar{
				{
					Name:  AzureFederatedTokenFileEnvVar,
					Value: filepath.Join(TokenFileMountPath, TokenFilePathName),
				},
			},
		}

		actualContainer := addEnvironmentVariables(container, "", "", "")
		if !reflect.DeepEqual(actualContainer, expectedContainer) {
			t.Fatalf("expected: %v, got: %v", expectedContainer, actualContainer)
		}
	})
}

func TestAddProjectServiceAccountTokenVolumeMount(t *testing.T) {
	tests := []struct {
		name              string
		container         corev1.Container
		expectedContainer corev1.Container
	}{
		{
			name: "volume mount added to container",
			container: corev1.Container{
				Name:  "cont1",
				Image: "image",
			},
			expectedContainer: corev1.Container{
				Name:  "cont1",
				Image: "image",
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      TokenFilePathName,
						MountPath: TokenFileMountPath,
						ReadOnly:  true,
					},
				},
			},
		},
		{
			name: "volume mount with name already exists, so skipped",
			container: corev1.Container{
				Name:  "cont1",
				Image: "image",
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      TokenFilePathName,
						MountPath: "mountPath",
					},
				},
			},
			expectedContainer: corev1.Container{
				Name:  "cont1",
				Image: "image",
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      TokenFilePathName,
						MountPath: "mountPath",
					},
				},
			},
		},
		{
			name: "volume mount added to existing volume mounts for container",
			container: corev1.Container{
				Name:  "cont1",
				Image: "image",
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      "my-volume-mount",
						MountPath: "/var/run/pods",
					},
				},
			},
			expectedContainer: corev1.Container{
				Name:  "cont1",
				Image: "image",
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      "my-volume-mount",
						MountPath: "/var/run/pods",
					},
					{
						Name:      TokenFilePathName,
						MountPath: TokenFileMountPath,
						ReadOnly:  true,
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actualContainer := addProjectedTokenVolumeMount(test.container)
			if !reflect.DeepEqual(actualContainer, test.expectedContainer) {
				t.Fatalf("expected: %v, got: %v", test.expectedContainer, actualContainer)
			}
		})
	}
}

func TestHandle(t *testing.T) {
	serviceAccounts := []client.Object{}
	for _, name := range []string{"default", "sa"} {
		serviceAccounts = append(serviceAccounts, &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: "ns1",
				Annotations: map[string]string{
					ClientIDAnnotation:                  "clientID",
					ServiceAccountTokenExpiryAnnotation: "4800",
				},
			},
		})
	}

	decoder := atypes.NewDecoder(runtime.NewScheme())

	tests := []struct {
		name          string
		rawPod        []byte
		clientObjects []client.Object
		readerObjects []client.Object
	}{
		{
			name:          "service account in cache",
			rawPod:        newPodRaw("pod", "ns1", "sa", nil, nil, false),
			clientObjects: serviceAccounts,
			readerObjects: nil,
		},
		{
			name:          "service account not in cache",
			rawPod:        newPodRaw("pod", "ns1", "sa", nil, nil, false),
			clientObjects: nil,
			readerObjects: serviceAccounts,
		},
		{
			name:          "default service account in cache",
			rawPod:        newPodRaw("pod", "ns1", "", nil, nil, false),
			clientObjects: serviceAccounts,
			readerObjects: nil,
		},
		{
			name:          "default service account not in cache",
			rawPod:        newPodRaw("pod", "ns1", "", nil, nil, false),
			clientObjects: nil,
			readerObjects: serviceAccounts,
		},
		{
			name:          "pod has the required label, no warnings",
			rawPod:        newPodRaw("pod", "ns1", "sa", map[string]string{UseWorkloadIdentityLabel: "true"}, nil, false),
			clientObjects: serviceAccounts,
			readerObjects: nil,
		},
		{
			name:          "pod has the required label, restart policy in init container",
			rawPod:        []byte(`{"metadata":{"name":"pod","namespace":"ns1","creationTimestamp":null,"labels":{"azure.workload.identity/use":"true"}},"spec":{"initContainers":[{"name":"init-container","image":"init-container-image","restartPolicy":"Always"}],"containers":[{"name":"container","image":"image","resources":{}}]}}`),
			clientObjects: serviceAccounts,
			readerObjects: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := registerMetrics(); err != nil {
				t.Fatalf("failed to register metrics: %v", err)
			}

			m := &podMutator{
				client:  fake.NewClientBuilder().WithObjects(test.clientObjects...).Build(),
				reader:  fake.NewClientBuilder().WithObjects(test.readerObjects...).Build(),
				config:  &config.Config{TenantID: "tenantID"},
				decoder: decoder,
			}

			req := atypes.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Kind: metav1.GroupVersionKind{
						Group:   "",
						Version: "v1",
						Kind:    "Pod",
					},
					Object:    runtime.RawExtension{Raw: test.rawPod},
					Namespace: "ns1",
					Operation: admissionv1.Create,
				},
			}

			resp := m.Handle(context.Background(), req)
			if !resp.Allowed {
				t.Fatalf("expected to be allowed")
			}
			for _, patch := range resp.Patches {
				if patch.Operation == "remove" {
					t.Errorf("expected no remove patches, got: %v", patch)
				}
			}
		})
	}
}

func TestGetAzureAuthorityHost(t *testing.T) {
	tests := []struct {
		name        string
		config      *config.Config
		want        string
		expectedErr bool
	}{
		{
			name:   "default azure environment",
			config: &config.Config{},
			want:   "https://login.microsoftonline.com/",
		},
		{
			name: "public cloud",
			config: &config.Config{
				Cloud: "AzurePublicCloud",
			},
			want: "https://login.microsoftonline.com/",
		},
		{
			name: "us government cloud",
			config: &config.Config{
				Cloud: "AzureUSGovernmentCloud",
			},
			want: "https://login.microsoftonline.us/",
		},
		{
			name: "china cloud",
			config: &config.Config{
				Cloud: "AzureChinaCloud",
			},
			want: "https://login.chinacloudapi.cn/",
		},
		{
			name: "german cloud",
			config: &config.Config{
				Cloud: "AzureGermanCloud",
			},
			want: "https://login.microsoftonline.de/",
		},
		{
			name: "invalid cloud name",
			config: &config.Config{
				Cloud: "InvalidCloud",
			},
			want:        "",
			expectedErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := getAzureAuthorityHost(test.config)
			if test.expectedErr && err == nil || !test.expectedErr && err != nil {
				t.Errorf("expected err: %v, got: %v", test.expectedErr, err)
			}
			if got != test.want {
				t.Errorf("getAzureAuthorityHost() = %v, want %v", got, test.want)
			}
		})
	}
}

func TestMutateContainers(t *testing.T) {
	azureAuthorityHost := "https://login.microsoftonline.com/"
	azureClientID := "client-id"
	azureTenantID := "tenant-id"

	tests := []struct {
		name               string
		containers         []corev1.Container
		skipContainers     sets.Set[string]
		expectedContainers []corev1.Container
	}{{
		name:               "no containers",
		containers:         []corev1.Container{},
		expectedContainers: []corev1.Container{},
	}, {
		name: "one container",
		containers: []corev1.Container{{
			Name:  "my-container",
			Image: "my-image",
		}},
		expectedContainers: []corev1.Container{{
			Name:  "my-container",
			Image: "my-image",
			Env: []corev1.EnvVar{
				{
					Name:  AzureClientIDEnvVar,
					Value: azureClientID,
				},
				{
					Name:  AzureTenantIDEnvVar,
					Value: azureTenantID,
				},
				{
					Name:  AzureFederatedTokenFileEnvVar,
					Value: filepath.Join(TokenFileMountPath, TokenFilePathName),
				},
				{
					Name:  AzureAuthorityHostEnvVar,
					Value: azureAuthorityHost,
				},
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      TokenFilePathName,
					MountPath: TokenFileMountPath,
					ReadOnly:  true,
				},
			},
		}},
	}, {
		name: "one container and one skip container",
		containers: []corev1.Container{{
			Name:  "my-container",
			Image: "my-image",
		}, {
			Name:  "skip-container",
			Image: "skip-image",
		}},
		skipContainers: sets.New("skip-container"),
		expectedContainers: []corev1.Container{{
			Name:  "my-container",
			Image: "my-image",
			Env: []corev1.EnvVar{
				{
					Name:  AzureClientIDEnvVar,
					Value: azureClientID,
				},
				{
					Name:  AzureTenantIDEnvVar,
					Value: azureTenantID,
				},
				{
					Name:  AzureFederatedTokenFileEnvVar,
					Value: filepath.Join(TokenFileMountPath, TokenFilePathName),
				},
				{
					Name:  AzureAuthorityHostEnvVar,
					Value: azureAuthorityHost,
				},
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      TokenFilePathName,
					MountPath: TokenFileMountPath,
					ReadOnly:  true,
				},
			},
		}, {
			Name:  "skip-container",
			Image: "skip-image",
		}},
	}}

	decoder := atypes.NewDecoder(runtime.NewScheme())
	m := &podMutator{
		client:             fake.NewClientBuilder().WithObjects().Build(),
		reader:             fake.NewClientBuilder().WithObjects().Build(),
		config:             &config.Config{TenantID: azureTenantID},
		decoder:            decoder,
		azureAuthorityHost: azureAuthorityHost,
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			containers := m.mutateContainers(test.containers, azureClientID, azureTenantID, test.skipContainers)
			if !reflect.DeepEqual(containers, test.expectedContainers) {
				t.Errorf("expected: %v, got: %v", test.expectedContainers, test.containers)
			}
		})
	}
}

func TestInjectProxyInitContainer(t *testing.T) {
	proxyPort := int32(8080)
	ProxyImageRegistry = "my.proxy-image-registry.io/azwi"
	ProxyImageVersion = "v1.0.0"
	imageURL := fmt.Sprintf("%s/%s:%s", ProxyImageRegistry, ProxyInitImageName, ProxyImageVersion)
	proxyInitContainer := corev1.Container{
		Name:            ProxyInitContainerName,
		Image:           imageURL,
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
	}

	tests := []struct {
		name               string
		containers         []corev1.Container
		expectedContainers []corev1.Container
	}{
		{
			name:               "no init containers",
			containers:         []corev1.Container{},
			expectedContainers: []corev1.Container{proxyInitContainer},
		},
		{
			name:               "proxy init container manually injected",
			containers:         []corev1.Container{proxyInitContainer},
			expectedContainers: []corev1.Container{proxyInitContainer},
		},
		{
			name: "inject proxy init container to existing init containers",
			containers: []corev1.Container{
				{
					Name:  "my-container",
					Image: "my-image",
				},
			},
			expectedContainers: []corev1.Container{
				{
					Name:  "my-container",
					Image: "my-image",
				},
				proxyInitContainer,
			},
		},
	}

	m := &podMutator{proxyInitImage: imageURL}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			containers := m.injectProxyInitContainer(test.containers, proxyPort)
			if !reflect.DeepEqual(containers, test.expectedContainers) {
				t.Errorf("expected: %v, got: %v", test.expectedContainers, test.containers)
			}
		})
	}
}

func TestInjectProxySidecarContainer(t *testing.T) {
	origLogLevel := currentLogLevel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // we do not need log flushing for this test

	if err := mlog.ValidateAndSetLogLevelAndFormatGlobally(ctx, mlog.LogSpec{
		Level:  mlog.LevelDebug, // this is the log level we expect the proxy to be running at in this test
		Format: mlog.FormatJSON,
	}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := mlog.ValidateAndSetLogLevelAndFormatGlobally(ctx, mlog.LogSpec{
			Level:  mlog.LogLevel(origLogLevel),
			Format: mlog.FormatJSON,
		}); err != nil {
			t.Fatal(err)
		}
	})

	proxyPort := int32(8081)
	ProxyImageRegistry = "my.proxy-image-registry.io/azwi"
	ProxyImageVersion = "v1.0.0"
	imageURL := fmt.Sprintf("%s/%s:%s", ProxyImageRegistry, ProxySidecarImageName, ProxyImageVersion)
	proxySidecarContainer := corev1.Container{
		Name:            ProxySidecarContainerName,
		Image:           imageURL,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Args: []string{
			fmt.Sprintf("--proxy-port=%d", proxyPort),
			"--log-level=debug",
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
						"--log-level=debug",
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
	}

	proxyNativeSidecarContainer := proxySidecarContainer
	proxyNativeSidecarContainer.RestartPolicy = ptr.To(corev1.ContainerRestartPolicyAlways)

	tests := []struct {
		name               string
		containers         []corev1.Container
		expectedContainers []corev1.Container
		restartPolicy      *corev1.ContainerRestartPolicy
	}{
		{
			name:               "no containers",
			containers:         []corev1.Container{},
			expectedContainers: []corev1.Container{proxySidecarContainer},
			restartPolicy:      nil,
		},
		{
			name:               "proxy sidecar container manually injected",
			containers:         []corev1.Container{proxySidecarContainer},
			expectedContainers: []corev1.Container{proxySidecarContainer},
			restartPolicy:      nil,
		},
		{
			name: "inject proxy sidecar container to existing containers",
			containers: []corev1.Container{
				{
					Name:  "my-container",
					Image: "my-image",
				},
			},
			expectedContainers: []corev1.Container{
				proxySidecarContainer,
				{
					Name:  "my-container",
					Image: "my-image",
				},
			},
			restartPolicy: nil,
		},
		{
			name: "inject proxy native sidecar container to existing containers when restartPolicy is set",
			containers: []corev1.Container{
				{
					Name:  "my-container",
					Image: "my-image",
				},
			},
			expectedContainers: []corev1.Container{
				proxyNativeSidecarContainer,
				{
					Name:  "my-container",
					Image: "my-image",
				},
			},
			restartPolicy: ptr.To(corev1.ContainerRestartPolicyAlways),
		},
	}

	m := &podMutator{proxyImage: imageURL}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			containers := m.injectProxySidecarContainer(test.containers, proxyPort, test.restartPolicy)
			if !reflect.DeepEqual(containers, test.expectedContainers) {
				t.Errorf("expected: %v, got: %v", test.expectedContainers, containers)
			}
		})
	}
}

func TestShouldInjectProxySidecar(t *testing.T) {
	tests := []struct {
		name     string
		pod      *corev1.Pod
		expected bool
	}{
		{
			name: "pod not annotated",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pod",
				},
			},
			expected: false,
		},
		{
			name: "pod is annotated with azure.workload.identity/inject-proxy-sidecar=true",
			pod: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name: "pod",
					Annotations: map[string]string{
						InjectProxySidecarAnnotation: "true",
					},
				},
			},
			expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := shouldInjectProxySidecar(test.pod)
			if actual != test.expected {
				t.Fatalf("expected: %v, got: %v", test.expected, actual)
			}
		})
	}
}

func TestGetProxyPort(t *testing.T) {
	type args struct {
		pod *corev1.Pod
	}
	tests := []struct {
		name    string
		args    args
		want    int32
		wantErr bool
	}{
		{

			name: "pod not annotated",
			args: args{
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name: "pod",
					},
				},
			},
			want:    DefaultProxySidecarPort,
			wantErr: false,
		},
		{
			name: "pod has no azure.workload.identity/proxy-sidecar-port annotation",
			args: args{
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name: "pod",
						Annotations: map[string]string{
							"test": "test",
						},
					},
				},
			},
			want:    DefaultProxySidecarPort,
			wantErr: false,
		},
		{
			name: "pod is annotated with azure.workload.identity/proxy-sidecar-port=8080",
			args: args{
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name: "pod",
						Annotations: map[string]string{
							ProxySidecarPortAnnotation: "8080",
						},
					},
				},
			},
			want:    8080,
			wantErr: false,
		},
		{
			name: "pod is annotated with azure.workload.identity/proxy-sidecar-port=invalid",
			args: args{
				pod: &corev1.Pod{
					ObjectMeta: metav1.ObjectMeta{
						Name: "pod",
						Annotations: map[string]string{
							ProxySidecarPortAnnotation: "invalid",
						},
					},
				},
			},
			want:    0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := getProxyPort(tt.args.pod)
			if (err != nil) != tt.wantErr {
				t.Errorf("getProxyPort() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("getProxyPort() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHandleError(t *testing.T) {
	serviceAccounts := []client.Object{}
	for _, name := range []string{"default", "sa"} {
		serviceAccounts = append(serviceAccounts, &corev1.ServiceAccount{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: "ns1",
				Annotations: map[string]string{
					ClientIDAnnotation:                  "clientID",
					ServiceAccountTokenExpiryAnnotation: "4800",
				},
			},
		})
	}

	decoder := atypes.NewDecoder(runtime.NewScheme())

	tests := []struct {
		name          string
		object        runtime.RawExtension
		clientObjects []client.Object
		expectedErr   string
	}{
		{
			name:          "failed to decode pod",
			object:        runtime.RawExtension{Raw: []byte("invalid")},
			clientObjects: serviceAccounts,
			expectedErr:   `couldn't get version/kind`,
		},
		{
			name:        "service account not found",
			object:      runtime.RawExtension{Raw: newPodRaw("pod", "ns1", "sa", map[string]string{UseWorkloadIdentityLabel: "true"}, nil, true)},
			expectedErr: `serviceaccounts "sa" not found`,
		},
		{
			name: "pod has host network",
			object: runtime.RawExtension{Raw: newPodRaw("pod", "ns1", "sa",
				map[string]string{UseWorkloadIdentityLabel: "true"}, map[string]string{InjectProxySidecarAnnotation: "true"}, true)},
			clientObjects: serviceAccounts,
			expectedErr:   "hostNetwork is set to true, cannot inject proxy sidecar",
		},
		{
			name: "invalid proxy port",
			object: runtime.RawExtension{Raw: newPodRaw("pod", "ns1", "sa", map[string]string{UseWorkloadIdentityLabel: "true"},
				map[string]string{InjectProxySidecarAnnotation: "true", ProxySidecarPortAnnotation: "invalid"}, false)},
			clientObjects: serviceAccounts,
			expectedErr:   `failed to parse proxy sidecar port: strconv.ParseInt: parsing "invalid": invalid syntax`,
		},
		{
			name: "invalid sa token expiry",
			object: runtime.RawExtension{Raw: newPodRaw("pod", "ns1", "sa", map[string]string{UseWorkloadIdentityLabel: "true"},
				map[string]string{ServiceAccountTokenExpiryAnnotation: "invalid"}, false)},
			clientObjects: serviceAccounts,
			expectedErr:   `strconv.ParseInt: parsing "invalid": invalid syntax`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := registerMetrics(); err != nil {
				t.Fatalf("failed to register metrics: %v", err)
			}

			m := &podMutator{
				client:  fake.NewClientBuilder().WithObjects(test.clientObjects...).Build(),
				reader:  fake.NewClientBuilder().WithObjects().Build(),
				config:  &config.Config{TenantID: "tenantID"},
				decoder: decoder,
			}

			req := atypes.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Kind: metav1.GroupVersionKind{
						Group:   "",
						Version: "v1",
						Kind:    "Pod",
					},
					Object:    test.object,
					Namespace: "ns1",
					Operation: admissionv1.Create,
				},
			}

			resp := m.Handle(context.Background(), req)
			if resp.Allowed {
				t.Fatalf("expected to be denied")
			}
			if !strings.Contains(resp.Result.Message, test.expectedErr) {
				t.Fatalf("expected error to contain: %v, got: %v", test.expectedErr, resp.Result.Message)
			}
		})
	}
}

func TestServerVersionGTE(t *testing.T) {
	tests := []struct {
		name          string
		serverVersion *utilversion.Version
		minVersion    *utilversion.Version
		want          bool
		wantErr       bool
	}{
		{
			name:          "Exact match",
			serverVersion: utilversion.MustParseGeneric("1.20.0"),
			minVersion:    utilversion.MajorMinor(1, 20),
			want:          true,
		},
		{
			name:          "Higher major version",
			serverVersion: utilversion.MustParseGeneric("2.0.0"),
			minVersion:    utilversion.MajorMinor(1, 25),
			want:          true,
		},
		{
			name:          "Higher minor version",
			serverVersion: utilversion.MustParseGeneric("1.25.0"),
			minVersion:    utilversion.MajorMinor(1, 20),
			want:          true,
		},
		{
			name:          "Lower minor version",
			serverVersion: utilversion.MustParseGeneric("1.18.0"),
			minVersion:    utilversion.MajorMinor(1, 20),
			want:          false,
		},
		{
			name:          "Lower major version",
			serverVersion: utilversion.MustParseGeneric("0.25.0"),
			minVersion:    utilversion.MajorMinor(1, 20),
			want:          false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			discoveryClient := kubernetesfake.NewClientset().Discovery()
			discoveryClient.(*discoveryfake.FakeDiscovery).FakedServerVersion = tt.serverVersion.Info()

			got, err := serverVersionGTE(discoveryClient, tt.minVersion)
			if (err != nil) != tt.wantErr {
				t.Errorf("isSupportedKubernetesVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("isSupportedKubernetesVersion() = %v, want %v", got, tt.want)
			}
		})
	}
}
