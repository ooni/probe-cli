package model

//
// Richer input
//

import (
	"context"
	"encoding/json"
)

// RicherInputSession is the richer inputs's view of a session.
type RicherInputSession interface {
	// A RicherInputSession is also an [ExperimentSession].
	ExperimentSession

	// Platform returns the platform (e.g., "linux").
	Platform() string

	// ProbeASNString returns the probe ASN as a string.
	ProbeASNString() string

	// ProbeIP returns the probe IP addr.
	ProbeIP() string

	// ProbeNetworkName returns the probe ASN network name.
	ProbeNetworkName() string

	// ResolverASNString returns the resolver ASN as a string.
	ResolverASNString() string

	// ResolverNetworkName returns the resoler ASN network name.
	ResolverNetworkName() string

	// SoftwareName returns the software name.
	SoftwareName() string

	// SoftwareVersion returns the software version.
	SoftwareVersion() string
}

// RicherInput contains richer input.
type RicherInput struct {
	// Annotations contains the annotations.
	Annotations map[string]string

	// Input is the input to use.
	Input string

	// Options contains unparsed options.
	Options json.RawMessage
}

// RicherInputExperiment is an experiment using richer input.
type RicherInputExperiment interface {
	// KibiBytesReceived returns the KiB received by the experiment.
	KibiBytesReceived() float64

	// KibiBytesSent returns the KiB send by the experiment.
	KibiBytesSent() float64

	// Measure performs a measurement using richer input.
	Measure(ctx context.Context, input RicherInput) (*Measurement, error)

	// NewReportTemplate creates a new report template suitable
	// for opening a report for this experiment.
	NewReportTemplate() *OOAPIReportTemplate
}
