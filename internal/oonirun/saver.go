package oonirun

import (
	"errors"

	"github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/internal/model"
)

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
func NewSaver(config SaverConfig) (model.Saver, error) {
	if !config.Enabled {
		return fakeSaver{}, nil
	}
	if config.FilePath == "" {
		return nil, errors.New("saver: passed an empty filepath")
	}
	return &realSaver{
		FilePath: config.FilePath,
		Logger:   config.Logger,
		savefunc: engine.SaveMeasurement,
	}, nil
}

type fakeSaver struct{}

func (fs fakeSaver) SaveMeasurement(m *model.Measurement) error {
	return nil
}

var _ model.Saver = fakeSaver{}

type realSaver struct {
	FilePath string
	Logger   model.Logger
	savefunc func(measurement *model.Measurement, filePath string) error
}

func (rs *realSaver) SaveMeasurement(m *model.Measurement) error {
	rs.Logger.Info("saving measurement to disk")
	return rs.savefunc(m, rs.FilePath)
}

var _ model.Saver = &realSaver{}
