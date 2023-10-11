package enginelocate

import (
	"context"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"net/http"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/multierror"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

var (
	// ErrAllIPLookuppersFailed indicates that we failed with looking
	// up the probe IP for with all the lookuppers that we tried.
	ErrAllIPLookuppersFailed = errors.New("all IP lookuppers failed")

	// ErrInvalidIPAddress indicates that the code returned to us a
	// string that actually isn't a valid IP address.
	ErrInvalidIPAddress = errors.New("lookupper did not return a valid IP")
)

type lookupFunc func(
	ctx context.Context, client *http.Client,
	logger model.Logger, userAgent string,
	resolver model.Resolver,
) (string, error)

type method struct {
	name string
	fn   lookupFunc
}

var (
	methods = []method{
		{
			name: "cloudflare",
			fn:   cloudflareIPLookup,
		},
		{
			name: "stun_ekiga",
			fn:   stunEkigaIPLookup,
		},
		{
			name: "stun_google",
			fn:   stunGoogleIPLookup,
		},
		{
			name: "ubuntu",
			fn:   ubuntuIPLookup,
		},
	}
)

type ipLookupClient struct {
	// Resolver is the resolver to use for HTTP.
	Resolver model.Resolver

	// Logger is the logger to use
	Logger model.Logger

	// UserAgent is the user agent to use
	UserAgent string
}

func makeSlice() []method {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	ret := make([]method, len(methods))
	perm := r.Perm(len(methods))
	for idx, randIdx := range perm {
		ret[idx] = methods[randIdx]
	}
	return ret
}

func contextForIPLookupWithTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	// TODO(https://github.com/ooni/probe/issues/2551): we must enforce a timeout this
	// large to ensure we give all resolvers a chance to run. We set this value as part of
	// an hotfix. The above mentioned issue explains how to improve the situation and
	// avoid the need of setting such large timeouts here.
	const timeout = 45 * time.Second
	return context.WithTimeout(ctx, timeout)
}

func (c ipLookupClient) doWithCustomFunc(
	ctx context.Context, fn lookupFunc,
) (string, error) {
	ctx, cancel := contextForIPLookupWithTimeout(ctx)
	defer cancel()

	// Implementation note: we MUST use an HTTP client that we're
	// sure IS NOT using any proxy. To this end, we construct a
	// client ourself that we know is not proxied.
	// TODO(https://github.com/ooni/probe/issues/2534): the NewHTTPTransportWithResolver has QUIRKS but
	// we don't care about them in this context
	txp := netxlite.NewHTTPTransportWithResolver(c.Logger, c.Resolver)
	clnt := &http.Client{Transport: txp}
	defer clnt.CloseIdleConnections()
	ip, err := fn(ctx, clnt, c.Logger, c.UserAgent, c.Resolver)
	if err != nil {
		return model.DefaultProbeIP, err
	}
	if net.ParseIP(ip) == nil {
		return model.DefaultProbeIP, fmt.Errorf("%w: %s", ErrInvalidIPAddress, ip)
	}
	c.Logger.Debugf("iplookup: IP: %s", ip)
	return ip, nil
}

func (c ipLookupClient) LookupProbeIP(ctx context.Context) (string, error) {
	union := multierror.New(ErrAllIPLookuppersFailed)
	for _, method := range makeSlice() {
		c.Logger.Infof("iplookup: using %s", method.name)
		ip, err := c.doWithCustomFunc(ctx, method.fn)
		if err == nil {
			return ip, nil
		}
		union.Add(err)
	}
	return model.DefaultProbeIP, union
}
