package iplookup

import (
	"time"

	"github.com/ooni/probe-cli/v3/internal/fallback"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// director implements [fallback.Director]
type director struct {
	client *Client
}

// newDirector creates a new [director] instance.
func newDirector(client *Client) *director {
	return &director{client}
}

var _ fallback.Director = &director{}

// KVStore implements fallback.Director
func (d *director) KVStore() model.KeyValueStore {
	return d.client.kvStore
}

// Key implements fallback.Director
func (d *director) Key() string {
	return "iplookup.state"
}

// ShuffleEvery implements fallback.Director
func (d *director) ShuffleEvery() time.Duration {
	return 300 * time.Second
}

// TimeNow implements fallback.Director
func (d *director) TimeNow() time.Time {
	return time.Now()
}
