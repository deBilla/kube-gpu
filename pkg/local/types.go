package local

// LocalGPU represents a GPU detected on the local machine.
type LocalGPU struct {
	Name        string
	Vendor      string // "NVIDIA", "Apple", "AMD", "Intel"
	Index       int
	MemoryMB    int64
	MemoryUsed  int64  // -1 if unknown
	UtilPercent int    // -1 if unknown
	Driver      string
	CUDAVersion string // NVIDIA only
	MetalFamily string // Apple only
	MPS         bool   // Apple Metal Performance Shaders
	MIGEnabled  bool   // NVIDIA MIG
	MIGProfiles []string
	Extra       map[string]string // additional properties
}

// LocalGPUInfo holds the complete local GPU report.
type LocalGPUInfo struct {
	OS       string
	Arch     string
	GPUs     []LocalGPU
	Warnings []string
}
