package k8s

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type statefulSetLocalObject struct {
	client.Object
}

func newStatefulSetLocalObject(obj client.Object) LocalObject {
	return &statefulSetLocalObject{
		Object: obj,
	}
}

func (o *statefulSetLocalObject) GetServiceAccountName() string {
	return o.Object.(*appsv1.StatefulSet).Spec.Template.Spec.ServiceAccountName
}

func (o *statefulSetLocalObject) SetServiceAccountName(name string) {
	o.Object.(*appsv1.StatefulSet).Spec.Template.Spec.ServiceAccountName = name
}

func (o *statefulSetLocalObject) GetContainers() []corev1.Container {
	return o.Object.(*appsv1.StatefulSet).Spec.Template.Spec.Containers
}

func (o *statefulSetLocalObject) SetContainers(containers []corev1.Container) {
	o.Object.(*appsv1.StatefulSet).Spec.Template.Spec.Containers = containers
}

func (o *statefulSetLocalObject) GetInitContainers() []corev1.Container {
	return o.Object.(*appsv1.StatefulSet).Spec.Template.Spec.InitContainers
}

func (o *statefulSetLocalObject) SetInitContainers(containers []corev1.Container) {
	o.Object.(*appsv1.StatefulSet).Spec.Template.Spec.InitContainers = containers
}

func (o *statefulSetLocalObject) SetGVK() {
	o.Object.(*appsv1.StatefulSet).SetGroupVersionKind(schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "StatefulSet"})
}

func (o *statefulSetLocalObject) ResetStatus() {
	o.Object.(*appsv1.StatefulSet).Status = appsv1.StatefulSetStatus{}
}

func (o *statefulSetLocalObject) GetObject() client.Object {
	return o.Object
}
