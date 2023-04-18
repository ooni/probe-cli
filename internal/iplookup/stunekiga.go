package iplookup

//
// IP lookup using ekiga STUN
//

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/fallback"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// ekigaSTUNLookup implements fallback.Service
type ekigaSTUNLookup struct {
	client *Client
}

var _ fallback.Service[model.AddressFamily, string] = &ekigaSTUNLookup{}

// newEkigaSTUNLookup creates a new [ekigaSTUNLookup] instance.
func newEkigaSTUNLookup(client *Client) *ekigaSTUNLookup {
	return &ekigaSTUNLookup{client}
}

// Run implements fallback.Service
func (svc *ekigaSTUNLookup) Run(ctx context.Context, family model.AddressFamily) (string, error) {
	return svc.client.lookupSTUNDomainPort(ctx, family, "stun.ekiga.net", "3478")
}

// URL implements fallback.Service
func (svc *ekigaSTUNLookup) URL() string {
	return "iplookup+stun://ekiga/"
}
