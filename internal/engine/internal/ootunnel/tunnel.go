package ootunnel

import (
	"net/url"
	"time"
)

// Config contains configuration for creating a tunnel.
type Config struct {
	// DeleteStateDirOnClose specifies whether to delete the
	// StateDir directory on close.
	DeleteStateDirOnClose bool

	// Name is the mandatory name of the tunnel.
	Name string

	// StateDir is the mandatory directory where to store
	// the state required by this tunnel.
	StateDir string

	// TorArgs contains optional extra arguments
	// for running the tor binary.
	TorArgs []string

	// TorBinary contains the optional path
	// to the tor binary.
	TorBinary string
}

// ManagedTunnel is a tunnel owned by the Broker.
type ManagedTunnel interface {
	// BootstrapTime returns the bootstrap time.
	BootstrapTime() time.Duration

	// Name returns the tunnel name.
	Name() string

	// ProxyURL returns the tunnel proxy URL.
	ProxyURL() *url.URL

	// StateDir returns the directory containing state.
	StateDir() string
}

// Tunnel is a circumvention tunnel.
type Tunnel interface {
	// Close closes the tunnel. Depending on the configuration, this
	// MAY also wipe the StateDir you did configure. This function
	// is guaranteed to be idempotent.
	Close() error

	// ManagedTunnel embeds the managed tunnel behavior.
	ManagedTunnel
}
