package local

import (
	"os/exec"
	"strconv"
	"strings"
)

func detectAMD() ([]LocalGPU, []string) {
	// Check if rocm-smi is available
	_, err := exec.LookPath("rocm-smi")
	if err != nil {
		return nil, nil
	}

	var gpus []LocalGPU
	var warnings []string

	// Query GPU info
	out, err := exec.Command("rocm-smi", "--showproductname", "--showmeminfo", "vram", "--showuse", "--csv").Output()
	if err != nil {
		return nil, []string{"rocm-smi found but query failed: " + err.Error()}
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	if len(lines) < 2 {
		return nil, nil
	}

	// Parse CSV header and rows
	header := strings.Split(lines[0], ",")
	colIdx := make(map[string]int)
	for i, h := range header {
		colIdx[strings.TrimSpace(h)] = i
	}

	for _, line := range lines[1:] {
		fields := strings.Split(line, ",")
		if len(fields) < 2 {
			continue
		}

		gpu := LocalGPU{
			Vendor:      "AMD",
			MemoryUsed:  -1,
			UtilPercent: -1,
			Extra:       make(map[string]string),
		}

		if idx, ok := colIdx["device"]; ok && idx < len(fields) {
			gpu.Index, _ = strconv.Atoi(strings.TrimSpace(fields[idx]))
		}
		if idx, ok := colIdx["Card series"]; ok && idx < len(fields) {
			gpu.Name = strings.TrimSpace(fields[idx])
		}
		if idx, ok := colIdx["GPU use (%)"]; ok && idx < len(fields) {
			gpu.UtilPercent, _ = strconv.Atoi(strings.TrimSpace(fields[idx]))
		}
		if idx, ok := colIdx["VRAM Total Memory (B)"]; ok && idx < len(fields) {
			bytes, _ := strconv.ParseInt(strings.TrimSpace(fields[idx]), 10, 64)
			gpu.MemoryMB = bytes / (1024 * 1024)
		}
		if idx, ok := colIdx["VRAM Total Used Memory (B)"]; ok && idx < len(fields) {
			bytes, _ := strconv.ParseInt(strings.TrimSpace(fields[idx]), 10, 64)
			gpu.MemoryUsed = bytes / (1024 * 1024)
		}

		gpus = append(gpus, gpu)
	}

	return gpus, warnings
}
