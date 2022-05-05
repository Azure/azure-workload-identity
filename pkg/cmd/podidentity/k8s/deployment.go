package k8s

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type deploymentLocalObject struct {
	client.Object
}

func newDeploymentLocalObject(obj client.Object) LocalObject {
	return &deploymentLocalObject{
		Object: obj,
	}
}

func (o *deploymentLocalObject) GetServiceAccountName() string {
	return o.Object.(*appsv1.Deployment).Spec.Template.Spec.ServiceAccountName
}

func (o *deploymentLocalObject) SetServiceAccountName(name string) {
	o.Object.(*appsv1.Deployment).Spec.Template.Spec.ServiceAccountName = name
}

func (o *deploymentLocalObject) GetContainers() []corev1.Container {
	return o.Object.(*appsv1.Deployment).Spec.Template.Spec.Containers
}

func (o *deploymentLocalObject) SetContainers(containers []corev1.Container) {
	o.Object.(*appsv1.Deployment).Spec.Template.Spec.Containers = containers
}

func (o *deploymentLocalObject) GetInitContainers() []corev1.Container {
	return o.Object.(*appsv1.Deployment).Spec.Template.Spec.InitContainers
}

func (o *deploymentLocalObject) SetInitContainers(containers []corev1.Container) {
	o.Object.(*appsv1.Deployment).Spec.Template.Spec.InitContainers = containers
}

func (o *deploymentLocalObject) SetGVK() {
	o.Object.(*appsv1.Deployment).SetGroupVersionKind(schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"})
}

func (o *deploymentLocalObject) ResetStatus() {
	o.Object.(*appsv1.Deployment).Status = appsv1.DeploymentStatus{}
}

func (o *deploymentLocalObject) GetObject() client.Object {
	return o.Object
}
