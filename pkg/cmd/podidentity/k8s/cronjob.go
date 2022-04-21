package k8s

import (
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type cronJobLocalObject struct {
	client.Object
}

func newCronJobLocalObject(obj client.Object) LocalObject {
	return &cronJobLocalObject{
		Object: obj,
	}
}

func (o *cronJobLocalObject) GetServiceAccountName() string {
	return o.Object.(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec.ServiceAccountName
}

func (o *cronJobLocalObject) SetServiceAccountName(name string) {
	o.Object.(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec.ServiceAccountName = name
}

func (o *cronJobLocalObject) GetContainers() []corev1.Container {
	return o.Object.(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec.Containers
}

func (o *cronJobLocalObject) SetContainers(containers []corev1.Container) {
	o.Object.(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec.Containers = containers
}

func (o *cronJobLocalObject) GetInitContainers() []corev1.Container {
	return o.Object.(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec.InitContainers
}

func (o *cronJobLocalObject) SetInitContainers(containers []corev1.Container) {
	o.Object.(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec.InitContainers = containers
}

func (o *cronJobLocalObject) SetGVK() {
	o.Object.(*batchv1.CronJob).SetGroupVersionKind(schema.GroupVersionKind{Group: "batch", Version: "v1", Kind: "CronJob"})
}

func (o *cronJobLocalObject) ResetStatus() {
	o.Object.(*batchv1.CronJob).Status = batchv1.CronJobStatus{}
}

func (o *cronJobLocalObject) GetObject() client.Object {
	return o.Object
}
