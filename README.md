# kube-gpu

A CLI tool that gives you a real-time view of GPU allocation across a Kubernetes cluster. Think `kubectl top nodes` but specifically for GPUs -- showing MIG partitions, which pods are using which GPU slices, and which jobs are queued.

## Features

- **MIG-aware** -- understands NVIDIA Multi-Instance GPU partitions (1g.5gb, 2g.10gb, 3g.20gb, 7g.40gb, etc.)
- **Full GPU tracking** -- also handles non-MIG GPU allocation
- **Pod correlation** -- shows exactly which pod is consuming which GPU slice
- **Pending job queue** -- surfaces pods waiting for GPU resources
- **kubectl plugin** -- works as `kubectl gpu` when installed via krew
- **JSON output** -- pipe into `jq` or other tools with `-o json`
- **GPU log streaming** -- tail pod logs with CUDA/GPU error highlighting
- **Event tracking** -- surface GPU scheduling failures and device plugin events
- **Demo mode** -- try it without a cluster using `--demo`

## Quick Start

```bash
# Using Homebrew
brew install deBilla/tap/kube-gpu

# Or build from source
go install github.com/deBilla/kube-gpu@latest

# Or clone and build
git clone https://github.com/deBilla/kube-gpu.git
cd kube-gpu
make build
```

## Usage

### Status overview

The main command. Shows all GPU nodes, MIG slices, pod assignments, and a summary.

```bash
$ kube-gpu status
```

```
NODE               GPU    MIG-SLICES           POD                       UTIL   MEM
gpu-node-01        A100   3g.20gb (1/2)        inference-api-7b4f        N/A    N/A
                          3g.20gb (2/2)        training-job-8291         N/A    N/A
                          1g.5gb (1/1)         experiment-jl-003         N/A    N/A
gpu-node-02        A100   [FULL GPU]           training-diffusion-42     N/A    N/A
gpu-node-03        A100   3g.20gb (1/2)        <idle>                    N/A    N/A
                          3g.20gb (2/2)        <idle>                    N/A    N/A
                          1g.5gb (1/1)         <idle>                    N/A    N/A

QUEUE: 3 pending jobs | TOTAL: 1 GPUs | 1 allocated | 6 MIG slices | 3 allocated | 3 idle
```

### Per-node detail

```bash
$ kube-gpu nodes
```

```
Node: gpu-node-01
  GPU Model:    A100
  Mode:         mig
  Plugin Ready: true
  Total GPUs:   1
  Allocated:    0
  MIG Slices:
    3g.20gb (1/2) -> default/inference-api-7b4f
    3g.20gb (2/2) -> default/training-job-8291
    1g.5gb (1/1) -> research/experiment-jl-003

Node: gpu-node-02
  GPU Model:    A100
  Mode:         full
  Plugin Ready: true
  Total GPUs:   1
  Allocated:    1
  Devices:
    GPU 0 -> ml-training/training-diffusion-42
```

### List GPU pods

```bash
$ kube-gpu pods
```

```
NAMESPACE       POD                            NODE               GPU-RESOURCE                 QTY    STATUS
default         inference-api-7b4f             gpu-node-01        nvidia.com/mig-3g.20gb       1      Running
default         training-job-8291              gpu-node-01        nvidia.com/mig-3g.20gb       1      Running
research        experiment-jl-003              gpu-node-01        nvidia.com/mig-1g.5gb        1      Running
ml-training     training-diffusion-42          gpu-node-02        nvidia.com/gpu               1      Running
default         batch-inference-001            <pending>          nvidia.com/gpu               1      Pending
ml-training     training-llm-finetune          <pending>          nvidia.com/mig-3g.20gb       1      Pending
research        experiment-jl-004              <pending>          nvidia.com/mig-1g.5gb        1      Pending
```

### GPU pod logs

Stream logs from GPU pods with automatic CUDA/GPU error highlighting. Lines containing GPU-related keywords (CUDA, NCCL, OOM, RuntimeError, etc.) are prefixed with `>>>`.

```bash
$ kube-gpu logs training-job-8291 -n default
```

```
[2026-04-05 10:00:01] INFO  Loading model weights...
>>> [2026-04-05 10:00:02] INFO  CUDA device: NVIDIA A100-SXM4-80GB
>>> [2026-04-05 10:00:02] INFO  GPU memory: 79.35 GB available
[2026-04-05 10:00:03] INFO  Loading checkpoint from /models/llama-7b/
>>> [2026-04-05 10:00:15] INFO  Model loaded successfully on cuda:0
[2026-04-05 10:00:15] INFO  Starting training epoch 1/10
[2026-04-05 10:00:20] INFO  Batch 1/1000 | Loss: 2.4531 | LR: 1e-5
>>> [2026-04-05 10:00:30] WARN  CUDA warning: high memory usage (71.3/80 GB)
>>> [2026-04-05 10:00:40] ERROR CUDA out of memory. Tried to allocate 2.00 GiB.
>>> [2026-04-05 10:00:40] ERROR RuntimeError: CUDA error: out of memory
[2026-04-05 10:00:41] INFO  Reducing batch size and retrying...
>>> [2026-04-05 10:00:47] INFO  NCCL initialized with 1 GPU(s)
```

Follow logs in real-time:

```bash
$ kube-gpu logs training-job-8291 -f --tail 50
```

Without a pod name, lists all GPU pods available for log streaming:

```bash
$ kube-gpu logs
```

### GPU events

Show Kubernetes events related to GPU scheduling failures, device plugin status, and resource issues.

```bash
$ kube-gpu events
```

```
AGE    KIND   NAME                         REASON                 MESSAGE
2m     Pod    training-llm-finetune        FailedScheduling       0/3 nodes are available: 3 Insufficient nvidia.com/mig-3g.20gb...
2m     Pod    batch-inference-001          FailedScheduling       0/3 nodes are available: 1 Insufficient nvidia.com/gpu...
2m     Pod    experiment-jl-004            FailedScheduling       0/3 nodes are available: 2 Insufficient nvidia.com/mig-1g.5gb...
15m    Node   gpu-node-01                  DevicePluginReady      NVIDIA device plugin is ready. Detected 1 A100 GPU with MIG...
20m    Pod    inference-api-7b4f           Scheduled              Successfully assigned default/inference-api-7b4f to gpu-node-01
```

### JSON output

```bash
$ kube-gpu status -o json | jq '.nodes[].Name'
"gpu-node-01"
"gpu-node-02"
"gpu-node-03"
```

### Demo mode

Try it out without a GPU cluster:

```bash
$ kube-gpu status --demo
```

## Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--kubeconfig` | | Path to kubeconfig file (default: `~/.kube/config`) |
| `--context` | | Kubernetes context to use |
| `--namespace` | `-n` | Filter pods by namespace (default: all) |
| `--node` | | Filter to a specific node |
| `--output` | `-o` | Output format: `table` or `json` |
| `--no-color` | | Disable colored output |
| `--demo` | | Use built-in demo data (no cluster required) |

### Logs-specific flags

| Flag | Short | Description |
|------|-------|-------------|
| `--follow` | `-f` | Stream logs in real-time |
| `--tail` | | Number of recent lines to show (default: 100) |
| `--container` | `-c` | Specific container name |

### Events-specific flags

| Flag | Short | Description |
|------|-------|-------------|
| `--all` | `-A` | Show all events, not just GPU-related |

## How it works

kube-gpu reads GPU allocation data directly from the Kubernetes API:

1. **Node scanning** -- Lists nodes and checks `.status.capacity` for NVIDIA GPU resources (`nvidia.com/gpu`, `nvidia.com/mig-*`)
2. **MIG detection** -- Nodes with `nvidia.com/mig-*` resources are in MIG mode; those with only `nvidia.com/gpu` are in full GPU mode
3. **Pod correlation** -- Lists all pods requesting GPU resources and matches them to nodes via `pod.spec.nodeName`
4. **Queue detection** -- Pending pods with GPU requests are surfaced as the job queue

### Supported GPU resources

| Resource | Description |
|----------|-------------|
| `nvidia.com/gpu` | Full GPU (non-MIG) |
| `nvidia.com/mig-1g.5gb` | MIG 1g.5gb slice |
| `nvidia.com/mig-2g.10gb` | MIG 2g.10gb slice |
| `nvidia.com/mig-3g.20gb` | MIG 3g.20gb slice |
| `nvidia.com/mig-7g.40gb` | MIG 7g.40gb slice |

GPU model is detected from the `nvidia.com/gpu.product` node label set by the NVIDIA device plugin.

## kubectl plugin

kube-gpu can also be used as a kubectl plugin:

```bash
# Via krew (once published)
kubectl krew install gpu
kubectl gpu status

# Or manually: rename/symlink the binary
ln -s kube-gpu /usr/local/bin/kubectl-gpu
kubectl gpu status
```

## Building from source

```bash
git clone https://github.com/deBilla/kube-gpu.git
cd kube-gpu

# Build
make build

# Run tests
make test

# Install to $GOPATH/bin
make install

# Test goreleaser locally
make release-snapshot
```

## Releasing

Releases are automated via GitHub Actions and goreleaser. To create a release:

```bash
git tag v0.1.0
git push origin v0.1.0
```

This will:
- Build binaries for linux/darwin on amd64/arm64
- Create a GitHub Release with the binaries
- Update the Homebrew tap formula

## Project structure

```
kube-gpu/
├── main.go                     # Entry point
├── cmd/                        # CLI commands (cobra)
│   ├── root.go                 # Root command, flags, RunContext
│   ├── status.go               # kube-gpu status
│   ├── nodes.go                # kube-gpu nodes
│   ├── pods.go                 # kube-gpu pods
│   ├── logs.go                 # kube-gpu logs (CUDA error highlighting)
│   └── events.go               # kube-gpu events (scheduling failures)
├── pkg/
│   ├── gpu/
│   │   ├── resources.go        # nvidia.com/* resource parsing
│   │   ├── detector.go         # Scan nodes for GPU capacity
│   │   └── allocator.go        # Correlate pods to GPU slices
│   ├── model/types.go          # Domain types (GPUNode, MIGSlice, etc.)
│   ├── output/
│   │   ├── table.go            # Table renderer
│   │   └── json.go             # JSON renderer
│   ├── metrics/
│   │   ├── provider.go         # MetricsProvider interface
│   │   └── noop.go             # Default no-op provider
│   └── kube/client.go          # Kubernetes client factory
├── internal/testutil/          # Test fixtures and fake cluster data
├── deploy/krew/                # kubectl krew plugin manifest
├── .goreleaser.yml             # Release configuration
└── .github/workflows/          # CI and release pipelines
```

## Roadmap

- [ ] GPU utilization metrics via Prometheus/DCGM exporter (`--prometheus-url`)
- [ ] Interactive TUI watch mode (`kube-gpu status --watch`)
- [ ] Time-sliced GPU detection and annotation
- [ ] Publish to krew plugin index
- [ ] Namespace-scoped RBAC support

## License

MIT
