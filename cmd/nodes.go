package cmd

import (
	"github.com/dimuthu/kube-gpu/internal/testutil"
	"github.com/dimuthu/kube-gpu/pkg/gpu"
	"github.com/dimuthu/kube-gpu/pkg/output"
	"github.com/spf13/cobra"
)

func newNodesCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "nodes",
		Short: "Show detailed GPU information per node",
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
			nodes, _, err = allocator.Allocate(ctx, nodes, runCtx.Namespace)
			if err != nil {
				return err
			}

			if runCtx.OutputFormat == "json" {
				return output.RenderJSON(runCtx.Out, nodes, nil)
			}

			renderer := output.NewTableRenderer(runCtx.Out, runCtx.NoColor)
			return renderer.RenderNodes(nodes)
		},
	}
}
