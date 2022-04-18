package podidentity

import (
	"context"
	"fmt"
	"os"

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
	return err
}

func (dc *detectCmd) run() error {
	log.Debugf("detecting aad-pod-identity configuration in namespace: %s", dc.namespace)

	// Implementing force namespaced mode for now
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
			if len(pod.OwnerReferences) > 0 && pod.OwnerReferences[0].Controller != nil && *pod.OwnerReferences[0].Controller {
				ownerReferences[pod.OwnerReferences[0]] = azureIdentity.Spec.ClientID
			} else {
				// this is a standalone pod, so add it to the results
				results[&pod] = azureIdentity.Spec.ClientID
			}
		}
	}

	for ownerReference, clientID := range ownerReferences {
		owner, err := dc.getActualOwner(ownerReference)
		if err != nil {
			return err
		}
		results[owner] = clientID
	}

	// results contains all the pods and parents that we need to generate a config for
	// now get the service account for each entry in result and generate the config
	for obj, clientID := range results {
		serviceAccountName := getServiceAccountName(obj)
		sa := &corev1.ServiceAccount{}
		if serviceAccountName == "" || serviceAccountName == "default" {
			log.Debugf("%s is using default service account, generating a new service account", obj.GetName())
			// generate a new service account yaml file with owner name as service account name
			sa.SetName(obj.GetName())
			sa.SetNamespace(dc.namespace)
		} else {
			// get service account and generate config file with it
			sa, err = kuberneteshelper.GetServiceAccount(context.TODO(), dc.kubeClient, dc.namespace, serviceAccountName)
			if err != nil {
				return err
			}
		}
		// make directory in output dir with object name
		err = os.MkdirAll(dc.outputDir+"/"+obj.GetName(), 0755)
		if err != nil {
			return err
		}
		// write modified service account yaml file
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

		e := json.NewSerializerWithOptions(
			json.DefaultMetaFactory, scheme, scheme,
			json.SerializerOptions{
				Yaml:   true,
				Pretty: true,
				Strict: true,
			},
		)

		sa.SetGroupVersionKind(schema.GroupVersionKind{Group: "", Version: "v1", Kind: "ServiceAccount"})
		file, err := os.Create(dc.outputDir + "/" + obj.GetName() + "/" + sa.GetName() + "-serviceaccount.yaml")
		if err != nil {
			return err
		}
		defer file.Close()

		err = e.Encode(sa, file)
		if err != nil {
			return err
		}

		// mutate object with new containers and write that also to the output dir
		initContainers, containers := getContainers(obj)
		// add the init container to the container list
		initContainers = addProxyInitContainer(initContainers)
		// add the proxy container to the container list
		containers = addProxyContainer(containers)

		// set the new containers
		mutateObject(obj, initContainers, containers, sa.Name)
		// set the service account name

		obj.SetManagedFields(nil)
		obj.SetResourceVersion("")
		obj.SetUID("")
		obj.SetSelfLink("")
		obj.SetGeneration(0)
		obj.SetCreationTimestamp(metav1.Time{})
		obj.SetDeletionTimestamp(nil)
		annotations := obj.GetAnnotations()
		if annotations != nil {
			delete(annotations, "kubectl.kubernetes.io/last-applied-configuration")
			obj.SetAnnotations(annotations)
		}

		setGVK(obj)
		// write the modified object to the output dir
		file, err = os.Create(dc.outputDir + "/" + obj.GetName() + "/" + obj.GetName() + ".yaml")
		if err != nil {
			return err
		}
		defer file.Close()

		err = e.Encode(obj, file)
		if err != nil {
			return err
		}
		log.Debugf("generated config for %s, clientID: %s", obj.GetName(), clientID)
	}

	return nil
}

func (dc *detectCmd) getOwner(ownerRef metav1.OwnerReference) (client.Object, error) {
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

func getServiceAccountName(obj client.Object) string {
	switch obj.(type) {
	case *corev1.Pod:
		return obj.(*corev1.Pod).Spec.ServiceAccountName
	case *appsv1.Deployment:
		return obj.(*appsv1.Deployment).Spec.Template.Spec.ServiceAccountName
	case *appsv1.StatefulSet:
		return obj.(*appsv1.StatefulSet).Spec.Template.Spec.ServiceAccountName
	case *appsv1.DaemonSet:
		return obj.(*appsv1.DaemonSet).Spec.Template.Spec.ServiceAccountName
	case *appsv1.ReplicaSet:
		return obj.(*appsv1.ReplicaSet).Spec.Template.Spec.ServiceAccountName
	case *corev1.ReplicationController:
		return obj.(*corev1.ReplicationController).Spec.Template.Spec.ServiceAccountName
	case *batchv1.CronJob:
		return obj.(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec.ServiceAccountName
	case *batchv1.Job:
		return obj.(*batchv1.Job).Spec.Template.Spec.ServiceAccountName
	default:
		return ""
	}
}

func getContainers(obj client.Object) ([]corev1.Container, []corev1.Container) {
	switch obj.(type) {
	case *corev1.Pod:
		return obj.(*corev1.Pod).Spec.InitContainers, obj.(*corev1.Pod).Spec.Containers
	case *appsv1.Deployment:
		return obj.(*appsv1.Deployment).Spec.Template.Spec.InitContainers, obj.(*appsv1.Deployment).Spec.Template.Spec.Containers
	case *appsv1.StatefulSet:
		return obj.(*appsv1.StatefulSet).Spec.Template.Spec.InitContainers, obj.(*appsv1.StatefulSet).Spec.Template.Spec.Containers
	case *appsv1.DaemonSet:
		return obj.(*appsv1.DaemonSet).Spec.Template.Spec.InitContainers, obj.(*appsv1.DaemonSet).Spec.Template.Spec.Containers
	case *appsv1.ReplicaSet:
		return obj.(*appsv1.ReplicaSet).Spec.Template.Spec.InitContainers, obj.(*appsv1.ReplicaSet).Spec.Template.Spec.Containers
	case *corev1.ReplicationController:
		return obj.(*corev1.ReplicationController).Spec.Template.Spec.InitContainers, obj.(*corev1.ReplicationController).Spec.Template.Spec.Containers
	case *batchv1.CronJob:
		return obj.(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec.InitContainers, obj.(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec.Containers
	case *batchv1.Job:
		return obj.(*batchv1.Job).Spec.Template.Spec.InitContainers, obj.(*batchv1.Job).Spec.Template.Spec.Containers
	default:
		return nil, nil
	}
}

func setGVK(obj client.Object) {
	switch obj.(type) {
	case *corev1.Pod:
		obj.(*corev1.Pod).SetGroupVersionKind(schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"})
	case *appsv1.Deployment:
		obj.(*appsv1.Deployment).SetGroupVersionKind(schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"})
	case *appsv1.StatefulSet:
		obj.(*appsv1.StatefulSet).SetGroupVersionKind(schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "StatefulSet"})
	case *appsv1.DaemonSet:
		obj.(*appsv1.DaemonSet).SetGroupVersionKind(schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "DaemonSet"})
	case *appsv1.ReplicaSet:
		obj.(*appsv1.ReplicaSet).SetGroupVersionKind(schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "ReplicaSet"})
	case *corev1.ReplicationController:
		obj.(*corev1.ReplicationController).SetGroupVersionKind(schema.GroupVersionKind{Group: "", Version: "v1", Kind: "ReplicationController"})
	case *batchv1.CronJob:
		obj.(*batchv1.CronJob).SetGroupVersionKind(schema.GroupVersionKind{Group: "batch", Version: "v1", Kind: "CronJob"})
	case *batchv1.Job:
		obj.(*batchv1.Job).SetGroupVersionKind(schema.GroupVersionKind{Group: "batch", Version: "v1", Kind: "Job"})
	}
}

func mutateObject(obj client.Object, initContainers, containers []corev1.Container, serviceAccountName string) {
	switch obj.(type) {
	case *corev1.Pod:
		obj.(*corev1.Pod).Spec.InitContainers = initContainers
		obj.(*corev1.Pod).Spec.Containers = containers
		obj.(*corev1.Pod).Status = corev1.PodStatus{}
		obj.(*corev1.Pod).Spec.ServiceAccountName = serviceAccountName
	case *appsv1.Deployment:
		obj.(*appsv1.Deployment).Spec.Template.Spec.InitContainers = initContainers
		obj.(*appsv1.Deployment).Spec.Template.Spec.Containers = containers
		obj.(*appsv1.Deployment).Status = appsv1.DeploymentStatus{}
		obj.(*appsv1.Deployment).Spec.Template.Spec.ServiceAccountName = serviceAccountName
	case *appsv1.StatefulSet:
		obj.(*appsv1.StatefulSet).Spec.Template.Spec.InitContainers = initContainers
		obj.(*appsv1.StatefulSet).Spec.Template.Spec.Containers = containers
		obj.(*appsv1.StatefulSet).Status = appsv1.StatefulSetStatus{}
		obj.(*appsv1.StatefulSet).Spec.Template.Spec.ServiceAccountName = serviceAccountName
	case *appsv1.DaemonSet:
		obj.(*appsv1.DaemonSet).Spec.Template.Spec.InitContainers = initContainers
		obj.(*appsv1.DaemonSet).Spec.Template.Spec.Containers = containers
		obj.(*appsv1.DaemonSet).Status = appsv1.DaemonSetStatus{}
		obj.(*appsv1.DaemonSet).Spec.Template.Spec.ServiceAccountName = serviceAccountName
	case *appsv1.ReplicaSet:
		obj.(*appsv1.ReplicaSet).Spec.Template.Spec.InitContainers = initContainers
		obj.(*appsv1.ReplicaSet).Spec.Template.Spec.Containers = containers
		obj.(*appsv1.ReplicaSet).Status = appsv1.ReplicaSetStatus{}
		obj.(*appsv1.ReplicaSet).Spec.Template.Spec.ServiceAccountName = serviceAccountName
	case *corev1.ReplicationController:
		obj.(*corev1.ReplicationController).Spec.Template.Spec.InitContainers = initContainers
		obj.(*corev1.ReplicationController).Spec.Template.Spec.Containers = containers
		obj.(*corev1.ReplicationController).Status = corev1.ReplicationControllerStatus{}
		obj.(*corev1.ReplicationController).Spec.Template.Spec.ServiceAccountName = serviceAccountName
	case *batchv1.CronJob:
		obj.(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec.InitContainers = initContainers
		obj.(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec.Containers = containers
		obj.(*batchv1.CronJob).Status = batchv1.CronJobStatus{}
		obj.(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec.ServiceAccountName = serviceAccountName
	case *batchv1.Job:
		obj.(*batchv1.Job).Spec.Template.Spec.InitContainers = initContainers
		obj.(*batchv1.Job).Spec.Template.Spec.Containers = containers
		obj.(*batchv1.Job).Status = batchv1.JobStatus{}
		obj.(*batchv1.Job).Spec.Template.Spec.ServiceAccountName = serviceAccountName
	}
}

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

func (dc *detectCmd) getActualOwner(ownerRef metav1.OwnerReference) (owner client.Object, err error) {
	log.Debugf("getting owner reference: %s", ownerRef.Name)
	or, err := dc.getOwner(ownerRef)
	if err != nil {
		return nil, err
	}
	owners := or.GetOwnerReferences()
	if len(owners) > 0 && owners[0].Controller != nil && *owners[0].Controller {
		return dc.getActualOwner(owners[0])
	}
	return or, nil
}
