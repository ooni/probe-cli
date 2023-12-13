package dsljson

import (
	"encoding/json"
	"errors"

	"github.com/ooni/probe-cli/v3/internal/x/dslvm"
)

type dropValue struct {
	Input  string `json:"input"`
	Output string `json:"output"`
}

func (lx *loader) onDrop(raw json.RawMessage) error {
	// parse the raw value
	var value dropValue
	if err := json.Unmarshal(raw, &value); err != nil {
		return err
	}

	// create the required output registers
	output, err := registerMakeOutput[dslvm.Done](lx, value.Output)
	if err != nil {
		return err
	}

	// make sure we register output as something to wait for
	lx.toWait = append(lx.toWait, output)

	// fetch the required input register as a generic any value
	xinput, err := registerPopInputRaw(lx, value.Input)
	if err != nil {
		return err
	}

	// figure out the correct xinput type
	var sx dslvm.Stage
	switch input := xinput.(type) {
	case chan *dslvm.TCPConnection:
		sx = &dslvm.DropStage[*dslvm.TCPConnection]{
			Input:  input,
			Output: output,
		}

	case chan *dslvm.TLSConnection:
		sx = &dslvm.DropStage[*dslvm.TLSConnection]{
			Input:  input,
			Output: output,
		}

	case chan *dslvm.QUICConnection:
		sx = &dslvm.DropStage[*dslvm.QUICConnection]{
			Input:  input,
			Output: output,
		}

	case chan string:
		sx = &dslvm.DropStage[string]{
			Input:  input,
			Output: output,
		}

	default:
		return errors.New("drop: cannot instantiate output stage")
	}

	// remember the stage for later
	lx.stages = append(lx.stages, sx)
	return nil
}
