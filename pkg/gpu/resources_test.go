package gpu

import (
	"testing"
)

func TestIsGPUResource(t *testing.T) {
	tests := []struct {
		name     string
		resource string
		want     bool
	}{
		{"full gpu", "nvidia.com/gpu", true},
		{"mig 3g.20gb", "nvidia.com/mig-3g.20gb", true},
		{"mig 1g.5gb", "nvidia.com/mig-1g.5gb", true},
		{"mig 7g.40gb", "nvidia.com/mig-7g.40gb", true},
		{"cpu", "cpu", false},
		{"memory", "memory", false},
		{"other vendor", "amd.com/gpu", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsGPUResource(tt.resource); got != tt.want {
				t.Errorf("IsGPUResource(%q) = %v, want %v", tt.resource, got, tt.want)
			}
		})
	}
}

func TestIsMIGResource(t *testing.T) {
	tests := []struct {
		resource string
		want     bool
	}{
		{"nvidia.com/mig-3g.20gb", true},
		{"nvidia.com/mig-1g.5gb", true},
		{"nvidia.com/mig-7g.40gb", true},
		{"nvidia.com/mig-2g.10gb", true},
		{"nvidia.com/gpu", false},
		{"nvidia.com/mig-", false},
	}

	for _, tt := range tests {
		t.Run(tt.resource, func(t *testing.T) {
			if got := IsMIGResource(tt.resource); got != tt.want {
				t.Errorf("IsMIGResource(%q) = %v, want %v", tt.resource, got, tt.want)
			}
		})
	}
}

func TestParseMIGProfile(t *testing.T) {
	tests := []struct {
		resource    string
		wantName    string
		wantCompute int
		wantMem     int
		wantErr     bool
	}{
		{"nvidia.com/mig-3g.20gb", "3g.20gb", 3, 20, false},
		{"nvidia.com/mig-1g.5gb", "1g.5gb", 1, 5, false},
		{"nvidia.com/mig-7g.40gb", "7g.40gb", 7, 40, false},
		{"nvidia.com/gpu", "", 0, 0, true},
		{"invalid", "", 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.resource, func(t *testing.T) {
			profile, err := ParseMIGProfile(tt.resource)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseMIGProfile(%q) error = %v, wantErr %v", tt.resource, err, tt.wantErr)
				return
			}
			if err == nil {
				if profile.Name != tt.wantName {
					t.Errorf("Name = %q, want %q", profile.Name, tt.wantName)
				}
				if profile.Compute != tt.wantCompute {
					t.Errorf("Compute = %d, want %d", profile.Compute, tt.wantCompute)
				}
				if profile.MemoryGB != tt.wantMem {
					t.Errorf("MemoryGB = %d, want %d", profile.MemoryGB, tt.wantMem)
				}
			}
		})
	}
}
