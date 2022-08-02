package engine

import (
	"context"
	"sync"

	"github.com/ooni/probe-cli/v3/internal/atomicx"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// TODO(bassosimone): maybe keep track of which measurements
// could not be submitted by a specific submitter?

// AsyncSubmitter is a an async submitter. It runs one or more [Submitter]
// in the background and will use them for submitting measurements.
type AsyncSubmitter interface {
	// Submit requests the submitter to submit a measurement. The return
	// value indicates whether the measurement has been accepted by the
	// submitter or whether it has been rejected. The submitter will reject
	// measurements after you've called its Stop method.
	Submit(idx int, m *model.Measurement) bool

	// Stop tells the submitter that it should stop running as soon
	// as possible, which entails trying to submit all the queued
	// measurements to avoid losing them. Use Wait to know when the
	// submitter has finished submitting all measurements.
	Stop()

	// Wait waits for the submitter to finish submitting all the
	// measurements that are currently queued for submission. Note
	// that you should call Stop before calling Wait to inform
	// the submitter that it should stop running ASAP.
	Wait()
}

// measurementWithIndex is a measurement along with its index.
type measurementWithIndex struct {
	// idx is the index
	idx int

	// m is the measurement
	m *model.Measurement
}

// asyncSubmitterBuffer is the buffer used by the async submitter's queue.
const asyncSubmitterBuffer = 4

// asyncSubmitter implements AsyncSubmitter.
type asyncSubmitter struct {
	// asyncSaver is the AsyncSaver instance to use.
	asyncSaver AsyncSaver

	// logger is the Logger to use.
	logger model.Logger

	// queue is the queue containing measurements to submit.
	queue chan *measurementWithIndex

	// running tracks the running goroutines.
	running *sync.WaitGroup

	// stopped indicates that the submitter has stopped.
	stopped *atomicx.Int64
}

// StartAsyncSubmitter creates a new AsyncSubmitter using the
// given underlying [config]. We'll create a queue with a
// maximum buffer. When the queue is full, Submit blocks
// waiting for pending submissions to complete. This factory
// will create a single background goroutine that submits the
// measurements. You must use Stop to kill such a goroutine
// and Wait to wait for the goroutine to join. The [logger] argument
// contains the logger to be used. The [asyncSaver] argument is
// the saver that should write on disk measurements
// that have been submitted, with their correct report ID.
func StartAsyncSubmitter(
	logger model.Logger, submitter Submitter, asyncSaver AsyncSaver) AsyncSubmitter {
	asub := &asyncSubmitter{
		asyncSaver: asyncSaver,
		logger:     logger,
		queue:      make(chan *measurementWithIndex, asyncSubmitterBuffer),
		running:    &sync.WaitGroup{},
		stopped:    &atomicx.Int64{},
	}
	go asub.run(context.Background(), submitter)
	asub.running.Add(1)
	return asub
}

// Submit implements AsyncSubmitter.Submit.
func (asub *asyncSubmitter) Submit(idx int, m *model.Measurement) bool {
	if asub.stopped.Load() > 0 {
		return false
	}
	asub.queue <- &measurementWithIndex{
		idx: idx,
		m:   m,
	}
	return true
}

// run submits measurements in FIFO order.
func (asub *asyncSubmitter) run(ctx context.Context, submitter Submitter) {
	defer asub.running.Done()
	for m := range asub.queue {
		// TODO(bassosimone): add support for knowing when we could not
		// submit measurements. We will discuss this once we've tried
		// out this simple concept in a real-world experiment.
		//
		// Likewise, we should discuss policies regarding retries, which
		// we're not implementing at the moment for brevity.
		err := submitter.Submit(ctx, m.idx, m.m)
		if err != nil {
			asub.logger.Warnf("asyncSubmitter: cannot submit measurement: %s", err.Error())
			// FALLTHRU
		}
		// We chain saving after the submission such that the reportID, which is
		// modified by Submit (a choice that I regret of), get finally saved.
		_ = asub.asyncSaver.SaveMeasurement(m.idx, m.m)
	}
}

// Stop implements AsyncSubmitter.Stop.
func (asub *asyncSubmitter) Stop() {
	asub.stopped.Add(1) // must happen BEFORE closing the channel
	close(asub.queue)
}

// Wait implements AsyncSubmitter.Wait.
func (asub *asyncSubmitter) Wait() {
	asub.running.Wait()
}

// Submitter submits a measurement to the OONI collector.
type Submitter interface {
	// Submit submits the measurement and updates its
	// report ID field in case of success.
	Submit(ctx context.Context, idx int, m *model.Measurement) error
}

// SubmitterSession is the Submitter's view of the Session.
type SubmitterSession interface {
	// NewSubmitter creates a new probeservices Submitter.
	NewSubmitter(ctx context.Context) (Submitter, error)
}

// SubmitterConfig contains settings for NewSubmitter.
type SubmitterConfig struct {
	// Enabled is true if measurement submission is enabled.
	Enabled bool

	// Session is the current session.
	Session SubmitterSession

	// Logger is the logger to be used.
	Logger model.Logger
}

// NewSubmitter creates a new submitter instance. Depending on
// whether submission is enabled or not, the returned submitter
// instance migh just be a stub implementation.
func NewSubmitter(ctx context.Context, config SubmitterConfig) (Submitter, error) {
	if !config.Enabled {
		return stubSubmitter{}, nil
	}
	subm, err := config.Session.NewSubmitter(ctx)
	if err != nil {
		return nil, err
	}
	return realSubmitter{subm: subm, logger: config.Logger}, nil
}

type stubSubmitter struct{}

func (stubSubmitter) Submit(ctx context.Context, idx int, m *model.Measurement) error {
	return nil
}

var _ Submitter = stubSubmitter{}

type realSubmitter struct {
	subm   Submitter
	logger model.Logger
}

func (rs realSubmitter) Submit(ctx context.Context, idx int, m *model.Measurement) error {
	rs.logger.Info("submitting measurement to OONI collector; please be patient...")
	return rs.subm.Submit(ctx, idx, m)
}
