package podidentity

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	aadpodv1 "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"monis.app/mlog"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/Azure/azure-workload-identity/pkg/cmd/podidentity/k8s"
	"github.com/Azure/azure-workload-identity/pkg/cmd/serviceaccount/options"
	"github.com/Azure/azure-workload-identity/pkg/kuberneteshelper"
	"github.com/Azure/azure-workload-identity/pkg/webhook"
)

var (
	scheme = runtime.NewScheme()
)

const (
	imageRepository = "mcr.microsoft.com/oss/azure/workload-identity"
	imageTag        = "v1.5.1"

	proxyInitImageName     = "proxy-init"
	proxyImageName         = "proxy"
	proxyInitContainerName = "azwi-proxy-init"
	proxyContainerName     = "azwi-proxy"

	nextStepsLogMessage = `Next steps:
1. Install the Azure Workload Identity Webhook. Refer to https://azure.github.io/azure-workload-identity/docs/installation.html.
2. Create federated identity credential for all identities used in this namespace. Refer to https://azure.github.io/azure-workload-identity/docs/topics/federated-identity-credential.html.
3. Review the generated config files and apply them with 'kubectl apply -f <generated file>'.`
)

var (
	proxyInitImage = fmt.Sprintf("%s/%s:%s", imageRepository, proxyInitImageName, imageTag)
	proxyImage     = fmt.Sprintf("%s/%s:%s", imageRepository, proxyImageName, imageTag)
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
}

type detectCmd struct {
	namespace                     string
	outputDir                     string
	proxyPort                     int32
	serviceAccountTokenExpiration time.Duration
	tenantID                      string
	kubeClient                    client.Client
	serializer                    *json.Serializer
}

func newDetectCmd() *cobra.Command {
	detectCmd := &detectCmd{}

	cmd := &cobra.Command{
		Use:   "detect",
		Short: "Detect the existing aad-pod-identity configuration",
		Long:  "This command will detect the existing aad-pod-identity configuration and generate a sample configuration file for migration to workload identity",
		PreRunE: func(cmd *cobra.Command, args []string) error {
			return detectCmd.prerun()
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return detectCmd.run()
		},
	}

	f := cmd.Flags()
	f.StringVar(&detectCmd.namespace, "namespace", "default", "Namespace to detect the configuration")
	f.StringVarP(&detectCmd.outputDir, "output-dir", "o", "", "Output directory to write the configuration files")
	f.Int32VarP(&detectCmd.proxyPort, "proxy-port", "p", 8000, "Proxy port to use for the proxy container")
	f.DurationVar(&detectCmd.serviceAccountTokenExpiration, options.ServiceAccountTokenExpiration.Flag, time.Duration(webhook.DefaultServiceAccountTokenExpiration)*time.Second, options.ServiceAccountTokenExpiration.Description)
	f.StringVar(&detectCmd.tenantID, "tenant-id", "", "Managed identity tenant id. If specified, the tenant id will be set as an annotation on the service account.")

	_ = cmd.MarkFlagRequired("output-dir")

	return cmd
}

func (dc *detectCmd) prerun() error {
	dc.serializer = json.NewSerializerWithOptions(
		json.DefaultMetaFactory, scheme, scheme,
		json.SerializerOptions{
			Yaml:   true,
			Pretty: true,
			Strict: true,
		},
	)
	// TODO(aramase): this validation can be refactored to a common function as it's used in multiple places
	minTokenExpirationDuration := time.Duration(webhook.MinServiceAccountTokenExpiration) * time.Second
	maxTokenExpirationDuration := time.Duration(webhook.MaxServiceAccountTokenExpiration) * time.Second
	if dc.serviceAccountTokenExpiration < minTokenExpirationDuration {
		return errors.Errorf("--service-account-token-expiration must be greater than or equal to %s", minTokenExpirationDuration.String())
	}
	if dc.serviceAccountTokenExpiration > maxTokenExpirationDuration {
		return errors.Errorf("--service-account-token-expiration must be less than or equal to %s", maxTokenExpirationDuration.String())
	}

	var err error
	dc.kubeClient, err = kuberneteshelper.GetKubeClient()
	if err != nil {
		return errors.Wrap(err, "failed to get Kubernetes client")
	}

	// create output directory if it doesn't exist
	if _, err := os.Stat(dc.outputDir); os.IsNotExist(err) {
		return os.MkdirAll(dc.outputDir, 0755)
	}

	return nil
}

func (dc *detectCmd) run() error {
	mlog.Debug("detecting aad-pod-identity configuration", "namespace", dc.namespace)

	// Implementing force namespaced mode
	// 1. Get AzureIdentityBinding in the namespace
	// 2. Get AzureIdentity referenced by AzureIdentityBinding and store in map with aadpodidbinding label value as key and AzureIdentity as value
	// 3. Get all pods in the namespace that have aadpodidbinding label
	// 4. For each pod, check if there is an owner reference (deployment, statefulset, cronjob, job, daemonset, replicaset, replicationcontroller)
	// 5. If there is an owner reference, get the owner reference object and add to map with aadpodidbinding label value as key and owner reference as value
	// 6. If no owner reference, then assume it's a static pod and add to map with aadpodidbinding label value as key and pod as value
	// 7. Loop through the first map and generate new config file for each owner reference and service account
	//    1. If owner is using a service account, get the service account and generate a config file with it
	//    2. If owner doesn't use service account, generate a new service account yaml file with owner name as service account name

	azureIdentityBindings, err := kuberneteshelper.ListAzureIdentityBinding(context.TODO(), dc.kubeClient, dc.namespace)
	if err != nil {
		return err
	}
	azureIdentities, err := kuberneteshelper.ListAzureIdentity(context.TODO(), dc.kubeClient, dc.namespace)
	if err != nil {
		return err
	}
	azureIdentityMap := make(map[string]aadpodv1.AzureIdentity)
	for _, azureIdentity := range azureIdentities {
		if azureIdentity.Spec.Type == aadpodv1.UserAssignedMSI {
			azureIdentityMap[azureIdentity.Name] = azureIdentity
		}
	}

	labelsToAzureIdentityMap := filterAzureIdentities(azureIdentityBindings, azureIdentityMap)
	if count := len(labelsToAzureIdentityMap); count > 0 {
		mlog.Debug("found valid aad-pod-identity bindings", "count", count)
	} else {
		mlog.Debug("did not find any valid aad-pod-identity bindings")
	}

	ownerReferences := make(map[metav1.OwnerReference]string)
	results := make(map[client.Object]string)

	for selector, azureIdentity := range labelsToAzureIdentityMap {
		mlog.Debug("getting pods", "selector", selector)
		pods, err := kuberneteshelper.ListPods(context.TODO(), dc.kubeClient, dc.namespace, map[string]string{aadpodv1.CRDLabelKey: selector})
		if err != nil {
			return err
		}
		for i := range pods {
			// for pods created by higher level constructors like deployment, statefulset, cronjob, job, daemonset, replicaset, replicationcontroller
			// we can get the owner reference with pod.OwnerReferences
			ownerFound := false
			if len(pods[i].OwnerReferences) > 0 {
				for _, ownerReference := range pods[i].OwnerReferences {
					// only get the owner reference that was set by the parent controller
					if ownerReference.Controller != nil && *ownerReference.Controller {
						ownerReferences[ownerReference] = azureIdentity.Spec.ClientID
						ownerFound = true
						break
					}
				}
			}
			// this is a standalone pod, so add it to the results
			if !ownerFound {
				p := pods[i]
				results[&p] = azureIdentity.Spec.ClientID
			}
		}
	}

	for ownerReference, clientID := range ownerReferences {
		owner, err := dc.getOwner(ownerReference)
		if err != nil {
			return err
		}
		results[owner] = clientID
	}

	// results contains all the resources that we need to generate a config file.
	// for each entry in the results map, we will generate a service account yaml file
	// and a resource file
	for o, clientID := range results {
		localObject := k8s.NewLocalObject(o)

		sa, err := dc.createServiceAccountFile(localObject.GetServiceAccountName(), localObject.GetName(), clientID)
		if err != nil {
			return err
		}
		if err = dc.createResourceFile(localObject, sa); err != nil {
			return err
		}
		mlog.Debug("generated config",
			"kind", strings.ToLower(localObject.GetObjectKind().GroupVersionKind().Kind),
			"name", localObject.GetName(),
			"clientID", clientID,
		)
	}

	if len(results) == 0 {
		mlog.Debug("no aad-pod-identity configuration found", "namespace", dc.namespace)
		return nil
	}

	mlog.Info("generated resource and service account files", "directory", dc.outputDir)
	mlog.Info(nextStepsLogMessage)
	return nil
}

// createServiceAccountFile will create a service account yaml file
//  1. If the resource is using default service account, then a new service account yaml is generated
//     with the resource name as service account name
//  2. If the resource is already using a non-default service account, then we modify that service account
//     to generate the desired yaml file
//
// The service account yaml will contain the workload identity use label ("azure.workload.identity/use: true")
// and the client-id annotation ("azure.workload.identity/client-id: <client-id from AzureIdentity>")
func (dc *detectCmd) createServiceAccountFile(name, ownerName, clientID string) (*corev1.ServiceAccount, error) {
	sa := &corev1.ServiceAccount{}
	var err error
	if name == "" || name == "default" {
		mlog.Debug("generating a new service account instead of using default service account", "owner", ownerName)
		// generate a new service account yaml file with owner name as service account name
		sa.SetName(ownerName)
		sa.SetNamespace(dc.namespace)
	} else {
		// get service account referenced by the owner
		if sa, err = kuberneteshelper.GetServiceAccount(context.TODO(), dc.kubeClient, dc.namespace, name); err != nil {
			return nil, err
		}
	}

	// set the annotations for the service account
	saAnnotations := make(map[string]string)
	if sa.GetAnnotations() != nil {
		saAnnotations = sa.GetAnnotations()
	}
	saAnnotations[webhook.ClientIDAnnotation] = clientID
	// Round to the nearest second before converting to a string
	saAnnotations[webhook.ServiceAccountTokenExpiryAnnotation] = fmt.Sprintf("%.0f", dc.serviceAccountTokenExpiration.Round(time.Second).Seconds())
	if dc.tenantID != "" {
		saAnnotations[webhook.TenantIDAnnotation] = dc.tenantID
	}
	sa.SetAnnotations(saAnnotations)
	sa.SetGroupVersionKind(schema.GroupVersionKind{Group: "", Version: "v1", Kind: "ServiceAccount"})
	sa.SetResourceVersion("")

	fileName := filepath.Join(dc.getServiceAccountFileName(ownerName))
	// write the service account yaml file
	file, err := os.Create(fileName)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return sa, dc.serializer.Encode(sa, file)
}

// createResourceFile will create a resource yaml file
//
//	If the resource is using default service account, then the service account name is updated to the resource name
//	to match the service account yaml we generated in createServiceAccountFile()
//
// The resource yaml will contain:
// 1. proxy container that is required for migration
// 2. proxy-init init container that sets up iptables rules to redirect IMDS traffic to proxy
func (dc *detectCmd) createResourceFile(localObject k8s.LocalObject, sa *corev1.ServiceAccount) error {
	// add the init container to the container list
	localObject.SetInitContainers(dc.addProxyInitContainer(localObject.GetInitContainers()))
	// add the proxy container to the container list
	localObject.SetContainers(dc.addProxyContainer(localObject.GetContainers()))
	// set the service account name for the object
	localObject.SetServiceAccountName(sa.GetName())
	// reset the managed fields to reduce clutter in the output yaml
	localObject.SetManagedFields(nil)
	// reset the resource version, uid and other metadata to make the yaml file applyable
	localObject.SetResourceVersion("")
	localObject.SetUID("")
	localObject.SetCreationTimestamp(metav1.Time{})
	localObject.SetSelfLink("")
	localObject.SetGeneration(0)
	localObject.ResetStatus()
	// set the group version kind explicitly before serializing the object
	localObject.SetGVK()

	// write the modified object to the output dir
	file, err := os.Create(dc.getResourceFileName(localObject))
	if err != nil {
		return err
	}
	defer file.Close()

	return dc.serializer.Encode(localObject.GetObject(), file)
}

// addProxyInitContainer adds the proxy-init container to the list of init containers
func (dc *detectCmd) addProxyInitContainer(initContainers []corev1.Container) []corev1.Container {
	if initContainers == nil {
		initContainers = make([]corev1.Container, 0)
	}

	for _, container := range initContainers {
		if strings.HasPrefix(container.Image, fmt.Sprintf("%s/%s", imageRepository, proxyInitImageName)) {
			return initContainers
		}
	}

	trueVal := true
	// proxy-init needs to be run as root
	runAsRoot := int64(0)
	// add the init container to the container list
	proxyInitContainer := corev1.Container{
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
				Value: strconv.FormatInt(int64(dc.proxyPort), 10),
			},
		},
	}

	initContainers = append(initContainers, proxyInitContainer)
	return initContainers
}

// addProxyContainer adds the proxy container to the list of containers
func (dc *detectCmd) addProxyContainer(containers []corev1.Container) []corev1.Container {
	if containers == nil {
		containers = make([]corev1.Container, 0)
	}

	for _, container := range containers {
		if strings.HasPrefix(container.Image, fmt.Sprintf("%s/%s", imageRepository, proxyImageName)) {
			return containers
		}
	}

	logLevel := mlog.LevelInfo // somewhat arbitrary decision
	proxyContainer := corev1.Container{
		Name:            proxyContainerName,
		Image:           proxyImage,
		ImagePullPolicy: corev1.PullIfNotPresent,
		Args: []string{
			fmt.Sprintf("--proxy-port=%d", dc.proxyPort),
			fmt.Sprintf("--log-level=%s", logLevel),
		},
		Ports: []corev1.ContainerPort{
			{
				ContainerPort: dc.proxyPort,
			},
		},
		Lifecycle: &corev1.Lifecycle{
			PostStart: &corev1.LifecycleHandler{
				Exec: &corev1.ExecAction{
					Command: []string{
						"/proxy",
						fmt.Sprintf("--proxy-port=%d", dc.proxyPort),
						"--probe",
						fmt.Sprintf("--log-level=%s", logLevel),
					},
				},
			},
		},
	}

	containers = append(containers, proxyContainer)
	return containers
}

// getOwner returns the owner of the resource
// It makes a recursive call to get the top level owner of the resource
func (dc *detectCmd) getOwner(ownerRef metav1.OwnerReference) (owner client.Object, err error) {
	mlog.Debug("getting owner reference", "name", ownerRef.Name)
	or, err := dc.getOwnerObject(ownerRef)
	if err != nil {
		return nil, err
	}
	owners := or.GetOwnerReferences()
	for _, o := range owners {
		if o.Controller != nil && *o.Controller {
			return dc.getOwner(o)
		}
	}
	return or, nil
}

// getOwnerObject gets the owner object based on the owner reference kind
func (dc *detectCmd) getOwnerObject(ownerRef metav1.OwnerReference) (client.Object, error) {
	switch ownerRef.Kind {
	case "Deployment":
		return kuberneteshelper.GetObject(context.TODO(), dc.kubeClient, dc.namespace, ownerRef.Name, &appsv1.Deployment{})
	case "StatefulSet":
		return kuberneteshelper.GetObject(context.TODO(), dc.kubeClient, dc.namespace, ownerRef.Name, &appsv1.StatefulSet{})
	case "CronJob":
		return kuberneteshelper.GetObject(context.TODO(), dc.kubeClient, dc.namespace, ownerRef.Name, &batchv1.CronJob{})
	case "Job":
		return kuberneteshelper.GetObject(context.TODO(), dc.kubeClient, dc.namespace, ownerRef.Name, &batchv1.Job{})
	case "DaemonSet":
		return kuberneteshelper.GetObject(context.TODO(), dc.kubeClient, dc.namespace, ownerRef.Name, &appsv1.DaemonSet{})
	case "ReplicaSet":
		return kuberneteshelper.GetObject(context.TODO(), dc.kubeClient, dc.namespace, ownerRef.Name, &appsv1.ReplicaSet{})
	case "ReplicationController":
		return kuberneteshelper.GetObject(context.TODO(), dc.kubeClient, dc.namespace, ownerRef.Name, &corev1.ReplicationController{})
	default:
		return nil, fmt.Errorf("unsupported owner kind: %s", ownerRef.Kind)
	}
}

func (dc *detectCmd) getResourceFileName(obj k8s.LocalObject) string {
	return filepath.Join(dc.outputDir, obj.GetName()+".yaml")
}

func (dc *detectCmd) getServiceAccountFileName(prefix string) string {
	return filepath.Join(dc.outputDir, fmt.Sprintf("%s-serviceaccount.yaml", prefix))
}

// filterAzureIdentities will filter out the Azure identities referenced in AzureIdentityBinding
// the return value is a map of selector used in AzureIdentityBinding to the AzureIdentity
func filterAzureIdentities(bindings []aadpodv1.AzureIdentityBinding, identities map[string]aadpodv1.AzureIdentity) map[string]aadpodv1.AzureIdentity {
	labelsToAzureIdentityMap := make(map[string]aadpodv1.AzureIdentity)
	for _, binding := range bindings {
		if binding.Spec.Selector == "" || binding.Spec.AzureIdentity == "" {
			continue
		}
		// this can happen when multiple AzureIdentityBinding exist in the namespace with same selector
		// Multiple AzureIdentityBinding with same selector are configured in AAD Pod Identity to enable a
		// a single pod to have access to multiple identities.
		// In case of workload identity, we can only annotate with a single client id and there can only
		// be one AZURE_CLIENT_ID environment variable. The client id annotation will be configured to the first
		// AzureIdentityBinding with the selector. The workload will use the client id of the specific identity
		// to get a token and will not really use the AZURE_CLIENT_ID environment variable.
		if b, ok := labelsToAzureIdentityMap[binding.Spec.Selector]; ok {
			mlog.Debug("multiple AzureIdentityBinding found, using the first one",
				"selector", binding.Spec.Selector,
				"binding", b.Name,
			)
			continue
		}
		if azureIdentity, ok := identities[binding.Spec.AzureIdentity]; ok {
			labelsToAzureIdentityMap[binding.Spec.Selector] = azureIdentity
		}
	}

	return labelsToAzureIdentityMap
}
