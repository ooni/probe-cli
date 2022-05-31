// Package dialer allows you to create a net.Dialer-compatible
// DialContext-enabled dialer with error wrapping, optional logging,
// optional network-events saving, and optional proxying.
package dialer

import (
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/bytecounter"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/tracex"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// Config contains the settings for New.
type Config struct {
	// ContextByteCounting optionally configures context-based
	// byte counting. By default we don't do that.
	//
	// Use WithExperimentByteCounter and WithSessionByteCounter
	// to assign byte counters to a context. The code will use
	// corresponding, private functions to access the configured
	// byte counters and will notify them about I/O events.
	//
	// Bug
	//
	// This implementation cannot properly account for the bytes that are sent by
	// persistent connections, because they stick to the counters set when the
	// connection was established. This typically means we miss the bytes sent and
	// received when submitting a measurement. Such bytes are specifically not
	// seen by the experiment specific byte counter.
	//
	// For this reason, this implementation may be heavily changed/removed.
	ContextByteCounting bool

	// DialSaver is the optional saver for dialing events. If not
	// set, we will not save any dialing event.
	DialSaver *tracex.Saver

	// Logger is the optional logger. If not set, there
	// will be no logging from the new dialer.
	Logger model.DebugLogger

	// ProxyURL is the optional proxy URL.
	ProxyURL *url.URL

	// ReadWriteSaver is like DialSaver but for I/O events.
	ReadWriteSaver *tracex.Saver
}

// New creates a new Dialer from the specified config and resolver.
func New(config *Config, resolver model.Resolver) model.Dialer {
	var logger model.DebugLogger = model.DiscardLogger
	if config.Logger != nil {
		logger = config.Logger
	}
	modifiers := []netxlite.DialerWrapper{
		func(dialer model.Dialer) model.Dialer {
			if config.DialSaver != nil {
				dialer = &tracex.SaverDialer{Dialer: dialer, Saver: config.DialSaver}
			}
			return dialer
		},
		func(dialer model.Dialer) model.Dialer {
			if config.ReadWriteSaver != nil {
				dialer = &tracex.SaverConnDialer{Dialer: dialer, Saver: config.ReadWriteSaver}
			}
			return dialer
		},
	}
	d := netxlite.NewDialerWithResolver(logger, resolver, modifiers...)
	d = &netxlite.MaybeProxyDialer{ProxyURL: config.ProxyURL, Dialer: d}
	if config.ContextByteCounting {
		d = &bytecounter.ContextAwareDialer{Dialer: d}
	}
	return d
}
