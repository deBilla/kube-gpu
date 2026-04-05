package cmd

import (
	"context"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var eventsAll bool

var gpuEventPatterns = []string{
	"FailedScheduling",
	"Insufficient nvidia.com",
	"nvidia.com/gpu",
	"nvidia.com/mig",
	"DevicePlugin",
	"TopologyAffinity",
	"UnexpectedAdmissionError",
}

func newEventsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "events",
		Short: "Show Kubernetes events related to GPU scheduling and allocation",
		Long:  "Displays cluster events filtered to GPU-related scheduling failures, resource issues, and device plugin events.",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			if runCtx.Demo {
				return printDemoEvents(runCtx.Out)
			}

			return showGPUEvents(ctx, runCtx.Client, runCtx.Out, runCtx.Namespace)
		},
	}

	cmd.Flags().BoolVarP(&eventsAll, "all", "A", false, "show all events, not just GPU-related")

	return cmd
}

func showGPUEvents(ctx context.Context, client kubernetes.Interface, w io.Writer, namespace string) error {
	ns := ""
	if namespace != "" {
		ns = namespace
	}

	eventList, err := client.CoreV1().Events(ns).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list events: %w", err)
	}

	var filtered []corev1.Event
	for _, event := range eventList.Items {
		if isGPURelatedEvent(event) {
			filtered = append(filtered, event)
		}
	}

	// Sort by last timestamp, newest first
	sort.Slice(filtered, func(i, j int) bool {
		ti := filtered[i].LastTimestamp.Time
		tj := filtered[j].LastTimestamp.Time
		if ti.IsZero() {
			ti = filtered[i].CreationTimestamp.Time
		}
		if tj.IsZero() {
			tj = filtered[j].CreationTimestamp.Time
		}
		return ti.After(tj)
	})

	if len(filtered) == 0 {
		fmt.Fprintln(w, "No GPU-related events found.")
		return nil
	}

	fmt.Fprintf(w, "%-6s %-6s %-28s %-22s %s\n", "AGE", "KIND", "NAME", "REASON", "MESSAGE")

	now := time.Now()
	for _, event := range filtered {
		ts := event.LastTimestamp.Time
		if ts.IsZero() {
			ts = event.CreationTimestamp.Time
		}

		age := formatAge(now.Sub(ts))
		msg := event.Message
		if len(msg) > 90 {
			msg = msg[:90] + "..."
		}

		fmt.Fprintf(w, "%-6s %-6s %-28s %-22s %s\n",
			age,
			event.InvolvedObject.Kind,
			event.InvolvedObject.Name,
			event.Reason,
			msg)
	}

	return nil
}

func isGPURelatedEvent(event corev1.Event) bool {
	if eventsAll {
		return true
	}

	searchText := strings.Join([]string{
		event.Reason,
		event.Message,
		event.InvolvedObject.Name,
	}, " ")

	for _, pattern := range gpuEventPatterns {
		if strings.Contains(searchText, pattern) {
			return true
		}
	}

	return false
}

func formatAge(d time.Duration) string {
	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}

func printDemoEvents(w io.Writer) error {
	type demoEvent struct {
		age     string
		kind    string
		name    string
		reason  string
		message string
	}

	events := []demoEvent{
		{"2m", "Pod", "training-llm-finetune", "FailedScheduling",
			"0/3 nodes are available: 3 Insufficient nvidia.com/mig-3g.20gb. preemption: 0/3 nodes are available."},
		{"2m", "Pod", "batch-inference-001", "FailedScheduling",
			"0/3 nodes are available: 2 node(s) didn't match Pod's node affinity, 1 Insufficient nvidia.com/gpu."},
		{"2m", "Pod", "experiment-jl-004", "FailedScheduling",
			"0/3 nodes are available: 2 Insufficient nvidia.com/mig-1g.5gb, 1 node(s) had untolerated taint."},
		{"15m", "Node", "gpu-node-01", "DevicePluginReady",
			"NVIDIA device plugin is ready. Detected 1 A100 GPU with MIG enabled (2x 3g.20gb, 1x 1g.5gb)."},
		{"15m", "Node", "gpu-node-02", "DevicePluginReady",
			"NVIDIA device plugin is ready. Detected 1 A100 GPU in full GPU mode."},
		{"15m", "Node", "gpu-node-03", "DevicePluginReady",
			"NVIDIA device plugin is ready. Detected 1 A100 GPU with MIG enabled (2x 3g.20gb, 1x 1g.5gb)."},
		{"20m", "Pod", "inference-api-7b4f", "Scheduled",
			"Successfully assigned default/inference-api-7b4f to gpu-node-01"},
		{"20m", "Pod", "training-job-8291", "Scheduled",
			"Successfully assigned default/training-job-8291 to gpu-node-01"},
		{"20m", "Pod", "training-diffusion-42", "Scheduled",
			"Successfully assigned ml-training/training-diffusion-42 to gpu-node-02"},
		{"25m", "Pod", "experiment-jl-003", "Scheduled",
			"Successfully assigned research/experiment-jl-003 to gpu-node-01"},
	}

	fmt.Fprintf(w, "%-6s %-6s %-28s %-22s %s\n", "AGE", "KIND", "NAME", "REASON", "MESSAGE")

	for _, e := range events {
		msg := e.message
		if len(msg) > 90 {
			msg = msg[:90] + "..."
		}
		fmt.Fprintf(w, "%-6s %-6s %-28s %-22s %s\n",
			e.age, e.kind, e.name, e.reason, msg)
	}

	return nil
}
