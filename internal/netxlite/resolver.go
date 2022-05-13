package netxlite

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"golang.org/x/net/idna"
)

// ErrNoDNSTransport is the error returned when you attempt to perform
// a DNS operation that requires a custom DNSTransport (e.g., DNSOverHTTPS)
// but you are using the "system" resolver instead.
var ErrNoDNSTransport = errors.New("operation requires a DNS transport")

// NewResolverStdlib creates a new Resolver by combining WrapResolver
// with an internal "system" resolver type.
func NewResolverStdlib(logger model.DebugLogger) model.Resolver {
	return WrapResolver(logger, &resolverSystem{})
}

// NewResolverUDP creates a new Resolver using DNS-over-UDP.
//
// Arguments:
//
// - logger is the logger to use
//
// - dialer is the dialer to create and connect UDP conns
//
// - address is the server address (e.g., 1.1.1.1:53)
func NewResolverUDP(logger model.DebugLogger, dialer model.Dialer, address string) model.Resolver {
	return WrapResolver(logger, NewSerialResolver(
		NewDNSOverUDP(dialer, address),
	))
}

// WrapResolver creates a new resolver that wraps an
// existing resolver to add these properties:
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
//
// This is a low-level factory. Use only if out of alternatives.
func WrapResolver(logger model.DebugLogger, resolver model.Resolver) model.Resolver {
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

var _ model.Resolver = &resolverSystem{}

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
	return TProxy.LookupHost
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

func (r *resolverSystem) LookupHTTPS(
	ctx context.Context, domain string) (*model.HTTPSSvc, error) {
	return nil, ErrNoDNSTransport
}

// resolverLogger is a resolver that emits events
type resolverLogger struct {
	model.Resolver
	Logger model.DebugLogger
}

var _ model.Resolver = &resolverLogger{}

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

func (r *resolverLogger) LookupHTTPS(
	ctx context.Context, domain string) (*model.HTTPSSvc, error) {
	prefix := fmt.Sprintf("resolve[HTTPS] %s with %s (%s)", domain, r.Network(), r.Address())
	r.Logger.Debugf("%s...", prefix)
	start := time.Now()
	https, err := r.Resolver.LookupHTTPS(ctx, domain)
	elapsed := time.Since(start)
	if err != nil {
		r.Logger.Debugf("%s... %s in %s", prefix, err, elapsed)
		return nil, err
	}
	alpn := https.ALPN
	a := https.IPv4
	aaaa := https.IPv6
	r.Logger.Debugf("%s... %+v %+v %+v in %s", prefix, alpn, a, aaaa, elapsed)
	return https, nil
}

// resolverIDNA supports resolving Internationalized Domain Names.
//
// See RFC3492 for more information.
type resolverIDNA struct {
	model.Resolver
}

func (r *resolverIDNA) LookupHost(ctx context.Context, hostname string) ([]string, error) {
	host, err := idna.ToASCII(hostname)
	if err != nil {
		return nil, err
	}
	return r.Resolver.LookupHost(ctx, host)
}

func (r *resolverIDNA) LookupHTTPS(
	ctx context.Context, domain string) (*model.HTTPSSvc, error) {
	host, err := idna.ToASCII(domain)
	if err != nil {
		return nil, err
	}
	return r.Resolver.LookupHTTPS(ctx, host)
}

// resolverShortCircuitIPAddr recognizes when the input hostname is an
// IP address and returns it immediately to the caller.
type resolverShortCircuitIPAddr struct {
	model.Resolver
}

func (r *resolverShortCircuitIPAddr) LookupHost(ctx context.Context, hostname string) ([]string, error) {
	if net.ParseIP(hostname) != nil {
		return []string{hostname}, nil
	}
	return r.Resolver.LookupHost(ctx, hostname)
}

// IsIPv6 returns true if the given candidate is a valid IP address
// representation and such representation is IPv6.
func IsIPv6(candidate string) (bool, error) {
	if net.ParseIP(candidate) == nil {
		return false, ErrInvalidIP
	}
	return isIPv6(candidate), nil
}

// isIPv6 returns true if the given IP address is IPv6.
func isIPv6(candidate string) bool {
	// This check for identifying IPv6 is discussed
	// at https://stackoverflow.com/questions/22751035
	// and seems good-enough for our purposes.
	return strings.Contains(candidate, ":")
}

// ErrNoResolver is the type of error returned by "without resolver"
// dialer when asked to dial for and endpoint containing a domain name,
// since they can only dial for endpoints containing IP addresses.
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

func (r *nullResolver) LookupHTTPS(
	ctx context.Context, domain string) (*model.HTTPSSvc, error) {
	return nil, ErrNoResolver
}

// resolverErrWrapper is a Resolver that knows about wrapping errors.
type resolverErrWrapper struct {
	model.Resolver
}

var _ model.Resolver = &resolverErrWrapper{}

func (r *resolverErrWrapper) LookupHost(ctx context.Context, hostname string) ([]string, error) {
	addrs, err := r.Resolver.LookupHost(ctx, hostname)
	if err != nil {
		return nil, NewErrWrapper(classifyResolverError, ResolveOperation, err)
	}
	return addrs, nil
}

func (r *resolverErrWrapper) LookupHTTPS(
	ctx context.Context, domain string) (*model.HTTPSSvc, error) {
	out, err := r.Resolver.LookupHTTPS(ctx, domain)
	if err != nil {
		return nil, NewErrWrapper(classifyResolverError, ResolveOperation, err)
	}
	return out, nil
}
