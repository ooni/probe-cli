package dsljson

import (
	"encoding/json"

	"github.com/ooni/probe-cli/v3/internal/x/dslvm"
)

type teeAddrsValue struct {
	Input   string   `json:"input"`
	Outputs []string `json:"outputs"`
}

func (lx *loader) onTeeAddrs(raw json.RawMessage) error {
	// parse the raw value
	var value teeAddrsValue
	if err := json.Unmarshal(raw, &value); err != nil {
		return err
	}

	// create the required input registers
	input, err := registerPopInput[string](lx, value.Input)
	if err != nil {
		return err
	}

	// instantiate the stage
	sx := &dslvm.TeeAddrsStage{
		Input:   input,
		Outputs: []chan<- string{},
	}
	for _, name := range value.Outputs {
		input, err := registerMakeOutput[string](lx, name)
		if err != nil {
			return err
		}
		sx.Outputs = append(sx.Outputs, input)
	}

	// remember the stage for later
	lx.stages = append(lx.stages, sx)
	return nil
}
