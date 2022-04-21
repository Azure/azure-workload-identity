package podidentity

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Azure/azure-workload-identity/pkg/cmd/podidentity/k8s"
	"github.com/Azure/azure-workload-identity/pkg/kuberneteshelper"
	"github.com/Azure/azure-workload-identity/pkg/webhook"

	aadpodv1 "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	scheme = runtime.NewScheme()
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
}

type detectCmd struct {
	namespace  string
	outputDir  string
	kubeClient client.Client
	serializer *json.Serializer
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
	f.StringVar(&detectCmd.outputDir, "output-dir", "", "Output directory to write the configuration files")

	_ = cmd.MarkFlagRequired("output-dir")

	return cmd
}

func (dc *detectCmd) prerun() error {
	var err error
	dc.kubeClient, err = kuberneteshelper.GetKubeClient()
	dc.serializer = json.NewSerializerWithOptions(
		json.DefaultMetaFactory, scheme, scheme,
		json.SerializerOptions{
			Yaml:   true,
			Pretty: true,
			Strict: true,
		},
	)

	return err
}

func (dc *detectCmd) run() error {
	log.Debugf("detecting aad-pod-identity configuration in namespace: %s", dc.namespace)

	// Implementing force namespaced mode
	// 1. Get AzureIdentityBinding in the namespace
	// 2. Get AzureIdentity referenced by AzureIdentityBinding and store in map with aadpodidbinding label value as key and AzureIdentity as value
	// 3. Get all pods in the namespace that have aadpodidbinding label
	// 4. For each pod, check if there is an owner reference (deployment, statefulset, cronjob, job, daemonset, replicaset, replicationcontroller)
	// 5. If there is an owner reference, get the owner reference object and add to map with aadpodidbinding label value as key and owner reference as value
	// 6. If no owner reference, then assume it's a static pod and add to map with aadpodidbinding label value as key and pod as value
	// 7. Loop through the first map and generate new config file for each owner reference and service account
	//    1. If owner using service account, get service account and generate config file with it
	//    2. If owner doesn't use service account, generate a new service account yaml file with owner name as service account name

	azureIdentityBindings, err := kuberneteshelper.ListAzureIdentityBinding(context.TODO(), dc.kubeClient, dc.namespace)
	if err != nil {
		return err
	}
	azureIdentities, err := kuberneteshelper.ListAzureIdentity(context.TODO(), dc.kubeClient, dc.namespace)
	if err != nil {
		return err
	}

	labelsToAzureIdentityMap := make(map[string]aadpodv1.AzureIdentity)
	for _, azureIdentityBinding := range azureIdentityBindings {
		if azureIdentityBinding.Spec.Selector == "" || azureIdentityBinding.Spec.AzureIdentity == "" {
			continue
		}
		if azureIdentity, ok := azureIdentities[azureIdentityBinding.Spec.AzureIdentity]; ok {
			labelsToAzureIdentityMap[azureIdentityBinding.Spec.Selector] = azureIdentity
		}
	}

	ownerReferences := make(map[metav1.OwnerReference]string)
	results := make(map[client.Object]string)

	for selector, azureIdentity := range labelsToAzureIdentityMap {
		log.Debugf("getting pods with selector: %s", selector)
		pods, err := kuberneteshelper.ListPods(context.TODO(), dc.kubeClient, dc.namespace, map[string]string{"aadpodidbinding": selector})
		if err != nil {
			return err
		}
		for _, pod := range pods {
			// for pods created by higher level constructors like deployment, statefulset, cronjob, job, daemonset, replicaset, replicationcontroller
			// we can get the owner reference with pod.OwnerReferences
			ownerFound := false
			if len(pod.OwnerReferences) > 0 {
				for _, ownerReference := range pod.OwnerReferences {
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
				results[&pod] = azureIdentity.Spec.ClientID
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

	// results contains all the resources that we need to generate a config file
	// for each entry in the results map, we will generate a service account yaml file
	// and a resource file
	for o, clientID := range results {
		localObject := k8s.NewLocalObject(o)
		log.Debugf("generating config for %s, clientID: %s", localObject.GetName(), clientID)

		// make directory in output dir with object name
		dir := filepath.Join(dc.outputDir, localObject.GetName())
		if err = os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		sa, err := dc.createServiceAccountFile(dir, localObject.GetServiceAccountName(), localObject.GetName(), clientID)
		if err != nil {
			return err
		}
		if err = dc.createResourceFile(dir, localObject, sa); err != nil {
			return err
		}
		log.Debugf("generated config for %s, clientID: %s", localObject.GetName(), clientID)
	}

	return nil
}

// createServiceAccountFile will create a service account yaml file
// 1. If the resource is using default service account, then a new service account yaml is generated
//    with the resource name as service account name
// 2. If the resource is already using a non-default service account, then we modify that service account
//    to generate the desired yaml file
// The service account yaml will contain the workload identity use label ("azure.workload.identity/use: true")
// and the client-id annotation ("azure.workload.identity/client-id: <client-id from AzureIdentity>")
func (dc *detectCmd) createServiceAccountFile(dir, name, ownerName, clientID string) (*corev1.ServiceAccount, error) {
	sa := &corev1.ServiceAccount{}
	var err error
	if name == "" || name == "default" {
		log.Debugf("%s is using default service account, generating a new service account", ownerName)
		// generate a new service account yaml file with owner name as service account name
		sa.SetName(ownerName)
		sa.SetNamespace(dc.namespace)
	} else {
		// get service account referenced by the owner
		if sa, err = kuberneteshelper.GetServiceAccount(context.TODO(), dc.kubeClient, dc.namespace, name); err != nil {
			return nil, err
		}
	}

	saLabels := make(map[string]string)
	if sa.GetLabels() != nil {
		saLabels = sa.GetLabels()
	}
	saLabels[webhook.UseWorkloadIdentityLabel] = "true"
	sa.SetLabels(saLabels)

	// set the annotations for the service account
	saAnnotations := make(map[string]string)
	if sa.GetAnnotations() != nil {
		saAnnotations = sa.GetAnnotations()
	}
	saAnnotations[webhook.ClientIDAnnotation] = clientID
	sa.SetAnnotations(saAnnotations)
	sa.SetGroupVersionKind(schema.GroupVersionKind{Group: "", Version: "v1", Kind: "ServiceAccount"})

	// write the service account yaml file
	file, err := os.Create(filepath.Join(dir, sa.GetName()+"-serviceaccount.yaml"))
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return sa, dc.serializer.Encode(sa, file)
}

// createResourceFile will create a resource yaml file
//   If the resource is using default service account, then the service account name is updated to the resource name
//   to match the service account yaml we generated in createServiceAccountFile()
// The resource yaml will contain:
// 1. proxy container that is required for migration
// 2. proxy-init init container that sets up iptables rules to redirect IMDS traffic to proxy
func (dc *detectCmd) createResourceFile(dir string, localObject k8s.LocalObject, sa *corev1.ServiceAccount) error {
	// add the init container to the container list
	localObject.SetInitContainers(addProxyInitContainer(localObject.GetInitContainers()))
	// add the proxy container to the container list
	localObject.SetContainers(addProxyContainer(localObject.GetContainers()))
	// set the service account name for the object
	localObject.SetServiceAccountName(sa.GetName())
	// reset the managed fields to reduce clutter in the output yaml
	localObject.SetManagedFields(nil)
	// reset the resource version, uid and other metdata to make the yaml file applyable
	localObject.SetResourceVersion("")
	localObject.SetUID("")
	localObject.SetCreationTimestamp(metav1.Time{})
	localObject.SetSelfLink("")
	localObject.SetGeneration(0)
	localObject.ResetStatus()
	// set the group version kind explicitly before serializing the object
	localObject.SetGVK()

	// write the modified object to the output dir
	file, err := os.Create(filepath.Join(dir, localObject.GetName()+".yaml"))
	if err != nil {
		return err
	}
	defer file.Close()

	return dc.serializer.Encode(localObject, file)
}

// addProxyInitContainer adds the proxy-init container to the list of init containers
func addProxyInitContainer(initContainers []corev1.Container) []corev1.Container {
	if initContainers == nil {
		initContainers = make([]corev1.Container, 0)
	}

	trueVal := true
	// proxy-init needs to be run as root
	runAsRoot := int64(0)
	// add the init container to the container list
	proxyInitContainer := corev1.Container{
		Name:            "azwi-proxy-init",
		Image:           "mcr.microsoft.com/oss/azure/workload-identity/proxy-init:v0.9.0",
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

	initContainers = append(initContainers, proxyInitContainer)
	return initContainers
}

// addProxyContainer adds the proxy container to the list of containers
func addProxyContainer(containers []corev1.Container) []corev1.Container {
	if containers == nil {
		containers = make([]corev1.Container, 0)
	}

	proxyContainer := corev1.Container{
		Name:            "azwi-proxy",
		Image:           "mcr.microsoft.com/oss/azure/workload-identity/proxy:v0.9.0",
		ImagePullPolicy: corev1.PullIfNotPresent,
		Args:            []string{"--log-encoder=json"},
		Ports: []corev1.ContainerPort{
			{
				Name:          "http",
				ContainerPort: 8000,
			},
		},
	}

	containers = append(containers, proxyContainer)
	return containers
}

// getOwner returns the owner of the resource
// It makes a recursive call to get the top level owner of the resource
func (dc *detectCmd) getOwner(ownerRef metav1.OwnerReference) (owner client.Object, err error) {
	log.Debugf("getting owner reference: %s", ownerRef.Name)
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
