// Package mocks contains mocks for tunnel.
package mocks

import (
	"net/url"
	"time"

	"github.com/ooni/probe-cli/v3/internal/tunnel"
)

// Tunnel allows mocking a tunnel.
type Tunnel struct {
	// MockBootstrapTime allows to mock BootstrapTime.
	MockBootstrapTime func() time.Duration

	// MockSOCKS5ProxyURL allows to mock Socks5ProxyURL.
	MockSOCKS5ProxyURL func() *url.URL

	// MockStop allows to mock Stop.
	MockStop func()
}

func (t *Tunnel) BootstrapTime() time.Duration {
	return t.MockBootstrapTime()
}

// SOCKS5ProxyURL implements Tunnel.SOCKS5ProxyURL.
func (t *Tunnel) SOCKS5ProxyURL() *url.URL {
	return t.MockSOCKS5ProxyURL()
}

// Stop implements Tunnel.Stop.
func (t *Tunnel) Stop() {
	t.MockStop()
}

var _ tunnel.Tunnel = &Tunnel{}
