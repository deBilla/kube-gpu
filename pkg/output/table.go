package output

import (
	"fmt"
	"io"
	"strings"

	"github.com/deBilla/kube-gpu/pkg/model"
)

type TableRenderer struct {
	w       io.Writer
	noColor bool
}

func NewTableRenderer(w io.Writer, noColor bool) *TableRenderer {
	return &TableRenderer{w: w, noColor: noColor}
}

func (r *TableRenderer) RenderStatus(nodes []model.GPUNode, pending []model.GPUPod) error {
	if len(nodes) == 0 {
		fmt.Fprintln(r.w, "No GPU nodes found in the cluster.")
		return nil
	}

	// Header
	fmt.Fprintf(r.w, "%-18s %-6s %-20s %-25s %-6s %s\n",
		"NODE", "GPU", "MIG-SLICES", "POD", "UTIL", "MEM")

	totalGPUs := 0
	allocatedGPUs := 0
	totalSlices := 0
	allocatedSlices := 0

	for _, node := range nodes {
		firstRow := true

		if !node.PluginReady {
			nodeName := node.Name
			if !firstRow {
				nodeName = ""
			}
			fmt.Fprintf(r.w, "%-18s %-6s %-20s %-25s %-6s %s\n",
				nodeName, node.GPUModel, "[PLUGIN NOT READY]", "-", "-", "-")
			totalGPUs += node.TotalGPUs
			continue
		}

		for _, dev := range node.Devices {
			if dev.Mode == model.GPUModeMIG {
				for _, slice := range dev.MIGSlices {
					nodeName := ""
					gpuModel := ""
					if firstRow {
						nodeName = node.Name
						gpuModel = node.GPUModel
						firstRow = false
					}

					podName := "<idle>"
					if slice.Pod != nil {
						podName = slice.Pod.Name
						allocatedSlices++
					}
					totalSlices++

					migLabel := fmt.Sprintf("%s (%d/%d)", slice.Profile.Name, slice.Index, slice.TotalCount)

					fmt.Fprintf(r.w, "%-18s %-6s %-20s %-25s %-6s %s\n",
						nodeName, gpuModel, migLabel, podName, "N/A", "N/A")
				}
			} else {
				nodeName := ""
				gpuModel := ""
				if firstRow {
					nodeName = node.Name
					gpuModel = node.GPUModel
					firstRow = false
				}

				podName := "<idle>"
				if dev.Pod != nil {
					podName = dev.Pod.Name
					allocatedGPUs++
				}
				totalGPUs++

				fmt.Fprintf(r.w, "%-18s %-6s %-20s %-25s %-6s %s\n",
					nodeName, gpuModel, "[FULL GPU]", podName, "N/A", "N/A")
			}
		}
	}

	fmt.Fprintln(r.w)
	parts := []string{
		fmt.Sprintf("QUEUE: %d pending jobs", len(pending)),
	}
	if totalGPUs > 0 {
		parts = append(parts, fmt.Sprintf("TOTAL: %d GPUs", totalGPUs))
		parts = append(parts, fmt.Sprintf("%d allocated", allocatedGPUs))
	}
	if totalSlices > 0 {
		parts = append(parts, fmt.Sprintf("%d MIG slices", totalSlices))
		parts = append(parts, fmt.Sprintf("%d allocated", allocatedSlices))
		parts = append(parts, fmt.Sprintf("%d idle", totalSlices-allocatedSlices))
	}
	fmt.Fprintln(r.w, strings.Join(parts, " | "))

	return nil
}

func (r *TableRenderer) RenderNodes(nodes []model.GPUNode) error {
	if len(nodes) == 0 {
		fmt.Fprintln(r.w, "No GPU nodes found in the cluster.")
		return nil
	}

	for i, node := range nodes {
		if i > 0 {
			fmt.Fprintln(r.w)
		}
		fmt.Fprintf(r.w, "Node: %s\n", node.Name)
		fmt.Fprintf(r.w, "  GPU Model:    %s\n", node.GPUModel)
		fmt.Fprintf(r.w, "  Mode:         %s\n", node.Mode)
		fmt.Fprintf(r.w, "  Plugin Ready: %v\n", node.PluginReady)
		fmt.Fprintf(r.w, "  Total GPUs:   %d\n", node.TotalGPUs)
		fmt.Fprintf(r.w, "  Allocated:    %d\n", node.AllocatedGPUs)

		if node.Mode == model.GPUModeMIG {
			fmt.Fprintln(r.w, "  MIG Slices:")
			for _, dev := range node.Devices {
				for _, slice := range dev.MIGSlices {
					podInfo := "<idle>"
					if slice.Pod != nil {
						podInfo = fmt.Sprintf("%s/%s", slice.Pod.Namespace, slice.Pod.Name)
					}
					fmt.Fprintf(r.w, "    %s (%d/%d) -> %s\n",
						slice.Profile.Name, slice.Index, slice.TotalCount, podInfo)
				}
			}
		} else {
			fmt.Fprintln(r.w, "  Devices:")
			for _, dev := range node.Devices {
				podInfo := "<idle>"
				if dev.Pod != nil {
					podInfo = fmt.Sprintf("%s/%s", dev.Pod.Namespace, dev.Pod.Name)
				}
				fmt.Fprintf(r.w, "    GPU %d -> %s\n", dev.Index, podInfo)
			}
		}
	}

	return nil
}

func (r *TableRenderer) RenderPods(pods []model.GPUPod) error {
	if len(pods) == 0 {
		fmt.Fprintln(r.w, "No GPU pods found.")
		return nil
	}

	fmt.Fprintf(r.w, "%-15s %-30s %-18s %-28s %-6s %s\n",
		"NAMESPACE", "POD", "NODE", "GPU-RESOURCE", "QTY", "STATUS")

	for _, pod := range pods {
		firstRes := true
		for resName, qty := range pod.GPURequest {
			ns := ""
			name := ""
			nodeName := ""
			status := ""
			if firstRes {
				ns = pod.Namespace
				name = pod.Name
				nodeName = pod.NodeName
				status = pod.Phase
				if nodeName == "" {
					nodeName = "<pending>"
				}
				firstRes = false
			}
			fmt.Fprintf(r.w, "%-15s %-30s %-18s %-28s %-6d %s\n",
				ns, name, nodeName, resName, qty, status)
		}
	}

	return nil
}
