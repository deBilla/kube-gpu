package local

import "runtime"

// Detect detects all GPUs on the local machine.
func Detect() (*LocalGPUInfo, error) {
	info := &LocalGPUInfo{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}

	// Try NVIDIA first (works on Linux, Windows, and some macOS)
	nvidiaGPUs, nvidiaWarnings := detectNvidia()
	info.GPUs = append(info.GPUs, nvidiaGPUs...)
	info.Warnings = append(info.Warnings, nvidiaWarnings...)

	// On macOS, detect Apple Silicon / Metal GPUs
	if runtime.GOOS == "darwin" {
		appleGPUs, appleWarnings := detectApple()
		info.GPUs = append(info.GPUs, appleGPUs...)
		info.Warnings = append(info.Warnings, appleWarnings...)
	}

	// On Linux, try AMD ROCm
	if runtime.GOOS == "linux" {
		amdGPUs, amdWarnings := detectAMD()
		info.GPUs = append(info.GPUs, amdGPUs...)
		info.Warnings = append(info.Warnings, amdWarnings...)
	}

	if len(info.GPUs) == 0 {
		info.Warnings = append(info.Warnings, "No GPUs detected on this machine")
	}

	return info, nil
}
