package gpu

import (
	"context"

	"github.com/deBilla/kube-gpu/pkg/model"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type Allocator struct {
	client kubernetes.Interface
}

func NewAllocator(client kubernetes.Interface) *Allocator {
	return &Allocator{client: client}
}

func (a *Allocator) Allocate(ctx context.Context, nodes []model.GPUNode, namespace string) ([]model.GPUNode, []model.GPUPod, error) {
	opts := metav1.ListOptions{
		FieldSelector: "status.phase!=Succeeded,status.phase!=Failed",
	}

	ns := ""
	if namespace != "" {
		ns = namespace
	}

	podList, err := a.client.CoreV1().Pods(ns).List(ctx, opts)
	if err != nil {
		return nil, nil, err
	}

	var gpuPods []model.GPUPod
	for _, p := range podList.Items {
		if gpuPod, ok := extractGPUPod(p); ok {
			gpuPods = append(gpuPods, gpuPod)
		}
	}

	// Build node name -> index map
	nodeIdx := make(map[string]int)
	for i := range nodes {
		nodeIdx[nodes[i].Name] = i
	}

	var pending []model.GPUPod

	for i := range gpuPods {
		pod := &gpuPods[i]
		if pod.Phase == "Pending" {
			pending = append(pending, *pod)
			continue
		}

		idx, ok := nodeIdx[pod.NodeName]
		if !ok {
			continue
		}

		assignPodToNode(&nodes[idx], pod)
	}

	for i := range nodes {
		nodes[i].AllocatedGPUs = countAllocated(&nodes[i])
	}

	return nodes, pending, nil
}

func extractGPUPod(pod corev1.Pod) (model.GPUPod, bool) {
	gpuReqs := make(map[string]int64)

	for _, container := range pod.Spec.Containers {
		for resName, qty := range container.Resources.Requests {
			name := string(resName)
			if IsGPUResource(name) {
				gpuReqs[name] += qty.Value()
			}
		}
	}

	if len(gpuReqs) == 0 {
		return model.GPUPod{}, false
	}

	return model.GPUPod{
		Name:       pod.Name,
		Namespace:  pod.Namespace,
		NodeName:   pod.Spec.NodeName,
		Phase:      string(pod.Status.Phase),
		GPURequest: gpuReqs,
	}, true
}

func assignPodToNode(node *model.GPUNode, pod *model.GPUPod) {
	for resName, qty := range pod.GPURequest {
		remaining := qty

		if IsMIGResource(resName) {
			for d := range node.Devices {
				for s := range node.Devices[d].MIGSlices {
					if remaining <= 0 {
						break
					}
					slice := &node.Devices[d].MIGSlices[s]
					if slice.Profile.Resource == resName && slice.Pod == nil {
						slice.Pod = pod
						remaining--
					}
				}
			}
		} else if resName == ResourceFullGPU {
			for d := range node.Devices {
				if remaining <= 0 {
					break
				}
				dev := &node.Devices[d]
				if dev.Mode == model.GPUModeFull && dev.Pod == nil {
					dev.Pod = pod
					remaining--
				}
			}
		}
	}
}

func countAllocated(node *model.GPUNode) int {
	count := 0
	for _, dev := range node.Devices {
		if dev.Mode == model.GPUModeFull && dev.Pod != nil {
			count++
		}
	}
	return count
}
