package k8s

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type replicaSetLocalObject struct {
	client.Object
}

func newReplicaSetLocalObject(obj client.Object) LocalObject {
	return &replicaSetLocalObject{
		Object: obj,
	}
}

func (o *replicaSetLocalObject) GetServiceAccountName() string {
	return o.Object.(*appsv1.ReplicaSet).Spec.Template.Spec.ServiceAccountName
}

func (o *replicaSetLocalObject) SetServiceAccountName(name string) {
	o.Object.(*appsv1.ReplicaSet).Spec.Template.Spec.ServiceAccountName = name
}

func (o *replicaSetLocalObject) GetContainers() []corev1.Container {
	return o.Object.(*appsv1.ReplicaSet).Spec.Template.Spec.Containers
}

func (o *replicaSetLocalObject) SetContainers(containers []corev1.Container) {
	o.Object.(*appsv1.ReplicaSet).Spec.Template.Spec.Containers = containers
}

func (o *replicaSetLocalObject) GetInitContainers() []corev1.Container {
	return o.Object.(*appsv1.ReplicaSet).Spec.Template.Spec.InitContainers
}

func (o *replicaSetLocalObject) SetInitContainers(containers []corev1.Container) {
	o.Object.(*appsv1.ReplicaSet).Spec.Template.Spec.InitContainers = containers
}

func (o *replicaSetLocalObject) SetGVK() {
	o.Object.(*appsv1.ReplicaSet).SetGroupVersionKind(schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "ReplicaSet"})
}

func (o *replicaSetLocalObject) ResetStatus() {
	o.Object.(*appsv1.ReplicaSet).Status = appsv1.ReplicaSetStatus{}
}

func (o *replicaSetLocalObject) GetObject() client.Object {
	return o.Object
}
