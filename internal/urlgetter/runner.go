package urlgetter

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// RunnerTestKeys is the [*Runner] view of the [*TestKeys].
type RunnerTestKeys interface {
	// AppendNetworkEvents appends network events to the test keys.
	AppendNetworkEvents(values ...*model.ArchivalNetworkEvent)

	// AppendQueries appends DNS lookup results to the test keys.
	AppendQueries(values ...*model.ArchivalDNSLookupResult)

	// AppendTCPConnect appends TCP connect results to the test keys.
	AppendTCPConnect(values ...*model.ArchivalTCPConnectResult)

	// AppendTLSHandshakes appends TLS handshakes results to the test keys.
	AppendTLSHandshakes(values ...*model.ArchivalTLSOrQUICHandshakeResult)

	// MaybeSetFailedOperation sets the failed operation field if it's not already set.
	MaybeSetFailedOperation(operation string)

	// MaybeSetFailure sets the failure string field if it's not already set.
	MaybeSetFailure(failure string)

	// PrependRequests appends HTTP requests results to the test keys.
	PrependRequests(values ...*model.ArchivalHTTPRequestResult)
}

// RunnerTraceIndexGenerator generates trace indexes.
type RunnerTraceIndexGenerator interface {
	Next() int64
}

// RunnerSession is the measurement session as seen by a [*Runner].
type RunnerSession interface {
	// Logger returns the logger use.
	Logger() model.Logger
}

// Runner performs measurements.
//
// The zero value is invalid; init all the fields marked as MANDATORY.
type Runner struct {
	// Begin is the MANDATORY time when we started measuring.
	Begin time.Time

	// IndexGen is the MANDATORY index generator.
	IndexGen RunnerTraceIndexGenerator

	// Session is the MANDATORY session.
	Session RunnerSession

	// TestKeys contains the MANDATORY test keys.
	TestKeys RunnerTestKeys

	// UNet is the OPTIONAL underlying network.
	UNet model.UnderlyingNetwork
}

// ErrUnknownURLScheme indicates that we don't know how to handle a given target URL scheme.
var ErrUnknownURLScheme = errors.New("unknown URL scheme")

// Run measures the given [*url.URL] using the given [*Config].
func (rx *Runner) Run(ctx context.Context, config *Config, URL *url.URL) error {
	switch scheme := URL.Scheme; scheme {
	case "http", "https":
		// Implementation note: only report error for fundamental failures
		_ = rx.HTTPTransaction(ctx, config, URL)
		return nil

	case "tlshandshake":
		// Implementation note: only report error for fundamental failures
		_ = rx.TLSHandshake(ctx, config, URL)
		return nil

	case "tcpconnect":
		// Implementation note: only report error for fundamental failures
		_ = rx.TCPConnect(ctx, config, URL)
		return nil

	case "dnslookup":
		// Implementation note: only report error for fundamental failures
		_ = rx.DNSLookup(ctx, config, URL)
		return nil

	default:
		return fmt.Errorf("%w: %s", ErrUnknownURLScheme, scheme)
	}
}
