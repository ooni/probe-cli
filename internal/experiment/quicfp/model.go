package quicfp

// This file contains minimal local stand-ins for types that, inside the real
// ooni/probe-cli tree, live in internal/model and internal/legacy/legacymodel.
// They are reproduced here (with the same field names/JSON tags) purely so
// that this package builds and can be unit-tested on its own. When vendoring
// this experiment into probe-cli proper, delete this file and switch the
// imports in the other files to:
//
//   "github.com/ooni/probe-cli/v3/internal/legacy/legacymodel"
//   "github.com/ooni/probe-cli/v3/internal/legacy/tracex"
//   "github.com/ooni/probe-cli/v3/internal/model"
//
// exactly like internal/experiment/quicping does.

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net"
	"time"
)

// UDPLikeConn is the subset of net.PacketConn that the experiment needs.
// It mirrors model.UDPLikeConn so a mock can be substituted in tests.
type UDPLikeConn interface {
	WriteTo(p []byte, addr net.Addr) (int, error)
	ReadFrom(p []byte) (int, net.Addr, error)
	SetDeadline(t time.Time) error
	Close() error
}

// Measurement is a minimal stand-in for model.Measurement: just enough
// fields for this experiment to timestamp events relative to a start time
// and to carry the arbitrary TestKeys payload and string Input.
type Measurement struct {
	Input                     string
	MeasurementStartTimeSaved time.Time
	TestKeys                  interface{}
}

// ExperimentArgs is a stand-in for model.ExperimentArgs.
type ExperimentArgs struct {
	Callbacks   interface{}
	Measurement *Measurement
	Session     ExperimentSession
}

// ExperimentSession is a stand-in for model.ExperimentSession: just logging.
type ExperimentSession interface {
	Logger() Logger
}

// Logger is a stand-in for model.Logger.
type Logger interface {
	Infof(format string, v ...interface{})
	Warnf(format string, v ...interface{})
}

// ExperimentMeasurer is a stand-in for model.ExperimentMeasurer.
type ExperimentMeasurer interface {
	ExperimentName() string
	ExperimentVersion() string
	Run(ctx context.Context, args *ExperimentArgs) error
}

// ArchivalMaybeBinaryData is a stand-in for legacymodel.ArchivalMaybeBinaryData:
// it serializes as a plain JSON string when the underlying data is valid UTF-8,
// and as {"format":"base64","data":"..."} otherwise -- matching OONI's archival
// format for possibly-binary payloads (e.g. raw QUIC/TLS bytes on the wire).
type ArchivalMaybeBinaryData struct {
	Value string
}

// MarshalJSON implements json.Marshaler.
func (d ArchivalMaybeBinaryData) MarshalJSON() ([]byte, error) {
	if isValidUTF8(d.Value) {
		return json.Marshal(d.Value)
	}
	type binaryData struct {
		Format string `json:"format"`
		Data   string `json:"data"`
	}
	return json.Marshal(binaryData{
		Format: "base64",
		Data:   base64.StdEncoding.EncodeToString([]byte(d.Value)),
	})
}

func isValidUTF8(s string) bool {
	for _, r := range s {
		if r == '\uFFFD' {
			return false
		}
	}
	return true
}

// NewFailure is a stand-in for tracex.NewFailure: converts an error into the
// OONI "failure string" convention (nil on success, else err.Error()).
func NewFailure(err error) *string {
	if err == nil {
		return nil
	}
	s := err.Error()
	return &s
}
