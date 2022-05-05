package podidentity

import (
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/Azure/azure-workload-identity/pkg/cmd/podidentity/k8s"
	"github.com/Azure/azure-workload-identity/pkg/webhook"

	aadpodv1 "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var (
	trueVal   = true
	runAsRoot = int64(0)

	expectedProxyInitContainer = corev1.Container{
		Name:            proxyInitContainerName,
		Image:           proxyInitImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		SecurityContext: &corev1.SecurityContext{
			Privileged: &trueVal,
			RunAsUser:  &runAsRoot,
			Capabilities: &corev1.Capabilities{
				Add:  []corev1.Capability{"NET_ADMIN"},
				Drop: []corev1.Capability{"ALL"},
			},
		},
		Env: []corev1.EnvVar{
			{
				Name:  "PROXY_PORT",
				Value: "8000",
			},
		},
	}

	expectedProxyContainer = corev1.Container{
		Name:            proxyContainerName,
		Image:           proxyImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Args:            []string{"--log-encoder=json"},
		Ports: []corev1.ContainerPort{
			{
				Name:          "http",
				ContainerPort: 8000,
			},
		},
	}
)

func TestDetectCmdPreRunError(t *testing.T) {
	tests := []struct {
		name      string
		detectCmd *detectCmd
		errorMsg  string
	}{
		{
			name: "token expiration >= minimum token expiration",
			detectCmd: &detectCmd{
				serviceAccountTokenExpiration: 1 * time.Hour,
			},
			errorMsg: "",
		},
		{
			name: "token expiration < minimum token expiration",
			detectCmd: &detectCmd{
				serviceAccountTokenExpiration: 1 * time.Minute,
			},
			errorMsg: "--service-account-token-expiration must be greater than or equal to 1h0m0s",
		},
		{
			name: "token expiration > maximum token expiration",
			detectCmd: &detectCmd{
				serviceAccountTokenExpiration: 25 * time.Hour,
			},
			errorMsg: "--service-account-token-expiration must be less than or equal to 24h0m0s",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := test.detectCmd.prerun(); err == nil {
				t.Errorf("preRun() error is nil, want error %v", test.errorMsg)
			}
		})
	}
}

func TestAddProxyInitContainers(t *testing.T) {
	tests := []struct {
		name             string
		actualContainers []corev1.Container
		wantContainers   []corev1.Container
	}{
		{
			name:             "no init containers",
			actualContainers: nil,
			wantContainers:   []corev1.Container{expectedProxyInitContainer},
		},
		{
			name: "one init container",
			actualContainers: []corev1.Container{
				{
					Name:  "cont1",
					Image: "image1",
				},
			},
			wantContainers: []corev1.Container{
				{
					Name:  "cont1",
					Image: "image1",
				},
				expectedProxyInitContainer,
			},
		},
		{
			name: "proxy-init container already exists",
			actualContainers: []corev1.Container{
				{
					Name:  "proxy-init",
					Image: "mcr.microsoft.com/oss/azure/workload-identity/proxy-init:v0.8.0",
				},
			},
			wantContainers: []corev1.Container{
				{
					Name:  "proxy-init",
					Image: "mcr.microsoft.com/oss/azure/workload-identity/proxy-init:v0.8.0",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dc := &detectCmd{
				proxyPort: 8000,
			}
			gotContainers := dc.addProxyInitContainer(tt.actualContainers)
			if !reflect.DeepEqual(gotContainers, tt.wantContainers) {
				t.Errorf("addProxyInitContainers() = %v, want %v", gotContainers, tt.wantContainers)
			}
		})
	}
}

func TestAddProxyContainer(t *testing.T) {
	tests := []struct {
		name             string
		actualContainers []corev1.Container
		wantContainers   []corev1.Container
	}{
		{
			name:             "no containers",
			actualContainers: nil,
			wantContainers:   []corev1.Container{expectedProxyContainer},
		},
		{
			name: "one container",
			actualContainers: []corev1.Container{
				{
					Name:  "cont1",
					Image: "image1",
				},
			},
			wantContainers: []corev1.Container{
				{
					Name:  "cont1",
					Image: "image1",
				},
				expectedProxyContainer,
			},
		},
		{
			name: "proxy container already exists",
			actualContainers: []corev1.Container{
				{
					Name:  "proxy",
					Image: "mcr.microsoft.com/oss/azure/workload-identity/proxy:v0.8.0",
				},
			},
			wantContainers: []corev1.Container{
				{
					Name:  "proxy",
					Image: "mcr.microsoft.com/oss/azure/workload-identity/proxy:v0.8.0",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dc := &detectCmd{
				proxyPort: 8000,
			}
			gotContainers := dc.addProxyContainer(tt.actualContainers)
			if !reflect.DeepEqual(gotContainers, tt.wantContainers) {
				t.Errorf("addProxyContainer() = %v, want %v", gotContainers, tt.wantContainers)
			}
		})
	}
}

func TestCreateServiceAccountFileError(t *testing.T) {
	dc := &detectCmd{
		kubeClient: fake.NewClientBuilder().Build(),
	}
	if _, err := dc.createServiceAccountFile("sa", "deployment", "client-id"); err == nil {
		t.Errorf("createServiceAccountFile() error is nil, want error")
	}
}

func TestCreateServiceAccountFile(t *testing.T) {
	tests := []struct {
		name              string
		saName            string
		ownerName         string
		clientID          string
		tenantID          string
		initObjects       []client.Object
		wantedAnnotations []string
	}{
		{
			name:              "using default service account",
			saName:            "default",
			ownerName:         "deployment",
			clientID:          "client-id",
			initObjects:       []client.Object{},
			wantedAnnotations: []string{webhook.ClientIDAnnotation, webhook.ServiceAccountTokenExpiryAnnotation},
		},
		{
			name:      "using custom service account",
			saName:    "deployment-sa",
			ownerName: "deployment",
			clientID:  "client-id",
			initObjects: []client.Object{
				&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "deployment-sa",
						Namespace: "default",
					},
				},
			},
			wantedAnnotations: []string{webhook.ClientIDAnnotation, webhook.ServiceAccountTokenExpiryAnnotation},
		},
		{
			name:      "using custom service account with tenant id",
			saName:    "deployment-sa",
			ownerName: "deployment",
			clientID:  "client-id",
			tenantID:  "tenant-id",
			initObjects: []client.Object{
				&corev1.ServiceAccount{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "deployment-sa",
						Namespace: "default",
					},
				},
			},
			wantedAnnotations: []string{webhook.ClientIDAnnotation, webhook.ServiceAccountTokenExpiryAnnotation, webhook.TenantIDAnnotation},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outDir, _ := os.MkdirTemp("", "")
			defer os.RemoveAll(outDir)

			dc := &detectCmd{
				kubeClient: fake.NewClientBuilder().WithObjects(tt.initObjects...).Build(),
				namespace:  "default",
				serializer: json.NewSerializerWithOptions(
					json.DefaultMetaFactory, scheme, scheme,
					json.SerializerOptions{
						Yaml:   true,
						Pretty: true,
						Strict: true,
					},
				),
				serviceAccountTokenExpiration: 3600 * time.Second,
				tenantID:                      tt.tenantID,
				outputDir:                     outDir,
			}
			if _, err := dc.createServiceAccountFile(tt.saName, tt.ownerName, tt.clientID); err != nil {
				t.Errorf("createServiceAccountFile() error = %v, want nil", err)
			}

			saFile := dc.getServiceAccountFileName(tt.ownerName)
			if _, err := os.Stat(saFile); os.IsNotExist(err) {
				t.Errorf("createServiceAccountFile() file %s does not exist, want it to exist", saFile)
			}

			gotServiceAccountFile, err := os.ReadFile(saFile)
			if err != nil {
				t.Errorf("createServiceAccountFile() error = %v, want nil", err)
			}

			if !strings.Contains(string(gotServiceAccountFile), webhook.UseWorkloadIdentityLabel) {
				t.Errorf("createServiceAccountFile() file %s does not contain %s, want it to contain it", saFile, webhook.UseWorkloadIdentityLabel)
			}
			for _, annotation := range tt.wantedAnnotations {
				if !strings.Contains(string(gotServiceAccountFile), annotation) {
					t.Errorf("createServiceAccountFile() file %s does not contain annotation %s, want it to contain it", saFile, annotation)
				}
			}
		})
	}
}

func TestCreateResourceFile(t *testing.T) {
	testSA := &corev1.ServiceAccount{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-sa",
			Namespace: "default",
		},
	}

	tests := []struct {
		name string
		obj  client.Object
	}{
		{
			name: "deployment resource",
			obj: &appsv1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "deployment",
					Namespace: "default",
				},
				Spec: appsv1.DeploymentSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "container",
									Image: "image",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "daemon set resource",
			obj: &appsv1.DaemonSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "daemon-set",
					Namespace: "default",
				},
				Spec: appsv1.DaemonSetSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "container",
									Image: "image",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "stateful set resource",
			obj: &appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "stateful-set",
					Namespace: "default",
				},
				Spec: appsv1.StatefulSetSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "container",
									Image: "image",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "job resource",
			obj: &batchv1.Job{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "job",
					Namespace: "default",
				},
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  "container",
									Image: "image",
								},
							},
						},
					},
				},
			},
		},
		{
			name: "cron job resource",
			obj: &batchv1.CronJob{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "cron-job",
					Namespace: "default",
				},
				Spec: batchv1.CronJobSpec{
					JobTemplate: batchv1.JobTemplateSpec{
						Spec: batchv1.JobSpec{
							Template: corev1.PodTemplateSpec{
								Spec: corev1.PodSpec{
									Containers: []corev1.Container{
										{
											Name:  "container",
											Image: "image",
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
			name: "pod resource",
			obj: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pod",
					Namespace: "default",
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "container",
							Image: "image",
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			outDir, _ := os.MkdirTemp("", "")
			defer os.RemoveAll(outDir)

			dc := &detectCmd{
				kubeClient: fake.NewClientBuilder().Build(),
				namespace:  "default",
				serializer: json.NewSerializerWithOptions(
					json.DefaultMetaFactory, scheme, scheme,
					json.SerializerOptions{
						Yaml:   true,
						Pretty: true,
						Strict: true,
					},
				),
				outputDir: outDir,
			}
			localObj := k8s.NewLocalObject(tt.obj)
			if err := dc.createResourceFile(localObj, testSA); err != nil {
				t.Errorf("createResourceFile() error = %v, want nil", err)
			}

			resourceFile := dc.getResourceFileName(localObj)
			if _, err := os.Stat(resourceFile); os.IsNotExist(err) {
				t.Errorf("createResourceFile() file %s does not exist, want it to exist", resourceFile)
			}

			if _, err := os.ReadFile(resourceFile); err != nil {
				t.Errorf("createResourceFile() error = %v, want nil", err)
			}
		})
	}
}

func TestFilterAzureIdentities(t *testing.T) {
	tests := []struct {
		name                  string
		azureIdentityBindings []aadpodv1.AzureIdentityBinding
		azureIdentities       map[string]aadpodv1.AzureIdentity
		expected              map[string]aadpodv1.AzureIdentity
	}{
		{
			name: "no azure identities",
			azureIdentityBindings: []aadpodv1.AzureIdentityBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "binding-1",
						Namespace: "default",
					},
					Spec: aadpodv1.AzureIdentityBindingSpec{
						AzureIdentity: "identity-1",
					},
				},
			},
			expected: map[string]aadpodv1.AzureIdentity{},
		},
		{
			name: "invalid azureidentitybinding selector",
			azureIdentityBindings: []aadpodv1.AzureIdentityBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "binding-1",
						Namespace: "default",
					},
					Spec: aadpodv1.AzureIdentityBindingSpec{
						AzureIdentity: "identity-1",
						Selector:      "",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "binding-2",
						Namespace: "default",
					},
					Spec: aadpodv1.AzureIdentityBindingSpec{
						AzureIdentity: "identity-2",
						Selector:      "selector-2",
					},
				},
			},
			azureIdentities: map[string]aadpodv1.AzureIdentity{
				"identity-1": {
					ObjectMeta: metav1.ObjectMeta{
						Name:      "identity-1",
						Namespace: "default",
					},
				},
				"identity-2": {
					ObjectMeta: metav1.ObjectMeta{
						Name:      "identity-2",
						Namespace: "default",
					},
				},
			},
			expected: map[string]aadpodv1.AzureIdentity{
				"selector-2": {
					ObjectMeta: metav1.ObjectMeta{
						Name:      "identity-2",
						Namespace: "default",
					},
				},
			},
		},
		{
			name: "multiple azureidentitybindings with same selector",
			azureIdentityBindings: []aadpodv1.AzureIdentityBinding{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "binding-1",
						Namespace: "default",
					},
					Spec: aadpodv1.AzureIdentityBindingSpec{
						AzureIdentity: "identity-1",
						Selector:      "selector-1",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "binding-2",
						Namespace: "default",
					},
					Spec: aadpodv1.AzureIdentityBindingSpec{
						AzureIdentity: "identity-2",
						Selector:      "selector-2",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "binding-3",
						Namespace: "default",
					},
					Spec: aadpodv1.AzureIdentityBindingSpec{
						AzureIdentity: "identity-3",
						Selector:      "selector-1",
					},
				},
			},
			azureIdentities: map[string]aadpodv1.AzureIdentity{
				"identity-1": {
					ObjectMeta: metav1.ObjectMeta{
						Name:      "identity-1",
						Namespace: "default",
					},
				},
				"identity-2": {
					ObjectMeta: metav1.ObjectMeta{
						Name:      "identity-2",
						Namespace: "default",
					},
				},
				"identity-3": {
					ObjectMeta: metav1.ObjectMeta{
						Name:      "identity-3",
						Namespace: "default",
					},
				},
			},
			expected: map[string]aadpodv1.AzureIdentity{
				"selector-1": {
					ObjectMeta: metav1.ObjectMeta{
						Name:      "identity-1",
						Namespace: "default",
					},
				},
				"selector-2": {
					ObjectMeta: metav1.ObjectMeta{
						Name:      "identity-2",
						Namespace: "default",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := filterAzureIdentities(tt.azureIdentityBindings, tt.azureIdentities)
			if !reflect.DeepEqual(actual, tt.expected) {
				t.Errorf("filterAzureIdentities() = %v, want %v", actual, tt.expected)
			}
		})
	}
}
