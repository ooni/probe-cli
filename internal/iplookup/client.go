package iplookup

//
// Client definition
//

import (
	"context"
	"errors"
	"math/rand"
	"net"
	"net/http"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/multierror"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/stunx"
)

// ErrAllEndpointsFailed indicates that we failed to lookup
// with all the available endpoints we tried.
var ErrAllEndpointsFailed = errors.New("iplookup: all endpoints failed")

// ErrAllMethodsFailed indicates that we failed to lookup
// with all the [Method] we tried.
var ErrAllMethodsFailed = errors.New("iplookup: all methods failed")

// ErrHTTPRequestFailed indicates that an HTTP request failed.
var ErrHTTPRequestFailed = errors.New("iplookup: http request failed")

// ErrInvalidIPAddressForFamily indicates that a string expected to be a valid IP
// address was not a valid IP address for the family we're resolving for.
var ErrInvalidIPAddressForFamily = errors.New("iplookup: invalid IP address for family")

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
// - family is the [model.AddressFamily] you want us to exclusively use.
//
// When the family is [model.AddressFamilyINET], this function tries
// to find out the probe's IPv4 address; when it is [model.AddressFamilyINET6],
// this function tries to find out the probe's IPv6 address.
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
) (addr string, err error) {
	// issue the lookup using the proper method
	switch method {
	case MethodSTUNEkiga:
		addr, err = c.lookupSTUNDomainPort(ctx, family, "stun.ekiga.net", "3478")

	case MethodSTUNGoogle:
		addr, err = c.lookupSTUNDomainPort(ctx, family, "stun.l.google.com", "19302")

	case MethodWebClouflare:
		addr, err = c.lookupCloudflare(ctx, family)

	case MethodWebUbuntu:
		addr, err = c.lookupUbuntu(ctx, family)

	default:
		addr, err = "", ErrNoSuchMethod
	}

	// immediately handle errors
	if err != nil {
		return "", err
	}

	// make sure the IP address is valid for the expected family
	if !netxlite.AddressBelongsToAddressFamily(addr, family) {
		return "", ErrInvalidIPAddressForFamily
	}

	// finally, return the result
	return addr, nil
}

// lookupSTUNDomainPort performs the lookup using the STUN server at the given domain and port.
func (c *Client) lookupSTUNDomainPort(
	ctx context.Context, family model.AddressFamily, domain, port string) (string, error) {
	// Note: create an address-family aware resolver to make sure we're not
	// going to use an IP addresses belongong to the wrong family.
	reso := c.newAddressFamilyResolver(family)

	// resolve the given domain name to IP addresses
	addrs, err := reso.LookupHost(ctx, domain)
	if err != nil {
		return "", err
	}

	// try each available address in sequence until one of them works
	for _, addr := range addrs {
		// create the destination endpoint
		endpoint := net.JoinHostPort(addr, port)

		// resolve the external address
		publicAddr, err := c.lookupSTUNEndpoint(ctx, endpoint)
		if err != nil {
			continue
		}
		return publicAddr, nil
	}

	return "", ErrAllEndpointsFailed
}

// lookupSTUNEndpoint uses the given STUN endpoint to lookup the IP address.
func (c *Client) lookupSTUNEndpoint(ctx context.Context, endpoint string) (string, error) {
	// make sure we eventually time out
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	// create client and lookup the IP address
	client := stunx.NewClient(endpoint, c.Logger)
	return client.LookupIPAddr(ctx)
}

// httpDo is the common function to issue an HTTP request and get the response body.
func (c *Client) httpDo(req *http.Request, family model.AddressFamily) ([]byte, error) {
	// honour the TestingHTTPDo hook, if needed.
	if c.TestingHTTPDo != nil {
		return c.TestingHTTPDo(req)
	}

	// make sure we eventually time out
	ctx := req.Context()
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	req = req.WithContext(ctx)

	// create HTTP client
	//
	// Note: create an address-family aware resolver to make sure we're not
	// going to use an IP addresses belongong to the wrong family.
	httpClient := netxlite.NewHTTPClientWithResolver(c.Logger, c.newAddressFamilyResolver(family))
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

// newAddressFamilyResolver creates a new [model.Resolver] using the given address
// family and the underlying [model.Resolver] used by the [Client].
func (c *Client) newAddressFamilyResolver(family model.AddressFamily) model.Resolver {
	return netxlite.NewAddressFamilyResolver(c.Resolver, family)
}
