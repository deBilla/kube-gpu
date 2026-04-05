package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/dimuthu/kube-gpu/pkg/local"
	"github.com/spf13/cobra"
)

func newLocalCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "local",
		Short: "Detect and display GPU capabilities of the local machine",
		Long:  "Scans the local machine for GPUs (NVIDIA, Apple Silicon, AMD) and displays capabilities, memory, utilization, and ML framework compatibility.",
		RunE: func(cmd *cobra.Command, args []string) error {
			info, err := local.Detect()
			if err != nil {
				return err
			}

			if runCtx != nil && runCtx.OutputFormat == "json" {
				enc := json.NewEncoder(cmd.OutOrStdout())
				enc.SetIndent("", "  ")
				return enc.Encode(info)
			}

			w := cmd.OutOrStdout()
			if runCtx != nil {
				w = runCtx.Out
			}
			return renderLocalGPUs(w, info)
		},
	}
}

func renderLocalGPUs(w io.Writer, info *local.LocalGPUInfo) error {
	fmt.Fprintf(w, "System: %s/%s\n\n", info.OS, info.Arch)

	if len(info.GPUs) == 0 {
		fmt.Fprintln(w, "No GPUs detected.")
		for _, warn := range info.Warnings {
			fmt.Fprintf(w, "  Warning: %s\n", warn)
		}
		return nil
	}

	for i, gpu := range info.GPUs {
		if i > 0 {
			fmt.Fprintln(w)
		}

		fmt.Fprintf(w, "GPU %d: %s (%s)\n", gpu.Index, gpu.Name, gpu.Vendor)
		fmt.Fprintln(w, strings.Repeat("-", 50))

		// Memory
		if gpu.MemoryMB > 0 {
			memStr := formatMemoryMB(gpu.MemoryMB)
			memType := ""
			if t, ok := gpu.Extra["memory_type"]; ok {
				memType = fmt.Sprintf(" (%s)", t)
			}
			if gpu.MemoryUsed >= 0 {
				usedStr := formatMemoryMB(gpu.MemoryUsed)
				fmt.Fprintf(w, "  Memory:       %s / %s%s\n", usedStr, memStr, memType)
			} else {
				fmt.Fprintf(w, "  Memory:       %s%s\n", memStr, memType)
			}
		}

		// Utilization
		if gpu.UtilPercent >= 0 {
			fmt.Fprintf(w, "  Utilization:  %d%%\n", gpu.UtilPercent)
		}

		// Driver
		if gpu.Driver != "" {
			fmt.Fprintf(w, "  Driver:       %s\n", gpu.Driver)
		}

		// CUDA
		if gpu.CUDAVersion != "" {
			fmt.Fprintf(w, "  CUDA:         %s\n", gpu.CUDAVersion)
		}

		// Metal
		if gpu.MetalFamily != "" {
			fmt.Fprintf(w, "  Metal:        %s\n", gpu.MetalFamily)
		}
		if ms, ok := gpu.Extra["metal_support"]; ok {
			fmt.Fprintf(w, "  Metal Support: %s\n", ms)
		}

		// GPU cores (Apple)
		if cores, ok := gpu.Extra["gpu_cores"]; ok {
			fmt.Fprintf(w, "  GPU Cores:    %s\n", cores)
		}

		// MPS (Apple Silicon)
		if gpu.MPS {
			fmt.Fprintf(w, "  MPS:          supported\n")
		}

		// PyTorch device
		if dev, ok := gpu.Extra["pytorch_device"]; ok {
			fmt.Fprintf(w, "  PyTorch:      torch.device(\"%s\")\n", dev)
		}

		// MIG
		if gpu.MIGEnabled {
			fmt.Fprintf(w, "  MIG:          enabled\n")
			if len(gpu.MIGProfiles) > 0 {
				fmt.Fprintf(w, "  MIG Profiles: %s\n", strings.Join(gpu.MIGProfiles, ", "))
			}
		}

		// ML framework compatibility summary
		fmt.Fprintln(w)
		fmt.Fprintln(w, "  ML Frameworks:")
		switch gpu.Vendor {
		case "NVIDIA":
			fmt.Fprintln(w, "    PyTorch:      torch.device(\"cuda\")")
			fmt.Fprintln(w, "    TensorFlow:   tf.device(\"/GPU:0\")")
			fmt.Fprintln(w, "    JAX:          jax.devices(\"gpu\")")
		case "Apple":
			if gpu.MPS {
				fmt.Fprintln(w, "    PyTorch:      torch.device(\"mps\")")
				fmt.Fprintln(w, "    TensorFlow:   tensorflow-metal plugin")
				fmt.Fprintln(w, "    MLX:          supported (Apple native)")
			}
		case "AMD":
			fmt.Fprintln(w, "    PyTorch:      torch.device(\"cuda\") via ROCm")
			fmt.Fprintln(w, "    TensorFlow:   tensorflow-rocm")
		}
	}

	if len(info.Warnings) > 0 {
		fmt.Fprintln(w)
		for _, warn := range info.Warnings {
			fmt.Fprintf(w, "Warning: %s\n", warn)
		}
	}

	return nil
}

func formatMemoryMB(mb int64) string {
	if mb >= 1024 {
		gb := float64(mb) / 1024.0
		if gb == float64(int(gb)) {
			return fmt.Sprintf("%d GB", int(gb))
		}
		return fmt.Sprintf("%.1f GB", gb)
	}
	return fmt.Sprintf("%d MB", mb)
}
