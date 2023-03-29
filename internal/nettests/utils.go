package nettests

import (
	"encoding/json"
	"os"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// SaveMeasurement saves the given measurement in the given directory.
func SaveMeasurement(measurement *model.Measurement, filename string) error {
	data, err := json.Marshal(measurement)
	runtimex.PanicOnError(err, "json.Marshal failed")
	data = append(data, byte('\n'))
	return os.WriteFile(filename, data, 0600)
}
