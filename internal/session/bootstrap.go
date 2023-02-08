package session

import (
	"context"
	"errors"
)

// BootstrapRequest is a request to bootstrap the [Session] and
// contains the arguments requested by the bootstrap. You can
// boostrap a [Session] just once. All operations you would like
// to perform with a [Session] require a boostrap first.
type BootstrapRequest struct {
	// SnowflakeRendezvousMethod is the OPTIONAL rendezvous
	// method to use when ProxyURL scheme is `torsf`.
	SnowflakeRendezvousMethod string

	// StateDir is the MANDATORY directory where to store
	// persistent engine state using a key-value store.
	StateDir string

	// ProxyURL is the OPTIONAL proxy URL. We accept the
	// following proxy URL schemes:
	//
	// - "socks5": configures a socks5 proxy;
	//
	// - "tor": requires a tor tunnel;
	//
	// - "torsf": requires a tor+snowflake tunnel;
	//
	// - "psiphon": requires a psiphon tunnel.
	//
	// When requesting a tunnel, we only check the URL scheme
	// and disregard the rest of the URL.
	ProxyURL string

	// SoftwareName is the MANDATORY software name.
	SoftwareName string

	// SoftwareVersion is the MANDATORY software version.
	SoftwareVersion string

	// TorArgs OPTIONALLY passes command line arguments to tor
	// when the ProxyURL scheme is "tor" or "torsf".
	TorArgs []string

	// TorBinary OPTIONALLY tells the engine to use a specific
	// binary for starting the "tor" and "torsf" tunnels.
	TorBinary string

	// TempDir is the MANDATORY base directory in which
	// the session should store temporary state.
	TempDir string

	// TunnelDir is the MANDATORY directory in which
	// to store persistent tunnel state.
	TunnelDir string

	// VerboseLogging OPTIONALLY enables verbose logging.
	VerboseLogging bool
}

// BootstrapEvent is the event emmitted at the end of the bootstrap.
type BootstrapEvent struct {
	// Error is the bootstrap result.
	Error error
}

// boostrap bootstraps a session.
func (s *Session) bootstrap(ctx context.Context, req *Request) {
	s.emit(&Event{
		Bootstrap: &BootstrapEvent{
			Error: s.dobootstrap(ctx, req),
		},
	})
}

// ErrAlreadyBootstrapped indicates that we already bootstrapped a [Session].
var ErrAlreadyBootstrapped = errors.New("session: already bootstrapped")

// dobootstrap implements bootstrap.
func (s *Session) dobootstrap(ctx context.Context, req *Request) error {
	if s.state != nil {
		return ErrAlreadyBootstrapped
	}
	state, err := s.newState(ctx, req)
	if err != nil {
		return err
	}
	s.state = state
	return nil
}
