package iplookup

//
// IP lookup using google STUN
//

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/fallback"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// googleSTUNLookup implements fallback.Service
type googleSTUNLookup struct {
	client *Client
}

var _ fallback.Service[model.AddressFamily, string] = &googleSTUNLookup{}

// newGoogleSTUNLookup creates a new [googleSTUNLookup] instance.
func newGoogleSTUNLookup(client *Client) *googleSTUNLookup {
	return &googleSTUNLookup{client}
}

// Run implements fallback.Service
func (svc *googleSTUNLookup) Run(ctx context.Context, family model.AddressFamily) (string, error) {
	return svc.client.lookupSTUNDomainPort(ctx, family, "stun.l.google.com", "19302")
}

// URL implements fallback.Service
func (svc *googleSTUNLookup) URL() string {
	return "iplookup+stun://google/"
}
