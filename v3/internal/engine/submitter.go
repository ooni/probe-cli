package engine

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/engine/model"
)

// TODO(bassosimone): maybe keep track of which measurements
// could not be submitted by a specific submitter?

// Submitter submits a measurement to the OONI collector.
type Submitter interface {
	// Submit submits the measurement and updates its
	// report ID field in case of success.
	Submit(ctx context.Context, m *model.Measurement) error
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

func (stubSubmitter) Submit(ctx context.Context, m *model.Measurement) error {
	return nil
}

var _ Submitter = stubSubmitter{}

type realSubmitter struct {
	subm   Submitter
	logger model.Logger
}

func (rs realSubmitter) Submit(ctx context.Context, m *model.Measurement) error {
	rs.logger.Info("submitting measurement to OONI collector; please be patient...")
	return rs.subm.Submit(ctx, m)
}
