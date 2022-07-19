package webhook

import (
	"context"
	"fmt"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	atypes "sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/Azure/azure-workload-identity/pkg/config"
)

var (
	serviceAccountTokenExpiry = MinServiceAccountTokenExpiration
)

func TestIsServiceAccountAnnotated(t *testing.T) {
	tests := []struct {
		name     string
		sa       *corev1.ServiceAccount
		expected bool
	}{
		{
			name: "service account not labeled",
			sa: &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sa",
					Namespace: "default",
				},
			},
			expected: false,
		},
		{
			name: "service account is labeled with azure.workload.identity/use=true",
			sa: &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sa",
					Namespace: "default",
					Labels:    map[string]string{UseWorkloadIdentityLabel: "true"},
				},
			},
			expected: true,
		},
		{
			name: "service account is annotated with azure.workload.identity/use=true",
			sa: &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:        "sa",
					Namespace:   "default",
					Annotations: map[string]string{UseWorkloadIdentityLabel: "true"},
				},
			},
			expected: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actual := isServiceAccountAnnotated(test.sa)
			if actual != test.expected {
				t.Fatalf("expected: %v, got: %v", test.expected, actual)
			}
		})
	}
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
		expectedSkipContainers map[string]struct{}
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
			expectedSkipContainers: map[string]struct{}{"container1": {}},
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
			expectedSkipContainers: map[string]struct{}{"container1": {}, "container2": {}},
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
			expectedSkipContainers: map[string]struct{}{"container1": {}, "container2": {}},
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
			err := addProjectedServiceAccountTokenVolume(test.pod, serviceAccountTokenExpiry, DefaultAudience)
			if err != nil {
				t.Fatalf("expected err to be nil, got: %v", err)
			}
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
				Env: []corev1.EnvVar{
					{
						Name:  AzureClientIDEnvVar,
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
				Labels:    map[string]string{UseWorkloadIdentityLabel: "true"},
				Annotations: map[string]string{
					ClientIDAnnotation:                  "clientID",
					ServiceAccountTokenExpiryAnnotation: "4800",
				},
			},
		})
	}

	decoder, _ := atypes.NewDecoder(runtime.NewScheme())

	tests := []struct {
		name               string
		serviceAccountName string
		clientObjects      []client.Object
		readerObjects      []client.Object
	}{
		{
			name:               "service account in cache",
			serviceAccountName: "sa",
			clientObjects:      serviceAccounts,
			readerObjects:      nil,
		},
		{
			name:               "service account not in cache",
			serviceAccountName: "sa",
			clientObjects:      nil,
			readerObjects:      serviceAccounts,
		},
		{
			name:          "default service account in cache",
			clientObjects: serviceAccounts,
			readerObjects: nil,
		},
		{
			name:          "default service account not in cache",
			clientObjects: nil,
			readerObjects: serviceAccounts,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			m := &podMutator{
				client:  fake.NewClientBuilder().WithObjects(test.clientObjects...).Build(),
				reader:  fake.NewClientBuilder().WithObjects(test.readerObjects...).Build(),
				config:  &config.Config{TenantID: "tenantID"},
				decoder: decoder,
			}

			raw := []byte(fmt.Sprintf(`{"apiVersion":"v1","kind":"Pod","metadata":{"name":"pod","namespace":"ns1"},"spec":{"initContainers":[{"image":"image","name":"cont1"}],"containers":[{"image":"image","name":"cont1"}],"serviceAccountName":"%s"}}`, test.serviceAccountName))

			req := atypes.Request{
				AdmissionRequest: admissionv1.AdmissionRequest{
					Kind: metav1.GroupVersionKind{
						Group:   "",
						Version: "v1",
						Kind:    "Pod",
					},
					Object:    runtime.RawExtension{Raw: raw},
					Namespace: "ns1",
					Operation: admissionv1.Create,
				},
			}

			resp := m.Handle(context.Background(), req)
			if !resp.Allowed {
				t.Fatalf("expected to be allowed")
			}
		})
	}
}

func TestAddProjectedSecretVolume(t *testing.T) {
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
									Secret: &corev1.SecretProjection{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: "localtoken-sa",
										},
										Items: []corev1.KeyToPath{
											{
												Key:  "token",
												Path: TokenFilePathName,
											},
										},
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
											Secret: &corev1.SecretProjection{
												LocalObjectReference: corev1.LocalObjectReference{
													Name: "localtoken-sa",
												},
												Items: []corev1.KeyToPath{
													{
														Key:  "token",
														Path: TokenFilePathName,
													},
												},
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
									Secret: &corev1.SecretProjection{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: "localtoken-sa",
										},
										Items: []corev1.KeyToPath{
											{
												Key:  "token",
												Path: TokenFilePathName,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
		{
			name: "existing projected secret volume not affected",
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
											Secret: &corev1.SecretProjection{
												LocalObjectReference: corev1.LocalObjectReference{
													Name: "my-secret",
												},
												Items: []corev1.KeyToPath{
													{
														Key:  "username",
														Path: "username",
													},
												},
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
									Secret: &corev1.SecretProjection{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: "my-secret",
										},
										Items: []corev1.KeyToPath{
											{
												Key:  "username",
												Path: "username",
											},
										},
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
									Secret: &corev1.SecretProjection{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: "localtoken-sa",
										},
										Items: []corev1.KeyToPath{
											{
												Key:  "token",
												Path: TokenFilePathName,
											},
										},
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
			err := addProjectedSecretVolume(test.pod, &config.Config{}, "localtoken-sa")
			if err != nil {
				t.Fatalf("expected err to be nil, got: %v", err)
			}
			if !reflect.DeepEqual(test.pod.Spec.Volumes, test.expectedVolume) {
				t.Fatalf("expected: %v, got: %v", test.pod.Spec.Volumes, test.expectedVolume)
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
		skipContainers     map[string]struct{}
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
		skipContainers: map[string]struct{}{
			"skip-container": {},
		},
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

	decoder, _ := atypes.NewDecoder(runtime.NewScheme())
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
			Privileged: pointer.BoolPtr(true),
			RunAsUser:  pointer.Int64Ptr(0),
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

	m := &podMutator{}
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
					},
				},
			},
		},
	}

	tests := []struct {
		name               string
		containers         []corev1.Container
		expectedContainers []corev1.Container
	}{
		{
			name:               "no containers",
			containers:         []corev1.Container{},
			expectedContainers: []corev1.Container{proxySidecarContainer},
		},
		{
			name:               "proxy sidecar container manually injected",
			containers:         []corev1.Container{proxySidecarContainer},
			expectedContainers: []corev1.Container{proxySidecarContainer},
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
				{
					Name:  "my-container",
					Image: "my-image",
				},
				proxySidecarContainer,
			},
		},
	}

	m := &podMutator{}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			containers := m.injectProxySidecarContainer(test.containers, proxyPort)
			if !reflect.DeepEqual(containers, test.expectedContainers) {
				t.Errorf("expected: %v, got: %v", test.expectedContainers, test.containers)
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
