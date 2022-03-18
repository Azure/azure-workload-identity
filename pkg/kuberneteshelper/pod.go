package kuberneteshelper

import (
	"context"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ListPods(ctx context.Context, kubeClient client.Client, namespace string, labels map[string]string) (map[string]corev1.Pod, error) {
	list := &corev1.PodList{}
	if err := kubeClient.List(ctx, list, client.InNamespace(namespace), client.MatchingLabels(labels)); err != nil {
		return nil, err
	}

	podMap := make(map[string]corev1.Pod)
	for _, pod := range list.Items {
		podMap[pod.Name] = pod
	}

	return podMap, nil
}
