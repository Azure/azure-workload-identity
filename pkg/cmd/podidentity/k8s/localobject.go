package k8s

import (
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type LocalObject interface {
	client.Object
	GetServiceAccountName() string
	SetServiceAccountName(name string)
	GetContainers() []corev1.Container
	SetContainers(containers []corev1.Container)
	GetInitContainers() []corev1.Container
	SetInitContainers(containers []corev1.Container)
	SetGVK()
	GetObject() client.Object
	ResetStatus()
}

func NewLocalObject(obj client.Object) LocalObject {
	switch obj.(type) {
	case *corev1.Pod:
		return newPodLocalObject(obj)
	case *appsv1.Deployment:
		return newDeploymentLocalObject(obj)
	case *appsv1.StatefulSet:
		return newStatefulSetLocalObject(obj)
	case *appsv1.DaemonSet:
		return newDaemonSetLocalObject(obj)
	case *appsv1.ReplicaSet:
		return newReplicaSetLocalObject(obj)
	case *corev1.ReplicationController:
		return newReplicationControllerLocalObject(obj)
	case *batchv1.CronJob:
		return newCronJobLocalObject(obj)
	case *batchv1.Job:
		return newJobLocalObject(obj)
	default:
		return nil
	}
}
