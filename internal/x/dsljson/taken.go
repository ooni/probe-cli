package dsljson

import (
	"encoding/json"
	"errors"

	"github.com/ooni/probe-cli/v3/internal/x/dslvm"
)

type takeNValue struct {
	Input  string `json:"input"`
	N      int64  `json:"n"`
	Output string `json:"output"`
}

func (lx *loader) onTakeN(raw json.RawMessage) error {
	// parse the raw value
	var value takeNValue
	if err := json.Unmarshal(raw, &value); err != nil {
		return err
	}

	// fetch the required input register as a generic any value
	xinput, err := registerPopInputRaw(lx, value.Input)
	if err != nil {
		return err
	}

	// figure out the correct xinput type
	var sx dslvm.Stage
	switch input := xinput.(type) {
	case chan *dslvm.TCPConnection:
		output, err := registerMakeOutput[*dslvm.TCPConnection](lx, value.Output)
		if err != nil {
			return err
		}
		sx = &dslvm.TakeNStage[*dslvm.TCPConnection]{
			Input:  input,
			N:      value.N,
			Output: output,
		}

	case chan *dslvm.TLSConnection:
		output, err := registerMakeOutput[*dslvm.TLSConnection](lx, value.Output)
		if err != nil {
			return err
		}
		sx = &dslvm.TakeNStage[*dslvm.TLSConnection]{
			Input:  input,
			N:      value.N,
			Output: output,
		}

	case chan *dslvm.QUICConnection:
		output, err := registerMakeOutput[*dslvm.QUICConnection](lx, value.Output)
		if err != nil {
			return err
		}
		sx = &dslvm.TakeNStage[*dslvm.QUICConnection]{
			Input:  input,
			N:      value.N,
			Output: output,
		}

	case chan string:
		output, err := registerMakeOutput[string](lx, value.Output)
		if err != nil {
			return err
		}
		sx = &dslvm.TakeNStage[string]{
			Input:  input,
			N:      value.N,
			Output: output,
		}

	default:
		return errors.New("take_n: cannot instantiate output stage")
	}

	// remember the stage for later
	lx.stages = append(lx.stages, sx)
	return nil
}
