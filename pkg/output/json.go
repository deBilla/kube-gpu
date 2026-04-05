package output

import (
	"encoding/json"
	"io"

	"github.com/dimuthu/kube-gpu/pkg/model"
)

type JSONOutput struct {
	Nodes   []model.GPUNode `json:"nodes"`
	Pending []model.GPUPod  `json:"pending"`
}

func RenderJSON(w io.Writer, nodes []model.GPUNode, pending []model.GPUPod) error {
	out := JSONOutput{
		Nodes:   nodes,
		Pending: pending,
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}
