package oobackend

import (
	"bytes"
	"errors"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/internal/multierror"
)

// TODO(bassosimone): apply similar changes to sessionresolver. In
// particular, let's avoid writing state in sessionresolver.Resolver
// by making the KVStore mandatory, which is a small change.

// Client is a client to speak to the OONI backend. In its
// default configuration (i.e., with no proxies), this client
// uses the best strategy to speak to the backend. Make sure
// to fill the mandatory fields before using a Client.
type Client struct {
	// KVStore is the mandatory KVStore from which we read
	// the state and in which we write the state.
	KVStore KVStore

	// HTTPClientWithProxy is an optional HTTP client that is
	// configured to always use a proxy. When this client is
	// present, we always and only use it, and we do not update
	// our strategy dynamically. In fact, when this client is
	// present, it means the user has requested the probe to
	// use a specific proxy explicitly.
	HTTPClientWithProxy HTTPClient

	// HTTPClientDefault is the HTTP client we use by default
	// when HTTPClientWithProxy is not set. If HTTPClientDefault
	// is not set, we use http.DefaultClient.
	HTTPClientDefault HTTPClient

	// HTTPTunnelBroker is a broker for creating HTTPTunnel
	// instances. If this field is not set, then any strategy
	// based on using a tunnel will fail.
	HTTPTunnelBroker HTTPTunnelBroker

	// codec is the optional codec to use when serializing and
	// deserializing the state. By default we use JSON.
	codec codec
}

// ErrBackend indicates we cannot contact the OONI backend.
var ErrBackend = errors.New("oobackend: cannot contact the OONI backend")

// Do sends the given request and returns the corresponding
// response, on success, or an error, on failure.
//
// If we have a configured HTTPClientWithProxy, we unconditionally
// use it. Otherwise, we try all strategies sorted by score. The
// score changes depending on whether a strategy works.
//
// With low probability, we reset the score of strategies so we
// are able to evaluate whether anything changed.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	if c.HTTPClientWithProxy != nil {
		return c.HTTPClientWithProxy.Do(req) // just use the user defined proxy
	}
	return c.do(req) // try all the strategies
}

// do tries with every available strategy.
func (c *Client) do(req *http.Request) (*http.Response, error) {
	body, err := c.readbody(req) // see below for why
	if err != nil {
		return nil, err
	}
	sgs := c.readstrategies(time.Now().UnixNano())
	me := multierror.New(ErrBackend)
	defer c.writestrategies(sgs) // update the datastore
	for _, sg := range sgs {
		// Note: we pass to sg.Do a copy of the request so it's free
		// to apply any required transformations. Because sending a
		// request consumes the request body, we need to recreate the
		// original body every time we're about to call Do.
		resp, err := sg.Do(c.clonerequest(req, body))
		if err != nil {
			me.Add(err)
			continue
		}
		return resp, err
	}
	return nil, me
}

// readbody reads the request body, if any.
func (c *Client) readbody(req *http.Request) ([]byte, error) {
	if req.Body == nil {
		return nil, nil // nil is an empty []byte
	}
	return ioutil.ReadAll(req.Body)
}

// clonerequest clones a request with optional body.
func (c *Client) clonerequest(req *http.Request, body []byte) *http.Request {
	req = req.Clone(req.Context()) // same ctx
	if body != nil {
		req.Body = ioutil.NopCloser(bytes.NewReader(body))
	}
	return req
}
