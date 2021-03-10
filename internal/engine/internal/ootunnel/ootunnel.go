// Package ootunnel contains code for managing circumvention tunnels
// of different types, e.g., Tor and Psiphon.
//
// To create tunnels, you use a Broker.
//
// There are two kind of tunnels: Tunnels and ManagedTunnels. A Tunnel
// is a tunnel that you own. That is, you MUST call its Close method
// when done to free resources. A ManagedTunnel's lifecycle, instead, is
// managed by the Broker. So, it will be closed when you call the
// Close method of the Broker.
//
// While you can have several Tunnels per type, a Broker will allow
// you to create a single ManagedTunnel per type.
//
// You should use Tunnels for one-off operations and ManagedTunnels for
// persistent tunnels. For example, you can create and use a ManagedTunnel
// for speaking with the OONI backend over a specific tunnel type.
package ootunnel

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"sync"
)

// Available tunnels.
const (
	Psiphon = "psiphon"
	Tor     = "tor"
)

// Errors returned by this package.
var (
	ErrEmptyStateDir       = errors.New("ootunnel: StateDir is empty")
	ErrNoSOCKSProxy        = errors.New("ootunnel: cannot get SOCKS proxy address")
	ErrNoSuchTunnel        = errors.New("ootunnel: no such tunnel")
	ErrTunnelAlreadyExists = errors.New("ootunnel: managed tunnel already exists")
	ErrUnsupportedProxy    = errors.New("ootunnel: unsupported proxy")
)

// Broker creates and manages tunnels.
type Broker struct {
	// mu protects this data structure.
	mu sync.Mutex

	// mkdirAll allows mocking os.MkdirAll
	mkdirAll func(path string, perm fs.FileMode) error

	// torLibrary is the optional torLibrary to use.
	torLibrary torLibrary

	// tuns contains managed tunnels.
	tuns map[string]Tunnel
}

// NewTunnel creates a new tunnel instance. You own the returned
// tunnel and must Close it when done.
func (b *Broker) NewTunnel(ctx context.Context, config *Config) (Tunnel, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		// fallthrough
	}
	if config.StateDir == "" {
		return nil, ErrEmptyStateDir
	}
	switch config.Name {
	case Tor:
		return b.newTor(ctx, config)
	case Psiphon:
		return b.newPsiphon(ctx, config)
	default:
		return nil, fmt.Errorf("%w: %s", ErrNoSuchTunnel, config.Name)
	}
}

// NewManagedTunnel constructs a managed tunnel with the given
// config.Name. It returns ErrTunnelAlreadyExists if we're already
// managing a tunnel with such config.Name. It may also return
// other errors. On success, it returns nil. In such case,
// use GetManagedTunnel to get the ManagedTunnel.
func (b *Broker) NewManagedTunnel(ctx context.Context, config *Config) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		// fallthrough
	}
	defer b.mu.Unlock()
	b.mu.Lock()
	tun, found := b.tuns[config.Name]
	if found == true {
		return fmt.Errorf("%w: %s", ErrTunnelAlreadyExists, config.Name)
	}
	tun, err := b.NewTunnel(ctx, config)
	if err != nil {
		return err
	}
	if b.tuns == nil {
		b.tuns = make(map[string]Tunnel)
	}
	b.tuns[config.Name] = tun
	return nil
}

// GetManagedTunnel returns a ManagedTunnel with the given name. If there
// is no such tunnel, this function returns nil and false. Otherwise, it
// returns a valid ManagedTunnel and true.
func (b *Broker) GetManagedTunnel(name string) (ManagedTunnel, bool) {
	defer b.mu.Unlock()
	b.mu.Lock()
	tun, ok := b.tuns[name]
	return tun, ok
}

// Close closes all the ManagedTunnel instances.
func (b *Broker) Close() error {
	defer b.mu.Unlock()
	b.mu.Lock()
	for _, tun := range b.tuns {
		tun.Close()
	}
	b.tuns = make(map[string]Tunnel) // clear
	return nil
}
