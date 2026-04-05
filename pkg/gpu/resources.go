package gpu

import (
	"fmt"
	"regexp"
	"strconv"

	"github.com/deBilla/kube-gpu/pkg/model"
)

const (
	ResourceFullGPU   = "nvidia.com/gpu"
	ResourceMIGPrefix = "nvidia.com/mig-"
	LabelGPUProduct   = "nvidia.com/gpu.product"
)

var migResourceRe = regexp.MustCompile(`^nvidia\.com/mig-(\d+)g\.(\d+)gb$`)

func IsGPUResource(name string) bool {
	return name == ResourceFullGPU || IsMIGResource(name)
}

func IsMIGResource(name string) bool {
	return migResourceRe.MatchString(name)
}

func ParseMIGProfile(resourceName string) (model.MIGProfile, error) {
	matches := migResourceRe.FindStringSubmatch(resourceName)
	if matches == nil {
		return model.MIGProfile{}, fmt.Errorf("not a MIG resource: %s", resourceName)
	}

	compute, _ := strconv.Atoi(matches[1])
	memGB, _ := strconv.Atoi(matches[2])

	name := fmt.Sprintf("%sg.%sgb", matches[1], matches[2])

	return model.MIGProfile{
		Name:     name,
		Resource: resourceName,
		Compute:  compute,
		MemoryGB: memGB,
	}, nil
}
