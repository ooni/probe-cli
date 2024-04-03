package engine

import (
	"encoding/json"
	"errors"
	"os"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// Saver is an alias for model.Saver.
type Saver = model.Saver

// SaverConfig is the configuration for creating a new Saver.
type SaverConfig struct {
	// Enabled is true if saving is enabled.
	Enabled bool

	// FilePath is the filepath where to append the measurement as a
	// serialized JSON followed by a newline character.
	FilePath string

	// Logger is the logger used by the saver.
	Logger model.Logger
}

// NewSaver creates a new instance of Saver.
func NewSaver(config SaverConfig) (Saver, error) {
	if !config.Enabled {
		return fakeSaver{}, nil
	}
	if config.FilePath == "" {
		return nil, errors.New("saver: passed an empty filepath")
	}
	return &realSaver{
		FilePath: config.FilePath,
		Logger:   config.Logger,
		savefunc: SaveMeasurement,
	}, nil
}

type fakeSaver struct{}

func (fs fakeSaver) SaveMeasurement(m *model.Measurement) error {
	return nil
}

var _ Saver = fakeSaver{}

type realSaver struct {
	FilePath string
	Logger   model.Logger
	savefunc func(measurement *model.Measurement, filePath string) error
}

func (rs *realSaver) SaveMeasurement(m *model.Measurement) error {
	rs.Logger.Info("saving measurement to disk")
	return rs.savefunc(m, rs.FilePath)
}

var _ Saver = &realSaver{}

// SaveMeasurement saves a measurement on the specified file path.
func SaveMeasurement(measurement *model.Measurement, filePath string) error {
	return saveMeasurement(
		measurement, filePath, json.Marshal, os.OpenFile,
		func(fp *os.File, b []byte) (int, error) {
			return fp.Write(b)
		},
	)
}

func saveMeasurement(
	measurement *model.Measurement, filePath string,
	marshal func(v interface{}) ([]byte, error),
	openFile func(name string, flag int, perm os.FileMode) (*os.File, error),
	write func(fp *os.File, b []byte) (n int, err error),
) error {
	data, err := marshal(measurement)
	if err != nil {
		return err
	}
	data = append(data, byte('\n'))
	filep, err := openFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	if _, err := write(filep, data); err != nil {
		return err
	}
	return filep.Close()
}
