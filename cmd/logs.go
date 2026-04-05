package cmd

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/dimuthu/kube-gpu/pkg/gpu"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	logsFollow    bool
	logsTail      int64
	logsContainer string
)

// CUDA and GPU-related error patterns to highlight
var gpuErrorPatterns = []string{
	"CUDA",
	"cuda",
	"NCCL",
	"nccl",
	"RuntimeError",
	"OutOfMemoryError",
	"OOM",
	"out of memory",
	"GPU",
	"XID",
	"cuDNN",
	"cublas",
	"NVML",
	"device-side assert",
	"illegal memory access",
	"unspecified launch failure",
}

func newLogsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "logs [pod-name]",
		Short: "Tail logs from a GPU pod with CUDA error highlighting",
		Long:  "Stream logs from a GPU-consuming pod. Lines containing CUDA/GPU errors are highlighted. If no pod is specified, lists GPU pods to choose from.",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			if runCtx.Demo {
				return printDemoLogs(runCtx.Out)
			}

			client := runCtx.Client

			if len(args) == 0 {
				return listGPUPods(ctx, client, runCtx.Out, runCtx.Namespace)
			}

			podName := args[0]
			ns := runCtx.Namespace
			if ns == "" {
				ns = "default"
			}

			return streamPodLogs(ctx, client, ns, podName, runCtx.Out)
		},
	}

	cmd.Flags().BoolVarP(&logsFollow, "follow", "f", false, "stream logs in real-time")
	cmd.Flags().Int64Var(&logsTail, "tail", 100, "number of recent lines to show")
	cmd.Flags().StringVarP(&logsContainer, "container", "c", "", "specific container name (default: first container)")

	return cmd
}

func listGPUPods(ctx context.Context, client kubernetes.Interface, w io.Writer, namespace string) error {
	detector := gpu.NewDetector(client)
	nodes, err := detector.DetectGPUNodes(ctx, "")
	if err != nil {
		return err
	}

	allocator := gpu.NewAllocator(client)
	nodes, pending, err := allocator.Allocate(ctx, nodes, namespace)
	if err != nil {
		return err
	}

	fmt.Fprintln(w, "GPU pods available for log streaming:")
	fmt.Fprintln(w)

	count := 0
	for _, node := range nodes {
		for _, dev := range node.Devices {
			if dev.Mode == "full" && dev.Pod != nil {
				fmt.Fprintf(w, "  %s/%s  (node: %s, resource: nvidia.com/gpu)\n",
					dev.Pod.Namespace, dev.Pod.Name, node.Name)
				count++
			}
			for _, slice := range dev.MIGSlices {
				if slice.Pod != nil {
					fmt.Fprintf(w, "  %s/%s  (node: %s, resource: %s)\n",
						slice.Pod.Namespace, slice.Pod.Name, node.Name, slice.Profile.Resource)
					count++
				}
			}
		}
	}

	for _, pod := range pending {
		fmt.Fprintf(w, "  %s/%s  (pending)\n", pod.Namespace, pod.Name)
		count++
	}

	if count == 0 {
		fmt.Fprintln(w, "  No GPU pods found.")
	} else {
		fmt.Fprintf(w, "\nRun: kube-gpu logs <pod-name> -n <namespace>\n")
	}

	return nil
}

func streamPodLogs(ctx context.Context, client kubernetes.Interface, namespace, podName string, w io.Writer) error {
	// Verify pod exists and uses GPU
	pod, err := client.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("pod %s/%s not found: %w", namespace, podName, err)
	}

	hasGPU := false
	for _, c := range pod.Spec.Containers {
		for resName := range c.Resources.Requests {
			if gpu.IsGPUResource(string(resName)) {
				hasGPU = true
				break
			}
		}
	}
	if !hasGPU {
		fmt.Fprintf(w, "Warning: pod %s/%s does not request GPU resources\n\n", namespace, podName)
	}

	opts := &corev1.PodLogOptions{
		Follow:    logsFollow,
		TailLines: &logsTail,
	}
	if logsContainer != "" {
		opts.Container = logsContainer
	}

	stream, err := client.CoreV1().Pods(namespace).GetLogs(podName, opts).Stream(ctx)
	if err != nil {
		return fmt.Errorf("failed to stream logs: %w", err)
	}
	defer stream.Close()

	scanner := bufio.NewScanner(stream)
	for scanner.Scan() {
		line := scanner.Text()
		if containsGPUError(line) {
			fmt.Fprintf(w, ">>> %s\n", line)
		} else {
			fmt.Fprintln(w, line)
		}
	}

	return scanner.Err()
}

func containsGPUError(line string) bool {
	upper := strings.ToUpper(line)
	for _, pattern := range gpuErrorPatterns {
		if strings.Contains(upper, strings.ToUpper(pattern)) {
			return true
		}
	}
	return false
}

func printDemoLogs(w io.Writer) error {
	demoLines := []string{
		"[2026-04-05 10:00:01] INFO  Loading model weights...",
		"[2026-04-05 10:00:02] INFO  CUDA device: NVIDIA A100-SXM4-80GB",
		"[2026-04-05 10:00:02] INFO  GPU memory: 79.35 GB available",
		"[2026-04-05 10:00:03] INFO  Loading checkpoint from /models/llama-7b/",
		"[2026-04-05 10:00:15] INFO  Model loaded successfully on cuda:0",
		"[2026-04-05 10:00:15] INFO  Starting training epoch 1/10",
		"[2026-04-05 10:00:20] INFO  Batch 1/1000 | Loss: 2.4531 | LR: 1e-5",
		"[2026-04-05 10:00:25] INFO  Batch 2/1000 | Loss: 2.3187 | LR: 1e-5",
		"[2026-04-05 10:00:30] WARN  CUDA warning: high memory usage (71.3/80 GB)",
		"[2026-04-05 10:00:35] INFO  Batch 3/1000 | Loss: 2.1842 | LR: 1e-5",
		"[2026-04-05 10:00:40] ERROR CUDA out of memory. Tried to allocate 2.00 GiB. GPU 0 has 79.35 GiB total, 71.30 GiB used.",
		"[2026-04-05 10:00:40] ERROR RuntimeError: CUDA error: out of memory",
		"[2026-04-05 10:00:41] INFO  Reducing batch size and retrying...",
		"[2026-04-05 10:00:42] INFO  Batch 3/1000 | Loss: 2.1790 | LR: 1e-5",
		"[2026-04-05 10:00:47] INFO  NCCL initialized with 1 GPU(s)",
		"[2026-04-05 10:00:50] INFO  Batch 4/1000 | Loss: 2.0556 | LR: 1e-5",
	}

	for _, line := range demoLines {
		if containsGPUError(line) {
			fmt.Fprintf(w, ">>> %s\n", line)
		} else {
			fmt.Fprintln(w, line)
		}
	}

	return nil
}
