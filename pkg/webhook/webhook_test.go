package webhook

import (
	"context"
	"path/filepath"
	"reflect"
	"testing"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	atypes "sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/Azure/aad-pod-managed-identity/pkg/config"
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
			name: "service account not annotated",
			sa: &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sa",
					Namespace: "default",
				},
			},
			expected: false,
		},
		{
			name: "service account is annotated with azure.pod.identity/use=true",
			sa: &corev1.ServiceAccount{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "sa",
					Namespace: "default",
					Labels:    map[string]string{UsePodIdentityLabel: "true"},
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
						Name:  "AZURE_TENANT_ID",
						Value: "tenantID",
					},
					{
						Name:  "TOKEN_FILE_PATH",
						Value: filepath.Join(TokenFileMountPath, TokenFilePathName),
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
						Name:  "AZURE_TENANT_ID",
						Value: "myTenantID",
					},
					{
						Name:  "TOKEN_FILE_PATH",
						Value: filepath.Join(TokenFileMountPath, TokenFilePathName),
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
						Name:  "AZURE_TENANT_ID",
						Value: "myTenantID",
					},
					{
						Name:  "TOKEN_FILE_PATH",
						Value: filepath.Join(TokenFileMountPath, TokenFilePathName),
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
						Name:  "AZURE_TENANT_ID",
						Value: "tenantID",
					},
					{
						Name:  "TOKEN_FILE_PATH",
						Value: filepath.Join(TokenFileMountPath, TokenFilePathName),
					},
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			actualContainer := addEnvironmentVariables(test.container, "clientID", "tenantID")
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
						MountPath: "/var/run/secrets/tokens",
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
						MountPath: "/var/run/secrets/tokens",
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
	serviceAccount := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sa",
			Namespace: "ns1",
			Labels:    map[string]string{UsePodIdentityLabel: "true"},
			Annotations: map[string]string{
				ClientIDAnnotation:                  "clientID",
				ServiceAccountTokenExpiryAnnotation: "4800",
			},
		},
	}

	decoder, _ := atypes.NewDecoder(runtime.NewScheme())

	m := &podMutator{
		client:  fake.NewClientBuilder().WithObjects(serviceAccount).Build(),
		config:  &config.Config{TenantID: "tenantID"},
		decoder: decoder,
	}

	raw := []byte(`{"apiVersion":"v1","kind":"Pod","metadata":{"name":"pod","namespace":"ns1"},"spec":{"containers":[{"image":"image","name":"cont1"}],"serviceAccountName":"sa"}}`)

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
