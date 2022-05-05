package k8s

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type replicationControllerLocalObject struct {
	client.Object
}

func newReplicationControllerLocalObject(obj client.Object) LocalObject {
	return &replicationControllerLocalObject{
		Object: obj,
	}
}

func (o *replicationControllerLocalObject) GetServiceAccountName() string {
	return o.Object.(*corev1.ReplicationController).Spec.Template.Spec.ServiceAccountName
}

func (o *replicationControllerLocalObject) SetServiceAccountName(name string) {
	o.Object.(*corev1.ReplicationController).Spec.Template.Spec.ServiceAccountName = name
}

func (o *replicationControllerLocalObject) GetContainers() []corev1.Container {
	return o.Object.(*corev1.ReplicationController).Spec.Template.Spec.Containers
}

func (o *replicationControllerLocalObject) SetContainers(containers []corev1.Container) {
	o.Object.(*corev1.ReplicationController).Spec.Template.Spec.Containers = containers
}

func (o *replicationControllerLocalObject) GetInitContainers() []corev1.Container {
	return o.Object.(*corev1.ReplicationController).Spec.Template.Spec.InitContainers
}

func (o *replicationControllerLocalObject) SetInitContainers(containers []corev1.Container) {
	o.Object.(*corev1.ReplicationController).Spec.Template.Spec.InitContainers = containers
}

func (o *replicationControllerLocalObject) SetGVK() {
	o.Object.(*corev1.ReplicationController).SetGroupVersionKind(schema.GroupVersionKind{Group: "", Version: "v1", Kind: "ReplicationController"})
}

func (o *replicationControllerLocalObject) ResetStatus() {
	o.Object.(*corev1.ReplicationController).Status = corev1.ReplicationControllerStatus{}
}

func (o *replicationControllerLocalObject) GetObject() client.Object {
	return o.Object
}
