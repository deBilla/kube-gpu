package metrics

import "github.com/dimuthu/kube-gpu/pkg/model"

type Provider interface {
	GetMetrics(nodeName string) ([]model.GPUMetrics, error)
}
