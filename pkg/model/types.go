package model

type GPUMode string

const (
	GPUModeFull GPUMode = "full"
	GPUModeMIG  GPUMode = "mig"
)

type MIGProfile struct {
	Name     string // e.g. "3g.20gb"
	Resource string // e.g. "nvidia.com/mig-3g.20gb"
	Compute  int    // GPU compute units (e.g. 3)
	MemoryGB int    // Memory in GB (e.g. 20)
}

type MIGSlice struct {
	Profile    MIGProfile
	Index      int     // 1-based index within this profile on this node
	TotalCount int     // total slices of this profile on this node
	Pod        *GPUPod // nil if idle
}

type GPUDevice struct {
	Index     int
	Model     string
	Mode      GPUMode
	MIGSlices []MIGSlice // populated if Mode == GPUModeMIG
	Pod       *GPUPod    // populated if Mode == GPUModeFull and allocated
}

type GPUNode struct {
	Name          string
	GPUModel      string // from label nvidia.com/gpu.product
	Mode          GPUMode
	Devices       []GPUDevice
	TotalGPUs     int
	AllocatedGPUs int
	PluginReady   bool // false if capacity > 0 but allocatable == 0
}

type GPUPod struct {
	Name       string
	Namespace  string
	NodeName   string
	Phase      string
	GPURequest map[string]int64 // resource name -> quantity
}

type GPUMetrics struct {
	NodeName      string
	GPUIndex      int
	MIGProfile    string
	UtilPercent   *float64
	MemUsedBytes  *int64
	MemTotalBytes *int64
}

type ClusterGPUSummary struct {
	TotalGPUs       int
	AllocatedGPUs   int
	TotalMIGSlices  int
	AllocatedSlices int
	IdleSlices      int
	PendingPods     int
}
