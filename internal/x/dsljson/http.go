package dsljson

import (
	"encoding/json"
	"errors"

	"github.com/ooni/probe-cli/v3/internal/x/dslvm"
)

type httpRoundTripValue struct {
	Accept              string `json:"accept"`
	AcceptLanguage      string `json:"accept_language"`
	Host                string `json:"host"`
	Input               string `json:"input"`
	MaxBodySnapshotSize int64  `json:"max_body_snapshot_size"`
	Method              string `json:"method"`
	Output              string `json:"output"`
	Referer             string `json:"referer"`
	URLPath             string `json:"url_path"`
	UserAgent           string `json:"user_agent"`
}

func (lx *loader) onHTTPRoundTrip(raw json.RawMessage) error {
	// parse the raw value
	var value httpRoundTripValue
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
		sx = &dslvm.HTTPRoundTripStage[*dslvm.TCPConnection]{
			Accept:              value.Accept,
			AcceptLanguage:      value.AcceptLanguage,
			Host:                value.Host,
			Input:               input,
			MaxBodySnapshotSize: value.MaxBodySnapshotSize,
			Method:              value.Method,
			Output:              output,
			Referer:             value.Referer,
			URLPath:             value.URLPath,
			UserAgent:           value.UserAgent,
		}

	case chan *dslvm.TLSConnection:
		sx = &dslvm.HTTPRoundTripStage[*dslvm.TLSConnection]{
			Accept:              value.Accept,
			AcceptLanguage:      value.AcceptLanguage,
			Host:                value.Host,
			Input:               input,
			MaxBodySnapshotSize: value.MaxBodySnapshotSize,
			Method:              value.Method,
			Output:              output,
			Referer:             value.Referer,
			URLPath:             value.URLPath,
			UserAgent:           value.UserAgent,
		}

	case chan *dslvm.QUICConnection:
		sx = &dslvm.HTTPRoundTripStage[*dslvm.QUICConnection]{
			Accept:              value.Accept,
			AcceptLanguage:      value.AcceptLanguage,
			Host:                value.Host,
			Input:               input,
			MaxBodySnapshotSize: value.MaxBodySnapshotSize,
			Method:              value.Method,
			Output:              output,
			Referer:             value.Referer,
			URLPath:             value.URLPath,
			UserAgent:           value.UserAgent,
		}

	default:
		return errors.New("http_round_trip: cannot instantiate output stage")
	}

	// remember the stage for later
	lx.stages = append(lx.stages, sx)
	return nil
}
