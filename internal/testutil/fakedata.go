package testutil

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
)

func DemoCluster() *fake.Clientset {
	objects := []runtime.Object{
		// gpu-node-01: A100 in MIG mode (2x 3g.20gb, 1x 1g.5gb)
		NewGPUNode("gpu-node-01", map[string]string{
			"nvidia.com/gpu.product": "NVIDIA-A100-SXM4-80GB",
		}, map[corev1.ResourceName]resource.Quantity{
			"nvidia.com/mig-3g.20gb": resource.MustParse("2"),
			"nvidia.com/mig-1g.5gb":  resource.MustParse("1"),
		}),

		// gpu-node-02: A100 full GPU mode (1 GPU)
		NewGPUNode("gpu-node-02", map[string]string{
			"nvidia.com/gpu.product": "NVIDIA-A100-SXM4-80GB",
		}, map[corev1.ResourceName]resource.Quantity{
			"nvidia.com/gpu": resource.MustParse("1"),
		}),

		// gpu-node-03: A100 in MIG mode (2x 3g.20gb, 1x 1g.5gb) — all idle
		NewGPUNode("gpu-node-03", map[string]string{
			"nvidia.com/gpu.product": "NVIDIA-A100-SXM4-80GB",
		}, map[corev1.ResourceName]resource.Quantity{
			"nvidia.com/mig-3g.20gb": resource.MustParse("2"),
			"nvidia.com/mig-1g.5gb":  resource.MustParse("1"),
		}),

		// Pods on gpu-node-01
		NewGPUPod("inference-api-7b4f", "default", "gpu-node-01",
			map[corev1.ResourceName]resource.Quantity{
				"nvidia.com/mig-3g.20gb": resource.MustParse("1"),
			}, corev1.PodRunning),

		NewGPUPod("training-job-8291", "default", "gpu-node-01",
			map[corev1.ResourceName]resource.Quantity{
				"nvidia.com/mig-3g.20gb": resource.MustParse("1"),
			}, corev1.PodRunning),

		NewGPUPod("experiment-jl-003", "research", "gpu-node-01",
			map[corev1.ResourceName]resource.Quantity{
				"nvidia.com/mig-1g.5gb": resource.MustParse("1"),
			}, corev1.PodRunning),

		// Pod on gpu-node-02 (full GPU)
		NewGPUPod("training-diffusion-42", "ml-training", "gpu-node-02",
			map[corev1.ResourceName]resource.Quantity{
				"nvidia.com/gpu": resource.MustParse("1"),
			}, corev1.PodRunning),

		// Pending pods (not yet scheduled)
		NewGPUPod("training-llm-finetune", "ml-training", "",
			map[corev1.ResourceName]resource.Quantity{
				"nvidia.com/mig-3g.20gb": resource.MustParse("1"),
			}, corev1.PodPending),

		NewGPUPod("batch-inference-001", "default", "",
			map[corev1.ResourceName]resource.Quantity{
				"nvidia.com/gpu": resource.MustParse("1"),
			}, corev1.PodPending),

		NewGPUPod("experiment-jl-004", "research", "",
			map[corev1.ResourceName]resource.Quantity{
				"nvidia.com/mig-1g.5gb": resource.MustParse("1"),
			}, corev1.PodPending),
	}

	return fake.NewSimpleClientset(objects...)
}
