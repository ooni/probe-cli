package dialer

import (
	"context"
	"net"
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/engine/netx/trace"
	"github.com/ooni/probe-cli/v3/internal/errorsx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// Dialer establishes network connections.
type Dialer interface {
	// DialContext behaves like net.Dialer.DialContext.
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// Resolver is the interface we expect from a DNS resolver.
type Resolver interface {
	// LookupHost behaves like net.Resolver.LookupHost.
	LookupHost(ctx context.Context, hostname string) (addrs []string, err error)
}

// Logger is the interface we expect from a logger.
type Logger interface {
	// Debugf formats and emits a debug message.
	Debugf(format string, v ...interface{})

	// Debug emits a debug message.
	Debug(msg string)
}

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
	// persistent connections, because they strick to the counters set when the
	// connection was established. This typically means we miss the bytes sent and
	// received when submitting a measurement. Such bytes are specifically not
	// seen by the experiment specific byte counter.
	//
	// For this reason, this implementation may be heavily changed/removed.
	ContextByteCounting bool

	// DialSaver is the optional saver for dialing events. If not
	// set, we will not save any dialing event.
	DialSaver *trace.Saver

	// Logger is the optional logger. If not set, there
	// will be no logging from the new dialer.
	Logger Logger

	// ProxyURL is the optional proxy URL.
	ProxyURL *url.URL

	// ReadWriteSaver is like DialSaver but for I/O events.
	ReadWriteSaver *trace.Saver
}

// New creates a new Dialer from the specified config and resolver.
func New(config *Config, resolver Resolver) Dialer {
	var d Dialer = netxlite.DefaultDialer
	d = &errorsx.ErrorWrapperDialer{Dialer: d}
	if config.Logger != nil {
		d = &netxlite.DialerLogger{Dialer: d, Logger: config.Logger}
	}
	if config.DialSaver != nil {
		d = &saverDialer{Dialer: d, Saver: config.DialSaver}
	}
	if config.ReadWriteSaver != nil {
		d = &saverConnDialer{Dialer: d, Saver: config.ReadWriteSaver}
	}
	d = &netxlite.DialerResolver{Resolver: resolver, Dialer: d}
	d = &proxyDialer{ProxyURL: config.ProxyURL, Dialer: d}
	if config.ContextByteCounting {
		d = &byteCounterDialer{Dialer: d}
	}
	d = &shapingDialer{Dialer: d}
	return d
}
