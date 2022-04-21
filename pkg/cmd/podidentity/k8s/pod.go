package k8s

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type podLocalObject struct {
	client.Object
}

func newPodLocalObject(obj client.Object) LocalObject {
	return &podLocalObject{
		Object: obj,
	}
}

func (o *podLocalObject) GetServiceAccountName() string {
	return o.Object.(*corev1.Pod).Spec.ServiceAccountName
}

func (o *podLocalObject) SetServiceAccountName(name string) {
	o.Object.(*corev1.Pod).Spec.ServiceAccountName = name
}

func (o *podLocalObject) GetContainers() []corev1.Container {
	return o.Object.(*corev1.Pod).Spec.Containers
}

func (o *podLocalObject) SetContainers(containers []corev1.Container) {
	o.Object.(*corev1.Pod).Spec.Containers = containers
}

func (o *podLocalObject) GetInitContainers() []corev1.Container {
	return o.Object.(*corev1.Pod).Spec.InitContainers
}

func (o *podLocalObject) SetInitContainers(containers []corev1.Container) {
	o.Object.(*corev1.Pod).Spec.InitContainers = containers
}

func (o *podLocalObject) SetGVK() {
	o.Object.(*corev1.Pod).SetGroupVersionKind(schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Pod"})
}

func (o *podLocalObject) ResetStatus() {
	o.Object.(*corev1.Pod).Status = corev1.PodStatus{}
}

func (o *podLocalObject) GetObject() client.Object {
	return o.Object
}
