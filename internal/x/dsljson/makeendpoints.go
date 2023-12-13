package dsljson

import (
	"encoding/json"

	"github.com/ooni/probe-cli/v3/internal/x/dslvm"
)

type makeEndpointsValue struct {
	Input  string `json:"input"`
	Output string `json:"output"`
	Port   string `json:"port"`
}

func (lx *loader) onMakeEndpoints(raw json.RawMessage) error {
	// parse the raw value
	var value makeEndpointsValue
	if err := json.Unmarshal(raw, &value); err != nil {
		return err
	}

	// create the required output registers
	output, err := registerMakeOutput[string](lx, value.Output)
	if err != nil {
		return err
	}

	// fetch the required input register
	input, err := registerPopInput[string](lx, value.Input)
	if err != nil {
		return err
	}

	// instantiate the ASM stage
	sx := &dslvm.MakeEndpointsStage{
		Input:  input,
		Output: output,
		Port:   value.Port,
	}

	// remember the stage for later
	lx.stages = append(lx.stages, sx)
	return nil
}
