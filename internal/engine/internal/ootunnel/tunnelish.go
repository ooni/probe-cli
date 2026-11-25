package ootunnel

import (
	"io"
	"net/url"
	"os"
	"sync"
	"time"
)

// tunnelish is the common structure to all tunnels.
type tunnelish struct {
	bootstrapTime         time.Duration // required
	closeOnce             sync.Once     // optional
	deleteStateDirOnClose bool          // optional
	name                  string        // required
	proxyURL              *url.URL      // required
	stateDir              string        // required
	t                     io.Closer     // required
}

// BootstrapTime implements Tunnel.BootstrapTime.
func (tu *tunnelish) BootstrapTime() time.Duration {
	return tu.bootstrapTime
}

// Close implements Tunnel.Close.
func (tu *tunnelish) Close() error {
	tu.closeOnce.Do(tu.doClose) // idempotent
	return nil
}

// doClose implements Close.
func (tu *tunnelish) doClose() {
	tu.t.Close()
	if tu.deleteStateDirOnClose == true {
		os.RemoveAll(tu.stateDir)
	}
}

// Name implements Tunnel.Name.
func (tu *tunnelish) Name() string {
	return tu.name
}

// ProxyURL implements Tunnel.ProxyURL.
func (tu *tunnelish) ProxyURL() *url.URL {
	return tu.proxyURL
}

// StateDir implements Tunnel.StateDir.
func (tu *tunnelish) StateDir() string {
	return tu.stateDir
}
