package engine

import (
	"os"

	"github.com/ooni/probe-cli/v3/internal/model"
)

func (e *Experiment) SaveMeasurementEx(
	measurement *model.Measurement, filePath string,
	marshal func(v interface{}) ([]byte, error),
	openFile func(name string, flag int, perm os.FileMode) (*os.File, error),
	write func(fp *os.File, b []byte) (n int, err error),
) error {
	return e.saveMeasurement(measurement, filePath, marshal, openFile, write)
}
