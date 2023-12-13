package dsljson

import (
	"encoding/json"

	"github.com/ooni/probe-cli/v3/internal/x/dslvm"
)

type getaddrinfoValue struct {
	Domain string   `json:"domain"`
	Output string   `json:"output"`
	Tags   []string `json:"tags"`
}

func (lx *loader) onGetaddrinfo(raw json.RawMessage) error {
	// parse the raw value
	var value getaddrinfoValue
	if err := json.Unmarshal(raw, &value); err != nil {
		return err
	}

	// create the required output registers
	output, err := registerMakeOutput[string](lx, value.Output)
	if err != nil {
		return err
	}

	// instantiate the stage
	sx := &dslvm.GetaddrinfoStage{
		Domain: value.Domain,
		Output: output,
		Tags:   value.Tags,
	}

	// remember the stage for later
	lx.stages = append(lx.stages, sx)
	return nil
}
