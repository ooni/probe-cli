// Package enginex contains ooni/probe-engine extensions.
package enginex

import (
	"encoding/json"

	"github.com/fatih/color"
	"github.com/ooni/probe-engine/log"
	"github.com/ooni/probe-engine/model"
)

// LoggerAdapter exposes an interface compatible with the logger expected
// by ooni/probe-engine, and forwards real log messages to a secondary logger
// having the same interface (compatible with apex/log).
type LoggerAdapter struct {
	// Logger is the underlying logger
	log.Logger
}

// Debug emits a debug message.
func (la LoggerAdapter) Debug(msg string) {
	la.Logger.Debug(color.WhiteString("engine") + ": " + msg)
}

// Debugf formats and emits a debug message.
func (la LoggerAdapter) Debugf(format string, v ...interface{}) {
	la.Logger.Debugf(color.WhiteString("engine")+": "+format, v...)
}

// Info emits an informational message.
func (la LoggerAdapter) Info(msg string) {
	la.Logger.Info(color.BlueString("engine") + ": " + msg)
}

// Infof format and emits an informational message.
func (la LoggerAdapter) Infof(format string, v ...interface{}) {
	la.Logger.Infof(color.BlueString("engine")+": "+format, v...)
}

// Warn emits a warning message.
func (la LoggerAdapter) Warn(msg string) {
	la.Logger.Warn(color.RedString("engine") + ": " + msg)
}

// Warnf formats and emits a warning message.
func (la LoggerAdapter) Warnf(format string, v ...interface{}) {
	la.Logger.Warnf(color.RedString("engine")+": "+format, v...)
}

// MakeGenericTestKeys casts the m.TestKeys to a map[string]interface{}.
//
// Ideally, all tests should have a clear Go structure, well defined, that
// will be stored in m.TestKeys as an interface. This is not already the
// case and it's just valid for tests written in Go. Until all tests will
// be written in Go, we'll keep this glue here to make sure we convert from
// the engine format to the cli format.
//
// This function will first attempt to cast directly to map[string]interface{},
// which is possible for MK tests, and then use JSON serialization and
// de-serialization only if that's required.
func MakeGenericTestKeys(m model.Measurement) (map[string]interface{}, error) {
	if result, ok := m.TestKeys.(map[string]interface{}); ok {
		return result, nil
	}
	data, err := json.Marshal(m.TestKeys)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	return result, err
}
