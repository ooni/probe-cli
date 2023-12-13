package dsljson

import (
	"encoding/json"

	"github.com/ooni/probe-cli/v3/internal/x/dslvm"
)

type tcpConnectValue struct {
	Input  string   `json:"input"`
	Output string   `json:"output"`
	Tags   []string `json:"tags"`
}

func (lx *loader) onTCPConnect(raw json.RawMessage) error {
	// parse the raw value
	var value tcpConnectValue
	if err := json.Unmarshal(raw, &value); err != nil {
		return err
	}

	// create the required output registers
	output, err := registerMakeOutput[*dslvm.TCPConnection](lx, value.Output)
	if err != nil {
		return err
	}

	// fetch the required input register
	input, err := registerPopInput[string](lx, value.Input)
	if err != nil {
		return err
	}

	// instantiate the stage
	sx := &dslvm.TCPConnectStage{
		Input:  input,
		Output: output,
		Tags:   value.Tags,
	}

	// remember the stage for later
	lx.stages = append(lx.stages, sx)
	return nil
}
