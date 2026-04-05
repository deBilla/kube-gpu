package metrics

import "github.com/deBilla/kube-gpu/pkg/model"

type Provider interface {
	GetMetrics(nodeName string) ([]model.GPUMetrics, error)
}
