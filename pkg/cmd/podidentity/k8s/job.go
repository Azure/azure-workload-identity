package k8s

import (
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type jobLocalObject struct {
	client.Object
}

func newJobLocalObject(obj client.Object) LocalObject {
	return &jobLocalObject{
		Object: obj,
	}
}

func (o *jobLocalObject) GetServiceAccountName() string {
	return o.Object.(*batchv1.Job).Spec.Template.Spec.ServiceAccountName
}

func (o *jobLocalObject) SetServiceAccountName(name string) {
	o.Object.(*batchv1.Job).Spec.Template.Spec.ServiceAccountName = name
}

func (o *jobLocalObject) GetContainers() []corev1.Container {
	return o.Object.(*batchv1.Job).Spec.Template.Spec.Containers
}

func (o *jobLocalObject) SetContainers(containers []corev1.Container) {
	o.Object.(*batchv1.Job).Spec.Template.Spec.Containers = containers
}

func (o *jobLocalObject) GetInitContainers() []corev1.Container {
	return o.Object.(*batchv1.Job).Spec.Template.Spec.InitContainers
}

func (o *jobLocalObject) SetInitContainers(containers []corev1.Container) {
	o.Object.(*batchv1.Job).Spec.Template.Spec.InitContainers = containers
}

func (o *jobLocalObject) SetGVK() {
	o.Object.(*batchv1.Job).SetGroupVersionKind(schema.GroupVersionKind{Group: "batch", Version: "v1", Kind: "Job"})
}

func (o *jobLocalObject) ResetStatus() {
	o.Object.(*batchv1.Job).Status = batchv1.JobStatus{}
}

func (o *jobLocalObject) GetObject() client.Object {
	return o.Object
}
