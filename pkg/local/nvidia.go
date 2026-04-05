package local

import (
	"encoding/csv"
	"os/exec"
	"strconv"
	"strings"
)

func detectNvidia() ([]LocalGPU, []string) {
	// Check if nvidia-smi is available
	path, err := exec.LookPath("nvidia-smi")
	if err != nil {
		return nil, nil // Not an error — just no NVIDIA GPU
	}
	_ = path

	var gpus []LocalGPU
	var warnings []string

	// Query GPU info in CSV format
	out, err := exec.Command("nvidia-smi",
		"--query-gpu=index,name,memory.total,memory.used,utilization.gpu,driver_version,mig.mode.current",
		"--format=csv,noheader,nounits",
	).Output()
	if err != nil {
		return nil, []string{"nvidia-smi found but query failed: " + err.Error()}
	}

	// Get CUDA version separately
	cudaVersion := getCUDAVersion()

	reader := csv.NewReader(strings.NewReader(strings.TrimSpace(string(out))))
	records, err := reader.ReadAll()
	if err != nil {
		return nil, []string{"failed to parse nvidia-smi output: " + err.Error()}
	}

	for _, record := range records {
		if len(record) < 6 {
			continue
		}

		index, _ := strconv.Atoi(strings.TrimSpace(record[0]))
		name := strings.TrimSpace(record[1])
		memTotal, _ := strconv.ParseInt(strings.TrimSpace(record[2]), 10, 64)
		memUsed, _ := strconv.ParseInt(strings.TrimSpace(record[3]), 10, 64)
		util, _ := strconv.Atoi(strings.TrimSpace(record[4]))
		driver := strings.TrimSpace(record[5])

		gpu := LocalGPU{
			Name:        name,
			Vendor:      "NVIDIA",
			Index:       index,
			MemoryMB:    memTotal,
			MemoryUsed:  memUsed,
			UtilPercent: util,
			Driver:      driver,
			CUDAVersion: cudaVersion,
			Extra:       make(map[string]string),
		}

		// Check MIG mode
		if len(record) >= 7 {
			migMode := strings.TrimSpace(record[6])
			if strings.EqualFold(migMode, "enabled") {
				gpu.MIGEnabled = true
				gpu.MIGProfiles = getMIGProfiles(index)
			}
		}

		gpus = append(gpus, gpu)
	}

	return gpus, warnings
}

func getCUDAVersion() string {
	out, err := exec.Command("nvidia-smi", "--query-gpu=driver_version", "--format=csv,noheader").Output()
	if err != nil {
		return ""
	}
	// Get CUDA version from nvidia-smi header output
	headerOut, err := exec.Command("nvidia-smi").Output()
	if err != nil {
		return ""
	}
	_ = out
	lines := strings.Split(string(headerOut), "\n")
	for _, line := range lines {
		if strings.Contains(line, "CUDA Version") {
			parts := strings.Split(line, "CUDA Version:")
			if len(parts) >= 2 {
				return strings.TrimSpace(strings.TrimRight(parts[1], " |"))
			}
		}
	}
	return ""
}

func getMIGProfiles(gpuIndex int) []string {
	out, err := exec.Command("nvidia-smi", "mig", "-lgip",
		"-i", strconv.Itoa(gpuIndex),
	).Output()
	if err != nil {
		return nil
	}

	var profiles []string
	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Look for profile lines like "MIG 3g.20gb"
		if strings.Contains(line, "g.") && strings.Contains(line, "gb") {
			fields := strings.Fields(line)
			for _, f := range fields {
				if strings.Contains(f, "g.") && strings.Contains(f, "gb") {
					profiles = append(profiles, f)
					break
				}
			}
		}
	}

	return profiles
}
