package podidentity

import (
	"context"
	"fmt"

	"github.com/Azure/azure-workload-identity/pkg/kuberneteshelper"

	aadpodv1 "github.com/Azure/aad-pod-identity/pkg/apis/aadpodidentity/v1"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

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

	ownerReferences := make(map[metav1.OwnerReference]bool)
	for selector := range labelsToAzureIdentityMap {
		log.Debugf("getting pods with selector: %s", selector)
		pods, err := kuberneteshelper.ListPods(context.TODO(), dc.kubeClient, dc.namespace, map[string]string{"aadpodidbinding": selector})
		if err != nil {
			return err
		}
		for _, pod := range pods {
			// for pods created by higher level constructors like deployment, statefulset, cronjob, job, daemonset, replicaset, replicationcontroller
			// we can get the owner reference with pod.OwnerReferences
			if len(pod.OwnerReferences) > 0 && pod.OwnerReferences[0].Controller != nil && *pod.OwnerReferences[0].Controller {
				ownerReferences[pod.OwnerReferences[0]] = true
			}
		}
	}

	for ownerReference := range ownerReferences {
		log.Debugf("getting owner reference: %s", ownerReference.Name)
		owner, err := dc.getOwner(ownerReference)
		if err != nil {
			return err
		}
		serviceAccountName := getServiceAccountName(owner)
		sa := &corev1.ServiceAccount{}
		if serviceAccountName == "" || serviceAccountName == "default" {
			// generate a new service account yaml file with owner name as service account name
			sa.SetName(owner.GetName())
			sa.SetNamespace(dc.namespace)
		} else {
			// get service account and generate config file with it
			sa, err = kuberneteshelper.GetServiceAccount(context.TODO(), dc.kubeClient, dc.namespace, serviceAccountName)
			if err != nil {
				return err
			}
		}

		// generate config file for owner

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

func getContainers(obj client.Object) *[]corev1.Container {
	switch obj.(type) {
	case *corev1.Pod:
		return &obj.(*corev1.Pod).Spec.Containers
	case *appsv1.Deployment:
		return &obj.(*appsv1.Deployment).Spec.Template.Spec.Containers
	case *appsv1.StatefulSet:
		return &obj.(*appsv1.StatefulSet).Spec.Template.Spec.Containers
	case *appsv1.DaemonSet:
		return &obj.(*appsv1.DaemonSet).Spec.Template.Spec.Containers
	case *appsv1.ReplicaSet:
		return &obj.(*appsv1.ReplicaSet).Spec.Template.Spec.Containers
	case *corev1.ReplicationController:
		return &obj.(*corev1.ReplicationController).Spec.Template.Spec.Containers
	case *batchv1.CronJob:
		return &obj.(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec.Containers
	case *batchv1.Job:
		return &obj.(*batchv1.Job).Spec.Template.Spec.Containers
	default:
		return nil
	}
}
