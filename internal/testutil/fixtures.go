package testutil

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewGPUNode(name string, labels map[string]string, capacity map[corev1.ResourceName]resource.Quantity) *corev1.Node {
	return &corev1.Node{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		Status: corev1.NodeStatus{
			Capacity:    capacity,
			Allocatable: capacity,
		},
	}
}

func NewGPUPod(name, namespace, nodeName string, requests map[corev1.ResourceName]resource.Quantity, phase corev1.PodPhase) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1.PodSpec{
			NodeName: nodeName,
			Containers: []corev1.Container{
				{
					Name: "main",
					Resources: corev1.ResourceRequirements{
						Requests: requests,
						Limits:   requests,
					},
				},
			},
		},
		Status: corev1.PodStatus{
			Phase: phase,
		},
	}
}
