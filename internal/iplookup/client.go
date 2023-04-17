package iplookup

//
// Client definition
//

import (
	"context"
	"errors"
	"math/rand"
	"net/http"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/multierror"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// ErrAllEndpointsFailed indicates that we failed to lookup
// with all the available endpoints we tried.
var ErrAllEndpointsFailed = errors.New("iplookup: all endpoints failed")

// ErrAllMethodsFailed indicates that we failed to lookup
// with all the [Method] we tried.
var ErrAllMethodsFailed = errors.New("iplookup: all methods failed")

// ErrHTTPRequestFailed indicates that an HTTP request failed.
var ErrHTTPRequestFailed = errors.New("iplookup: http request failed")

// ErrInvalidIPAddress indicates that a string expected to be a valid IP
// address was not a valid IP address.
var ErrInvalidIPAddress = errors.New("iplookup: invalid IP address")

// ErrNoSuchMethod indicates that you asked for a nonexisting [Method].
var ErrNoSuchMethod = errors.New("iplookup: no such method")

// defaultTimeout is the default timeout we use when
// performing the IP lookup.
const defaultTimeout = 7 * time.Second

// Client is an IP lookup client. The zero value of this struct is
// invalid; please, fill all the fields marked as MANDATORY.
type Client struct {
	// Logger is the MANDATORY [model.Logger] to use.
	Logger model.Logger

	// Resolver is the MANDATORY [model.Resolver] to use. We recommend
	// using a DNS-over-HTTPS resolver here, with fallback to the system
	// resolver, to reduce the chances that DNS censorship could cause
	// the IP lookup procedure to fail.
	Resolver model.Resolver

	// TestingHTTPDo is an OPTIONAL hook to override the default function
	// called to issue an HTTP request and read the response body.
	TestingHTTPDo func(req *http.Request) ([]byte, error)
}

// Method is an IP lookup method.
type Method string

// MethodAllRandom tries all the available methods in
// random order until one succeeds.
const MethodAllRandom = Method("all_random")

// MethodSTUNEkiga uses a STUN endpoint exposed by Ekiga.
const MethodSTUNEkiga = Method("stun_ekiga")

// MethodSTUNGoogle uses a STUN endpoint exposed by Google.
const MethodSTUNGoogle = Method("stun_google")

// MethodWebCloudflare uses a Web API exposed by Cloudflare.
const MethodWebClouflare = Method("web_cloudflare")

// MethodWebUbuntu uses a Web API exposed by Ubuntu.
const MethodWebUbuntu = Method("web_ubuntu")

// LookupIPAddr resolves the probe IP address.
//
// Arguments:
//
// - ctx is the context allowing to interrupt this function earlier;
//
// - method is the IP lookup method you would like us to use;
//
// - family is the address family you want us to use.
//
// The return value is either the discovered IPv4-or-IPv6 probe IP
// address or the error that occurred when trying to discover it.
func (c *Client) LookupIPAddr(
	ctx context.Context,
	method Method,
	family model.AddressFamily,
) (string, error) {
	var methods []Method

	// fill the methods list depending on the user preference
	switch method {
	case MethodAllRandom:
		methods = append(methods, MethodSTUNEkiga)
		methods = append(methods, MethodSTUNGoogle)
		methods = append(methods, MethodWebClouflare)
		methods = append(methods, MethodWebUbuntu)
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		r.Shuffle(len(methods), func(i, j int) {
			methods[i], methods[j] = methods[j], methods[i]
		})

	case MethodSTUNEkiga, MethodSTUNGoogle, MethodWebClouflare, MethodWebUbuntu:
		methods = append(methods, method)

	default:
		return "", ErrNoSuchMethod
	}

	// try each method in sequence
	me := multierror.New(ErrAllMethodsFailed)
	for _, method := range methods {
		addr, err := c.lookupMethod(ctx, method, family)
		if err != nil {
			me.Add(err)
			continue
		}
		return addr, nil
	}

	return "", me
}

// lookupMethod performs the IP lookup using the given method and the given family.
func (c *Client) lookupMethod(
	ctx context.Context,
	method Method,
	family model.AddressFamily,
) (string, error) {
	switch method {
	case MethodSTUNEkiga:
		return c.lookupSTUN(ctx, family, "stun.ekiga.net", "3478")

	case MethodSTUNGoogle:
		return c.lookupSTUN(ctx, family, "stun.l.google.com", "19302")

	case MethodWebClouflare:
		return c.lookupCloudflare(ctx, family)

	case MethodWebUbuntu:
		return c.lookupUbuntu(ctx, family)

	default:
		return "", ErrNoSuchMethod
	}
}

// httpDo is the common function to issue an HTTP request and get the response body.
func (c *Client) httpDo(req *http.Request, family model.AddressFamily) ([]byte, error) {
	// honour the TestingHTTPDo hook, if needed.
	if c.TestingHTTPDo != nil {
		return c.TestingHTTPDo(req)
	}

	// create HTTP client
	//
	// Note: we're using the family-specific resolver which ensures that we're not
	// going to use IP addresses for the wrong address family.
	httpClient := netxlite.NewHTTPClientWithResolver(c.Logger, c.newFamilyResolver(family))
	defer httpClient.CloseIdleConnections()

	// issue HTTP request and get response
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// make sure the request succeded
	if resp.StatusCode != 200 {
		return nil, ErrHTTPRequestFailed
	}

	// read response body
	return netxlite.ReadAllContext(req.Context(), resp.Body)
}

// newFamilyResolver creates a new [model.Resolver] using the given family
// and the underlying [model.Resolver] used by the [Client].
func (c *Client) newFamilyResolver(family model.AddressFamily) model.Resolver {
	return netxlite.NewAddressFamilyResolver(c.Resolver, family)
}
