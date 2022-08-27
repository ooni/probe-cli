package engine

import (
	"context"
	"errors"
	"sync"

	"github.com/ooni/probe-cli/v3/internal/atomicx"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// AsyncSaver is a an asynchronous [Saver].
type AsyncSaver interface {
	// SaveMeasurement requests the saver to save a measurement. The return
	// value indicates whether the measurement has been accepted by the
	// saver or whether it has been rejected. The saver will reject
	// measurements after you've called its Stop method.
	SaveMeasurement(idx int, m *model.Measurement) bool

	// Stop tells the saver that it should stop running as soon
	// as possible, which entails trying to finish saving all the queued
	// measurements to avoid losing them. Use Wait to know when the
	// saver has finished saver all measurements.
	Stop()

	// Wait waits for the saver to finish saving all the
	// measurements that are currently queued for writing. Note
	// that you should call Stop before calling Wait to inform
	// the saver that it should stop running ASAP.
	Wait()
}

// Saver saves a measurement on some persistent storage.
type Saver interface {
	SaveMeasurement(idx int, m *model.Measurement) error
}

// asyncSaver implements AsyncSaver.
type asyncSaver struct {
	// logger is the Logger to use.
	logger model.Logger

	// queue is the queue containing measurements to submit.
	queue chan *measurementWithIndex

	// running tracks the running goroutines.
	running *sync.WaitGroup

	// stopped indicates that the submitter has stopped.
	stopped *atomicx.Int64
}

// StartAsyncSaver creates a new [AsyncSaver] with the given [Saver]. We'll
// run the [AsyncSaver] in a background goroutine. You should call Stop to tell
// it it's time to stop and Wait to wait for it to complete.
func StartAsyncSaver(saver Saver) AsyncSaver {
	as := &asyncSaver{
		logger:  nil,
		queue:   make(chan *measurementWithIndex),
		running: &sync.WaitGroup{},
		stopped: &atomicx.Int64{},
	}
	go as.run(context.Background(), saver)
	as.running.Add(1)
	return as
}

// SaveMeasurement implements AsyncSaver.SaveMeasurement.
func (as *asyncSaver) SaveMeasurement(idx int, m *model.Measurement) bool {
	if as.stopped.Load() > 0 {
		return false
	}
	as.queue <- &measurementWithIndex{
		idx: idx,
		m:   m,
	}
	return true
}

// run saves measurements in FIFO order.
func (as *asyncSaver) run(ctx context.Context, saver Saver) {
	defer as.running.Done()
	for m := range as.queue {
		// TODO(bassosimone): should we tell anyone about this error?
		err := saver.SaveMeasurement(m.idx, m.m)
		if err != nil {
			as.logger.Warnf("asyncSaver: cannot save measurement: %s", err.Error())
			continue
		}
	}
}

// Stop implements AsyncSaver.Stop.
func (as *asyncSaver) Stop() {
	as.stopped.Add(1) // must happen BEFORE closing the channel
	close(as.queue)
}

// Wait implements AsyncSaver.Wait.
func (as *asyncSaver) Wait() {
	as.running.Wait()
}

// SaverConfig is the configuration for creating a new Saver.
type SaverConfig struct {
	// Enabled is true if saving is enabled.
	Enabled bool

	// Experiment is the experiment we're currently running.
	Experiment SaverExperiment

	// FilePath is the filepath where to append the measurement as a
	// serialized JSON followed by a newline character.
	FilePath string

	// Logger is the logger used by the saver.
	Logger model.Logger
}

// SaverExperiment is an experiment according to the Saver.
type SaverExperiment interface {
	SaveMeasurement(m *model.Measurement, filepath string) error
}

// NewSaver creates a new instance of Saver.
func NewSaver(config SaverConfig) (Saver, error) {
	if !config.Enabled {
		return fakeSaver{}, nil
	}
	if config.FilePath == "" {
		return nil, errors.New("saver: passed an empty filepath")
	}
	return realSaver{
		Experiment: config.Experiment,
		FilePath:   config.FilePath,
		Logger:     config.Logger,
	}, nil
}

type fakeSaver struct{}

func (fs fakeSaver) SaveMeasurement(idx int, m *model.Measurement) error {
	return nil
}

var _ Saver = fakeSaver{}

type realSaver struct {
	Experiment SaverExperiment
	FilePath   string
	Logger     model.Logger
}

func (rs realSaver) SaveMeasurement(idx int, m *model.Measurement) error {
	rs.Logger.Info("saving measurement to disk")
	return rs.Experiment.SaveMeasurement(m, rs.FilePath)
}

var _ Saver = realSaver{}
