package oobackend

import (
	"context"
	"net/http"
)

// HTTPClient is a generic HTTP client.
type HTTPClient interface {
	// Do should behave like http.Client.Do.
	Do(req *http.Request) (*http.Response, error)
}

// KVStore is a generic key-value store. We use it to store
// on disk persistent state used by this package.
type KVStore interface {
	// Get gets the value for the given key.
	Get(key string) ([]byte, error)

	// Set sets the value of the given key.
	Set(key string, value []byte) error
}

// HTTPTunnel tunnels an HTTP request over some circumvention
// mechanism and returns the result.
type HTTPTunnel interface {
	// Do should behave like http.Client.Do.
	Do(req *http.Request) (*http.Response, error)
}

// HTTPTunnelBroker allows to create and use HTTP tunnels. The
// broker SHOULD support a tunnel named "psiphon". If the psiphon
// configuration is not present, creating the tunnel will fail.
type HTTPTunnelBroker interface {
	// New creates an instance of the named tunnel. The returned
	// instance may be a cached instance. The result is a valid
	// HTTPTunnel, on success, and an error, on failure.
	New(ctx context.Context, name string) (HTTPTunnel, error)
}
