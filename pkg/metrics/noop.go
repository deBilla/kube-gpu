package metrics

import "github.com/dimuthu/kube-gpu/pkg/model"

type NoopProvider struct{}

func NewNoopProvider() *NoopProvider {
	return &NoopProvider{}
}

func (n *NoopProvider) GetMetrics(nodeName string) ([]model.GPUMetrics, error) {
	return nil, nil
}
