package cmd

import (
	"github.com/deBilla/kube-gpu/internal/testutil"
	"github.com/deBilla/kube-gpu/pkg/gpu"
	"github.com/deBilla/kube-gpu/pkg/model"
	"github.com/deBilla/kube-gpu/pkg/output"
	"github.com/spf13/cobra"
)

func newPodsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "pods",
		Short: "List pods consuming GPU resources",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			client := runCtx.Client
			if runCtx.Demo {
				client = testutil.DemoCluster()
			}

			detector := gpu.NewDetector(client)
			nodes, err := detector.DetectGPUNodes(ctx, runCtx.NodeFilter)
			if err != nil {
				return err
			}

			allocator := gpu.NewAllocator(client)
			nodes, pending, err := allocator.Allocate(ctx, nodes, runCtx.Namespace)
			if err != nil {
				return err
			}

			// Collect all GPU pods from allocated nodes
			var allPods []model.GPUPod
			for _, node := range nodes {
				for _, dev := range node.Devices {
					if dev.Mode == model.GPUModeFull && dev.Pod != nil {
						allPods = append(allPods, *dev.Pod)
					}
					for _, slice := range dev.MIGSlices {
						if slice.Pod != nil {
							allPods = append(allPods, *slice.Pod)
						}
					}
				}
			}
			allPods = append(allPods, pending...)

			if runCtx.OutputFormat == "json" {
				return output.RenderJSON(runCtx.Out, nil, allPods)
			}

			renderer := output.NewTableRenderer(runCtx.Out, runCtx.NoColor)
			return renderer.RenderPods(allPods)
		},
	}
}
