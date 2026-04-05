package cmd

import (
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/dimuthu/kube-gpu/pkg/kube"
	"github.com/dimuthu/kube-gpu/pkg/metrics"
	"github.com/spf13/cobra"
	"k8s.io/client-go/kubernetes"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

type RunContext struct {
	Client          kubernetes.Interface
	MetricsProvider metrics.Provider
	Out             io.Writer
	Namespace       string
	NodeFilter      string
	OutputFormat    string
	NoColor         bool
	Demo            bool
}

var (
	kubeconfig   string
	kubecontext  string
	namespace    string
	nodeFilter   string
	outputFormat string
	noColor      bool
	demo         bool
	runCtx       *RunContext
)

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "kube-gpu",
		Short: "Inspect GPU allocation across a Kubernetes cluster",
		Long:  "A CLI tool for viewing GPU and MIG partition allocation, pod assignments, and utilization across Kubernetes cluster nodes.",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			runCtx = &RunContext{
				Out:          os.Stdout,
				Namespace:    namespace,
				NodeFilter:   nodeFilter,
				OutputFormat: outputFormat,
				NoColor:      noColor,
				Demo:         demo,
			}

			if demo {
				runCtx.MetricsProvider = metrics.NewNoopProvider()
				return nil
			}

			client, err := kube.NewClient(kubeconfig, kubecontext)
			if err != nil {
				return err
			}
			runCtx.Client = client
			runCtx.MetricsProvider = metrics.NewNoopProvider()
			return nil
		},
		SilenceUsage: true,
	}

	// Detect kubectl plugin mode
	if strings.HasPrefix(filepath.Base(os.Args[0]), "kubectl-") {
		cmd.Use = "kubectl gpu"
	}

	cmd.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", "", "path to kubeconfig file (default: ~/.kube/config)")
	cmd.PersistentFlags().StringVar(&kubecontext, "context", "", "kubernetes context to use")
	cmd.PersistentFlags().StringVarP(&namespace, "namespace", "n", "", "filter pods by namespace (default: all namespaces)")
	cmd.PersistentFlags().StringVar(&nodeFilter, "node", "", "filter to a specific node")
	cmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "table", "output format: table or json")
	cmd.PersistentFlags().BoolVar(&noColor, "no-color", false, "disable colored output")
	cmd.PersistentFlags().BoolVar(&demo, "demo", false, "use built-in demo data (no cluster required)")

	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newStatusCmd())
	cmd.AddCommand(newNodesCmd())
	cmd.AddCommand(newPodsCmd())
	cmd.AddCommand(newLogsCmd())
	cmd.AddCommand(newEventsCmd())
	cmd.AddCommand(newLocalCmd())

	return cmd
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "version",
		Short: "Print version information",
		Run: func(cmd *cobra.Command, args []string) {
			cmd.Printf("kube-gpu %s (commit: %s, built: %s)\n", version, commit, date)
		},
	}
}

func Execute() error {
	return newRootCmd().Execute()
}
