package tortunnel

//
// Config implementation
//

import (
	"os"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// Config contains config for starting the tor tunnel.
type Config struct {
	// BootstrapEvents is the OPTIONAL channel where to send
	// events emitted during the bootstrap.
	BootstrapEvents chan<- string `json:",omitempty"`

	// Dependencies is OPTIONAL and allow one to mock the functions
	// called by Start, which is mainly useful for testing.
	Dependencies *Dependencies `json:",omitempty"`

	// Logger is the OPTIONAL logger to use during the bootstrap.
	Logger model.Logger `json:",omitempty"`

	// SnowflakeEnabled OPTIONALLY enables snowflake.
	SnowflakeEnabled bool

	// SnowflakeRendezvousMethod is the OPTIONAL snowflake rendezvous method.
	SnowflakeRendezvousMethod string

	// TunnelDir is the OPTIONAL directory in which to store state.
	TunnelDir string

	// TorArgs contains OPTIONAL arguments to pass to tor.
	TorArgs []string

	// TorBinary OPTIONAL tor binary path.
	TorBinary string

	// TorVersion is the OPTIONAL channel where we send the version
	// of tor that we are attempting to bootstrap.
	TorVersion chan<- string `json:",omitempty"`
}

// logger always returns a valid instance of [model.Logger].
func (cc *Config) logger() model.Logger {
	if cc.Logger == nil {
		return model.DiscardLogger
	}
	return cc.Logger
}

// dependencies always returns a valid instance of [Dependencies].
func (cc *Config) dependencies() *Dependencies {
	if cc.Dependencies == nil {
		return defaultDependencies
	}
	return cc.Dependencies
}

// tunnelDir returns the tunnel dir to use or an error.
func (cc *Config) tunnelDir(logger model.Logger) (string, func(), error) {
	if cc.TunnelDir == "" {
		dir, err := os.MkdirTemp("", "")
		if err != nil {
			return "", nil, err
		}
		return dir, cleanupTunnelDir(logger, dir), nil
	}
	return cc.TunnelDir, func() {}, nil
}

// cleanupTunnelDir returns a function that removes a temporary tunnel dir.
func cleanupTunnelDir(logger model.Logger, dir string) func() {
	return func() {
		maybeRemoveDir(logger, dir)
	}
}

// maybeRemoveDir removes a directory if needed.
func maybeRemoveDir(logger model.Logger, dir string) {
	logger.Infof("rm -rf %s", dir)
	os.RemoveAll(dir)
}
