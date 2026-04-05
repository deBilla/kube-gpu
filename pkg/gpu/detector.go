package gpu

import (
	"context"
	"sort"
	"strings"

	"github.com/dimuthu/kube-gpu/pkg/model"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type Detector struct {
	client kubernetes.Interface
}

func NewDetector(client kubernetes.Interface) *Detector {
	return &Detector{client: client}
}

func (d *Detector) DetectGPUNodes(ctx context.Context, nodeFilter string) ([]model.GPUNode, error) {
	nodeList, err := d.client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var gpuNodes []model.GPUNode
	for _, node := range nodeList.Items {
		if nodeFilter != "" && node.Name != nodeFilter {
			continue
		}

		gpuNode, ok := buildGPUNode(node)
		if !ok {
			continue
		}
		gpuNodes = append(gpuNodes, gpuNode)
	}

	sort.Slice(gpuNodes, func(i, j int) bool {
		return gpuNodes[i].Name < gpuNodes[j].Name
	})

	return gpuNodes, nil
}

func buildGPUNode(node corev1.Node) (model.GPUNode, bool) {
	capacity := node.Status.Capacity
	allocatable := node.Status.Allocatable

	hasMIG := false
	hasFullGPU := false
	var migSlices []model.MIGSlice
	var totalFullGPUs int64

	// Scan capacity for GPU resources
	for resName, qty := range capacity {
		name := string(resName)
		if IsMIGResource(name) {
			hasMIG = true
			profile, err := ParseMIGProfile(name)
			if err != nil {
				continue
			}
			count := qty.Value()
			for i := int64(1); i <= count; i++ {
				migSlices = append(migSlices, model.MIGSlice{
					Profile:    profile,
					Index:      int(i),
					TotalCount: int(count),
				})
			}
		} else if name == ResourceFullGPU {
			hasFullGPU = true
			totalFullGPUs = qty.Value()
		}
	}

	if !hasMIG && !hasFullGPU {
		return model.GPUNode{}, false
	}

	gpuModel := parseGPUModel(node.Labels)
	pluginReady := true

	// Check if allocatable is zero while capacity is non-zero
	if hasFullGPU && !hasMIG {
		allocQty := allocatable[corev1.ResourceName(ResourceFullGPU)]
		if allocQty.Value() == 0 && totalFullGPUs > 0 {
			pluginReady = false
		}
	}

	gpuNode := model.GPUNode{
		Name:        node.Name,
		GPUModel:    gpuModel,
		PluginReady: pluginReady,
	}

	if hasMIG {
		gpuNode.Mode = model.GPUModeMIG
		// Sort MIG slices by profile name then index
		sort.Slice(migSlices, func(i, j int) bool {
			if migSlices[i].Profile.Name != migSlices[j].Profile.Name {
				// Larger profiles first
				return migSlices[i].Profile.Compute > migSlices[j].Profile.Compute
			}
			return migSlices[i].Index < migSlices[j].Index
		})

		// Build a single device representing the MIG-partitioned GPU
		device := model.GPUDevice{
			Index:     0,
			Model:     gpuModel,
			Mode:      model.GPUModeMIG,
			MIGSlices: migSlices,
		}
		gpuNode.Devices = []model.GPUDevice{device}
		gpuNode.TotalGPUs = 1 // MIG means at least 1 physical GPU
	} else {
		gpuNode.Mode = model.GPUModeFull
		for i := int64(0); i < totalFullGPUs; i++ {
			gpuNode.Devices = append(gpuNode.Devices, model.GPUDevice{
				Index: int(i),
				Model: gpuModel,
				Mode:  model.GPUModeFull,
			})
		}
		gpuNode.TotalGPUs = int(totalFullGPUs)
	}

	return gpuNode, true
}

func parseGPUModel(labels map[string]string) string {
	product, ok := labels[LabelGPUProduct]
	if !ok {
		return "unknown"
	}
	// Clean up the label value: "NVIDIA-A100-SXM4-80GB" -> "A100"
	product = strings.ReplaceAll(product, "NVIDIA-", "")
	parts := strings.Split(product, "-")
	if len(parts) > 0 {
		return parts[0]
	}
	return product
}
