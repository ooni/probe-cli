package iplookup

//
// Client definition
//

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/ooni/probe-cli/v3/internal/fallback"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// ErrAllEndpointsFailed indicates that we failed to lookup
// with all the available endpoints we tried.
var ErrAllEndpointsFailed = errors.New("iplookup: all endpoints failed")

// ErrHTTPRequestFailed indicates that an HTTP request failed.
var ErrHTTPRequestFailed = errors.New("iplookup: http request failed")

// ErrInvalidIPAddressForFamily indicates that a string expected to be a valid IP
// address was not a valid IP address for the family we're resolving for.
var ErrInvalidIPAddressForFamily = errors.New("iplookup: invalid IP address for family")

// defaultTimeout is the default timeout we use when
// performing the IP lookup.
const defaultTimeout = 7 * time.Second

// Client is an IP lookup client. The zero value of this struct is
// invalid; please, use [NewClient] to construct.
type Client struct {
	// kvStore is the [model.KeyValueStore] to use.
	kvStore model.KeyValueStore

	// logger is the [model.Logger] to use.
	logger model.Logger

	// resolver is the [model.Resolver] to use.
	resolver model.Resolver

	// testingHTTPDo is an OPTIONAL hook to override the default function
	// called to issue an HTTP request and read the response body.
	testingHTTPDo func(req *http.Request) ([]byte, error)
}

// NewClient creates a new [Client].
//
// Arguments:
//
// - kvStore is the key-value store to keep persistent state;
//
// - logger is the logger to use;
//
// - resolver is the resolve to use when resolving the domain name of
// services helping with IP lookups. Since there may be DNS-level censorship,
// we recommend passing as argument a DNS-over-HTTPS resolver such as the
// one implemented in the [sessionresolver] package.
func NewClient(
	kvStore model.KeyValueStore,
	logger model.Logger,
	resolver model.Resolver,
) *Client {
	return &Client{
		kvStore:       kvStore,
		logger:        logger,
		resolver:      resolver,
		testingHTTPDo: nil,
	}
}

// LookupIPAddr resolves the probe IP address.
//
// Arguments:
//
// - ctx is the context allowing to interrupt this function earlier;
//
// - family is the [model.AddressFamily] you want us to exclusively use.
//
// When the family is [model.AddressFamilyINET], this function tries
// to find out the probe's IPv4 address; when it is [model.AddressFamilyINET6],
// this function tries to find out the probe's IPv6 address.
func (c *Client) LookupIPAddr(ctx context.Context, family model.AddressFamily) (string, error) {
	// create director for coordinating fallback
	director := newDirector(c)

	// create the services
	services := []fallback.Service[model.AddressFamily, string]{
		newFamilyWrapperLookup(newCloudflareWebLookup(c)),
		newFamilyWrapperLookup(newEkigaSTUNLookup(c)),
		newFamilyWrapperLookup(newGoogleSTUNLookup(c)),
		newFamilyWrapperLookup(newUbuntuWebLookup(c)),
	}

	// resolve the probe IP address
	return fallback.Run(ctx, director, family, services...)
}
