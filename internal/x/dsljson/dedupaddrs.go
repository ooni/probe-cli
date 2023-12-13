package dsljson

import (
	"encoding/json"

	"github.com/ooni/probe-cli/v3/internal/x/dslvm"
)

type dedupAddrsValue struct {
	Inputs []string `json:"inputs"`
	Output string   `json:"output"`
}

func (lx *loader) onDedupAddrs(raw json.RawMessage) error {
	// parse the raw value
	var value dedupAddrsValue
	if err := json.Unmarshal(raw, &value); err != nil {
		return err
	}

	// create the required output registers
	output, err := registerMakeOutput[string](lx, value.Output)
	if err != nil {
		return err
	}

	// instantiate the stage
	sx := &dslvm.DedupAddrsStage{
		Inputs: []<-chan string{},
		Output: output,
	}
	for _, name := range value.Inputs {
		input, err := registerPopInput[string](lx, name)
		if err != nil {
			return err
		}
		sx.Inputs = append(sx.Inputs, input)
	}

	// remember the stage for later
	lx.stages = append(lx.stages, sx)
	return nil
}
