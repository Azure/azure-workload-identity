package k8s

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type daemonSetLocalObject struct {
	client.Object
}

func newDaemonSetLocalObject(obj client.Object) LocalObject {
	return &daemonSetLocalObject{
		Object: obj,
	}
}

func (o *daemonSetLocalObject) GetServiceAccountName() string {
	return o.Object.(*appsv1.DaemonSet).Spec.Template.Spec.ServiceAccountName
}

func (o *daemonSetLocalObject) SetServiceAccountName(name string) {
	o.Object.(*appsv1.DaemonSet).Spec.Template.Spec.ServiceAccountName = name
}

func (o *daemonSetLocalObject) GetContainers() []corev1.Container {
	return o.Object.(*appsv1.DaemonSet).Spec.Template.Spec.Containers
}

func (o *daemonSetLocalObject) SetContainers(containers []corev1.Container) {
	o.Object.(*appsv1.DaemonSet).Spec.Template.Spec.Containers = containers
}

func (o *daemonSetLocalObject) GetInitContainers() []corev1.Container {
	return o.Object.(*appsv1.DaemonSet).Spec.Template.Spec.InitContainers
}

func (o *daemonSetLocalObject) SetInitContainers(containers []corev1.Container) {
	o.Object.(*appsv1.DaemonSet).Spec.Template.Spec.InitContainers = containers
}

func (o *daemonSetLocalObject) SetGVK() {
	o.Object.(*appsv1.DaemonSet).SetGroupVersionKind(schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "DaemonSet"})
}

func (o *daemonSetLocalObject) ResetStatus() {
	o.Object.(*appsv1.DaemonSet).Status = appsv1.DaemonSetStatus{}
}

func (o *daemonSetLocalObject) GetObject() client.Object {
	return o.Object
}
