package netxlite

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/netxlite/dnsx"
	"github.com/ooni/probe-cli/v3/internal/netxlite/errorsx"
	"golang.org/x/net/idna"
)

// HTTPSSvc is the type returned for HTTPSSvc queries.
type HTTPSSvc = dnsx.HTTPSSvc

// Resolver performs domain name resolutions.
type Resolver interface {
	// LookupHost behaves like net.Resolver.LookupHost.
	LookupHost(ctx context.Context, hostname string) (addrs []string, err error)

	// Network returns the resolver type (e.g., system, dot, doh).
	Network() string

	// Address returns the resolver address (e.g., 8.8.8.8:53).
	Address() string

	// CloseIdleConnections closes idle connections, if any.
	CloseIdleConnections()

	// LookupHostWithoutRetry issues a single lookup host query
	// for the given qtype (dns.TypeA or dns.TypeAAAA) without any
	// retry mechanism whatsoever.
	LookupHostWithoutRetry(
		ctx context.Context, domain string, qtype uint16) ([]string, error)

	// LookupHTTPSSvcWithoutRetry issues a single HTTPS query for
	// a domain without any retry mechanism whatsoever.
	LookupHTTPSSvcWithoutRetry(
		ctx context.Context, domain string) (HTTPSSvc, error)
}

// ErrNoDNSTransport indicates that the requested Resolver operation
// cannot be performed because we're using the "system" resolver.
var ErrNoDNSTransport = errors.New("operation requires a DNS transport")

// NewResolverStdlib creates a new resolver using system
// facilities for resolving domain names (e.g., getaddrinfo).
//
// The resolver will provide the following guarantees:
//
// 1. handles IDNA;
//
// 2. performs logging;
//
// 3. short-circuits IP addresses like getaddrinfo does (i.e.,
// resolving "1.1.1.1" yields []string{"1.1.1.1"};
//
// 4. wraps errors;
//
// 5. enforces reasonable timeouts (
// see https://github.com/ooni/probe/issues/1726).
func NewResolverStdlib(logger Logger) Resolver {
	return WrapResolver(logger, &resolverSystem{})
}

func WrapResolver(logger Logger, resolver Resolver) Resolver {
	return &resolverIDNA{
		Resolver: &resolverLogger{
			Resolver: &resolverShortCircuitIPAddr{
				Resolver: &resolverErrWrapper{
					Resolver: resolver,
				},
			},
			Logger: logger,
		},
	}
}

// resolverSystem is the system resolver.
type resolverSystem struct {
	testableTimeout    time.Duration
	testableLookupHost func(ctx context.Context, domain string) ([]string, error)
}

var _ Resolver = &resolverSystem{}

func (r *resolverSystem) LookupHost(ctx context.Context, hostname string) ([]string, error) {
	// This code forces adding a shorter timeout to the domain name
	// resolutions when using the system resolver. We have seen cases
	// in which such a timeout becomes too large. One such case is
	// described in https://github.com/ooni/probe/issues/1726.
	addrsch, errch := make(chan []string, 1), make(chan error, 1)
	ctx, cancel := context.WithTimeout(ctx, r.timeout())
	defer cancel()
	go func() {
		addrs, err := r.lookupHost()(ctx, hostname)
		if err != nil {
			errch <- err
			return
		}
		addrsch <- addrs
	}()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case addrs := <-addrsch:
		return addrs, nil
	case err := <-errch:
		return nil, err
	}
}

func (r *resolverSystem) timeout() time.Duration {
	if r.testableTimeout > 0 {
		return r.testableTimeout
	}
	return 15 * time.Second
}

func (r *resolverSystem) lookupHost() func(ctx context.Context, domain string) ([]string, error) {
	if r.testableLookupHost != nil {
		return r.testableLookupHost
	}
	return net.DefaultResolver.LookupHost
}

func (r *resolverSystem) Network() string {
	return "system"
}

func (r *resolverSystem) Address() string {
	return ""
}

func (r *resolverSystem) CloseIdleConnections() {
	// nothing to do
}

func (r *resolverSystem) LookupHostWithoutRetry(
	ctx context.Context, domain string, qtype uint16) ([]string, error) {
	return nil, ErrNoDNSTransport
}

func (r *resolverSystem) LookupHTTPSSvcWithoutRetry(
	ctx context.Context, domain string) (HTTPSSvc, error) {
	return nil, ErrNoDNSTransport
}

// resolverLogger is a resolver that emits events
type resolverLogger struct {
	Resolver
	Logger Logger
}

var _ Resolver = &resolverLogger{}

func (r *resolverLogger) LookupHost(ctx context.Context, hostname string) ([]string, error) {
	prefix := fmt.Sprintf("resolve[A,AAAA] %s with %s (%s)", hostname, r.Network(), r.Address())
	r.Logger.Debugf("%s...", prefix)
	start := time.Now()
	addrs, err := r.Resolver.LookupHost(ctx, hostname)
	elapsed := time.Since(start)
	if err != nil {
		r.Logger.Debugf("%s... %s in %s", prefix, err, elapsed)
		return nil, err
	}
	r.Logger.Debugf("%s... %+v in %s", prefix, addrs, elapsed)
	return addrs, nil
}

func (r *resolverLogger) LookupHostWithoutRetry(
	ctx context.Context, domain string, qtype uint16) ([]string, error) {
	qtypename := dns.TypeToString[qtype]
	prefix := fmt.Sprintf("resolve[%s] %s with %s (%s)", qtypename, domain, r.Network(), r.Address())
	r.Logger.Debugf("%s...", prefix)
	start := time.Now()
	addrs, err := r.Resolver.LookupHostWithoutRetry(ctx, domain, qtype)
	elapsed := time.Since(start)
	if err != nil {
		r.Logger.Debugf("%s... %s in %s", prefix, err, elapsed)
		return nil, err
	}
	r.Logger.Debugf("%s... %+v in %s", prefix, addrs, elapsed)
	return addrs, nil
}

func (r *resolverLogger) LookupHTTPSSvcWithoutRetry(
	ctx context.Context, domain string) (HTTPSSvc, error) {
	prefix := fmt.Sprintf("resolve[HTTPS] %s with %s (%s)", domain, r.Network(), r.Address())
	r.Logger.Debugf("%s...", prefix)
	start := time.Now()
	https, err := r.Resolver.LookupHTTPSSvcWithoutRetry(ctx, domain)
	elapsed := time.Since(start)
	if err != nil {
		r.Logger.Debugf("%s... %s in %s", prefix, err, elapsed)
		return nil, err
	}
	alpn := https.ALPN()
	a := https.IPv4Hint()
	aaaa := https.IPv6Hint()
	r.Logger.Debugf("%s... %+v %+v %+v in %s", prefix, alpn, a, aaaa, elapsed)
	return https, nil
}

// resolverIDNA supports resolving Internationalized Domain Names.
//
// See RFC3492 for more information.
type resolverIDNA struct {
	Resolver
}

func (r *resolverIDNA) LookupHost(ctx context.Context, hostname string) ([]string, error) {
	host, err := idna.ToASCII(hostname)
	if err != nil {
		return nil, err
	}
	return r.Resolver.LookupHost(ctx, host)
}

func (r *resolverIDNA) LookupHostWithoutRetry(
	ctx context.Context, domain string, qtype uint16) ([]string, error) {
	host, err := idna.ToASCII(domain)
	if err != nil {
		return nil, err
	}
	return r.Resolver.LookupHostWithoutRetry(ctx, host, qtype)
}

func (r *resolverIDNA) LookupHTTPSWithoutRetry(
	ctx context.Context, domain string) (HTTPSSvc, error) {
	host, err := idna.ToASCII(domain)
	if err != nil {
		return nil, err
	}
	return r.Resolver.LookupHTTPSSvcWithoutRetry(ctx, host)
}

// resolverShortCircuitIPAddr recognizes when the input hostname is an
// IP address and returns it immediately to the caller.
type resolverShortCircuitIPAddr struct {
	Resolver
}

func (r *resolverShortCircuitIPAddr) LookupHost(ctx context.Context, hostname string) ([]string, error) {
	if net.ParseIP(hostname) != nil {
		return []string{hostname}, nil
	}
	return r.Resolver.LookupHost(ctx, hostname)
}

// ErrNoResolver indicates you are using a dialer without a resolver.
var ErrNoResolver = errors.New("no configured resolver")

// nullResolver is a resolver that is not capable of resolving
// domain names to IP addresses and always returns ErrNoResolver.
type nullResolver struct{}

func (r *nullResolver) LookupHost(ctx context.Context, hostname string) (addrs []string, err error) {
	return nil, ErrNoResolver
}

func (r *nullResolver) Network() string {
	return "null"
}

func (r *nullResolver) Address() string {
	return ""
}

func (r *nullResolver) CloseIdleConnections() {
	// nothing to do
}

func (r *nullResolver) LookupHostWithoutRetry(
	ctx context.Context, domain string, qtype uint16) ([]string, error) {
	return nil, ErrNoDNSTransport
}

func (r *nullResolver) LookupHTTPSSvcWithoutRetry(
	ctx context.Context, domain string) (HTTPSSvc, error) {
	return nil, ErrNoDNSTransport
}

// resolverErrWrapper is a Resolver that knows about wrapping errors.
type resolverErrWrapper struct {
	Resolver
}

var _ Resolver = &resolverErrWrapper{}

func (r *resolverErrWrapper) LookupHost(ctx context.Context, hostname string) ([]string, error) {
	addrs, err := r.Resolver.LookupHost(ctx, hostname)
	if err != nil {
		return nil, errorsx.NewErrWrapper(
			errorsx.ClassifyResolverError, errorsx.ResolveOperation, err)
	}
	return addrs, nil
}

func (r *resolverErrWrapper) LookupHostWithoutRetry(
	ctx context.Context, domain string, qtype uint16) ([]string, error) {
	addrs, err := r.Resolver.LookupHostWithoutRetry(ctx, domain, qtype)
	if err != nil {
		return nil, errorsx.NewErrWrapper(
			errorsx.ClassifyResolverError, errorsx.ResolveOperation, err)
	}
	return addrs, nil
}

func (r *resolverErrWrapper) LookupHTTPSWithoutRetry(
	ctx context.Context, domain string) (HTTPSSvc, error) {
	out, err := r.Resolver.LookupHTTPSSvcWithoutRetry(ctx, domain)
	if err != nil {
		return nil, errorsx.NewErrWrapper(
			errorsx.ClassifyResolverError, errorsx.ResolveOperation, err)
	}
	return out, nil
}
