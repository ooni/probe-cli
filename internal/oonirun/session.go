package oonirun

//
// Definition of session.
//
// TODO(bassosimone): we should eventually have a common definition
// of session (which probably means a few distinct definitions?) inside
// the model package as an interface. Until we do that, which seems an
// heavy refactoring right now, this local definition will do.
//

import (
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/targetloading"
)

// Session is the definition of Session used by this package.
type Session interface {
	// A Session is also an [targetloading.Session].
	targetloading.Session

	// A Session is also a SubmitterSession.
	SubmitterSession

	// DefaultHTTPClient returns the session's default HTTPClient.
	DefaultHTTPClient() model.HTTPClient

	// Logger returns the logger used by this Session.
	Logger() model.Logger

	// NewExperimentBuilder creates a new engine.ExperimentBuilder.
	NewExperimentBuilder(name string) (model.ExperimentBuilder, error)
}
