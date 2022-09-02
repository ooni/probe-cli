// Package tunnel allows to create tunnels to speak
// with OONI backends and other services.
//
// You need to fill a Config object and call Start to
// obtain an instance of Tunnel. The tunnel will expose
// a SOCKS5 proxy. You need to configure your HTTP
// code to use such a proxy. Remember to call the Stop
// method of a tunnel when you are done.
//
// There are two use cases for this package. The first
// use case is to enable urlgetter to perform measurements
// over tunnels (mainly psiphon).
//
// The second use case is to use tunnels to reach to the
// OONI backend when it's blocked. For the latter case
// we currently mainly use psiphon. In such a case, we'll
// use a psiphon configuration embedded into the OONI
// binary itself. When you are running a version of OONI
// that does not embed such a configuration, it won't
// be possible to address this use case.
//
// For tor tunnels, we have two distinct configurations: on
// mobile we use github.com/ooni/go-libtor; on desktop we use
// a more complex strategy. If the OONI_TOR_BINARY environment
// variable is set, we assume its value is the path to the
// tor binary and use it. Otherwise, we search for an executable
// called "tor" in the PATH and use it. Those two strategies
// are implemented, respectively by tormobile.go and tordesktop.go.
//
// See session.go in the engine package for more details
// concerning this second use case.
package tunnel

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"
)

// Session is a measurement session. We filter for the only
// functionality we're interested to use. That is, fetching the
// psiphon configuration from the OONI backend (if possible).
//
// Depending on how OONI is compiled, the psiphon configuration
// may be embedded into the binary. In such a case, we won't
// need to download the configuration from the backend.
type Session interface {
	// FetchPsiphonConfig should fetch and return the psiphon config
	// as a serialized JSON, or fail with an error.
	FetchPsiphonConfig(ctx context.Context) ([]byte, error)
}

// Tunnel is a tunnel for communicating with OONI backends
// (and other services) to circumvent blocking.
type Tunnel interface {
	// BootstrapTime returns the time it required to
	// create a new tunnel instance.
	BootstrapTime() time.Duration

	// SOCKS5ProxyURL returns the SOCSK5 proxy URL.
	SOCKS5ProxyURL() *url.URL

	// Stop stops the tunnel. You should not attempt to
	// use any other tunnel method after Stop.
	Stop()
}

// ErrEmptyTunnelDir indicates that config.TunnelDir is empty.
var ErrEmptyTunnelDir = errors.New("TunnelDir is empty")

// ErrUnsupportedTunnelName indicates that the given tunnel name
// is not supported by this package.
var ErrUnsupportedTunnelName = errors.New("unsupported tunnel name")

// DebugInfo contains information useful to debug issues
// when starting up a given tunnel fails.
type DebugInfo struct {
	// LogFilePath is the path to the log file, which MAY
	// be empty in case we don't have a log file.
	LogFilePath string

	// Name is the name of the tunnel and will always
	// be properly set by the code.
	Name string

	// Version is the tunnel version. This field MAY be
	// empty if we don't know the version.
	Version string
}

// Start starts a new tunnel by name or returns an error. We currently
// support the following tunnels:
//
// The "tor" tunnel requires the "tor" binary to be installed on
// your system. You can use config.TorArgs and config.TorBinary to
// select what binary to execute and with which arguments.
//
// The "psiphon" tunnel requires a configuration. Some builds of
// ooniprobe embed a configuration into the binary. When this
// is the case, the config.Session is a mocked object that just
// returns such a configuration.
//
// Otherwise, If there is no embedded psiphon configuration, the
// config.Session must be an ordinary engine.Session. In such a
// case, fetching the Psiphon configuration from the backend may
// fail when the backend is not reachable.
//
// The "fake" tunnel is a fake tunnel that just exposes a
// SOCKS5 proxy and then connects directly to server. We use
// this special kind of tunnel to implement tests.
//
// The return value is a triple:
//
// 1. a valid Tunnel on success, nil on failure;
//
// 2. debugging information (both on success and failure);
//
// 3. nil on success, an error on failure.
func Start(ctx context.Context, config *Config) (Tunnel, DebugInfo, error) {
	switch config.Name {
	case "fake":
		return fakeStart(ctx, config)
	case "psiphon":
		return psiphonStart(ctx, config)
	case "torsf":
		return torsfStart(ctx, config)
	case "tor":
		return torStart(ctx, config)
	default:
		di := DebugInfo{}
		return nil, di, fmt.Errorf("%w: %s", ErrUnsupportedTunnelName, config.Name)
	}
}
