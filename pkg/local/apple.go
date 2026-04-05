package local

import (
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

func detectApple() ([]LocalGPU, []string) {
	if runtime.GOOS != "darwin" {
		return nil, nil
	}

	var gpus []LocalGPU
	var warnings []string

	// Get display/GPU info from system_profiler
	out, err := exec.Command("system_profiler", "SPDisplaysDataType").Output()
	if err != nil {
		return nil, []string{"system_profiler failed: " + err.Error()}
	}

	output := string(out)

	// Parse chipset/model
	chipset := extractField(output, "Chipset Model")
	if chipset == "" {
		return nil, nil
	}

	gpu := LocalGPU{
		Name:        chipset,
		Vendor:      "Apple",
		Index:       0,
		MemoryUsed:  -1,
		UtilPercent: -1,
		Extra:       make(map[string]string),
	}

	// Parse VRAM or unified memory
	vram := extractField(output, "VRAM")
	if vram == "" {
		// Apple Silicon uses unified memory — get from sysctl
		memMB := getUnifiedMemory()
		if memMB > 0 {
			gpu.MemoryMB = memMB
			gpu.Extra["memory_type"] = "unified"
		}
	} else {
		gpu.MemoryMB = parseMemoryString(vram)
		gpu.Extra["memory_type"] = "dedicated"
	}

	// Metal family
	metalFamily := extractField(output, "Metal Family")
	if metalFamily != "" {
		gpu.MetalFamily = metalFamily
	}

	// Metal support
	metalSupport := extractField(output, "Metal Support")
	if metalSupport != "" {
		gpu.Extra["metal_support"] = metalSupport
	}

	// Check if Apple Silicon (MPS capable)
	if isAppleSilicon() {
		gpu.MPS = true
		gpu.Extra["pytorch_device"] = "mps"
	}

	// GPU cores from sysctl
	cores := getGPUCoreCount()
	if cores > 0 {
		gpu.Extra["gpu_cores"] = strconv.Itoa(cores)
	}

	gpus = append(gpus, gpu)

	return gpus, warnings
}

func extractField(output, field string) string {
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, field+":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return ""
}

func parseMemoryString(s string) int64 {
	s = strings.ToLower(strings.TrimSpace(s))
	re := regexp.MustCompile(`(\d+)\s*(gb|mb|tb)`)
	matches := re.FindStringSubmatch(s)
	if matches == nil {
		return 0
	}

	val, _ := strconv.ParseInt(matches[1], 10, 64)
	switch matches[2] {
	case "tb":
		return val * 1024 * 1024
	case "gb":
		return val * 1024
	case "mb":
		return val
	}
	return 0
}

func getUnifiedMemory() int64 {
	out, err := exec.Command("sysctl", "-n", "hw.memsize").Output()
	if err != nil {
		return 0
	}
	bytes, err := strconv.ParseInt(strings.TrimSpace(string(out)), 10, 64)
	if err != nil {
		return 0
	}
	return bytes / (1024 * 1024) // Convert to MB
}

func isAppleSilicon() bool {
	return runtime.GOARCH == "arm64" && runtime.GOOS == "darwin"
}

func getGPUCoreCount() int {
	out, err := exec.Command("sysctl", "-n", "machdep.cpu.gpu_core_count").Output()
	if err != nil {
		// Try alternative
		out, err = exec.Command("sysctl", "-n", "hw.perflevel0.gpu_count").Output()
		if err != nil {
			return 0
		}
	}
	cores, err := strconv.Atoi(strings.TrimSpace(string(out)))
	if err != nil {
		return 0
	}
	return cores
}
