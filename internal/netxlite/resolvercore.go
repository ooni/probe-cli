package netxlite

//
// DNS resolver
//

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/model"
	"golang.org/x/net/idna"
)

// ErrNoDNSTransport is the error returned when you attempt to perform
// a DNS operation that requires a custom DNSTransport (e.g., DNSOverHTTPSTransport)
// but you are using the "system" resolver instead.
var ErrNoDNSTransport = errors.New("operation requires a DNS transport")

// NewStdlibResolver creates a new Resolver by combining WrapResolver
// with an internal "system" resolver type. The list of optional wrappers
// allow to wrap the underlying getaddrinfo transport. Any nil wrapper
// will be silently ignored by the code that performs the wrapping.
func NewStdlibResolver(logger model.DebugLogger, wrappers ...model.DNSTransportWrapper) model.Resolver {
	return WrapResolver(logger, NewUnwrappedStdlibResolver(wrappers...))
}

// NewParallelDNSOverHTTPSResolver creates a new DNS over HTTPS resolver
// that uses the standard library for all operations. This function constructs
// all the building blocks and calls WrapResolver on the returned resolver.
func NewParallelDNSOverHTTPSResolver(logger model.DebugLogger, URL string) model.Resolver {
	client := &http.Client{Transport: NewHTTPTransportStdlib(logger)}
	txp := WrapDNSTransport(NewUnwrappedDNSOverHTTPSTransport(client, URL))
	return WrapResolver(logger, NewUnwrappedParallelResolver(txp))
}

// NewUnwrappedStdlibResolver returns a new, unwrapped resolver using the standard
// library (i.e., getaddrinfo if possible and &net.Resolver{} otherwise). As the name
// implies, this function returns an unwrapped resolver.
func NewUnwrappedStdlibResolver(wrappers ...model.DNSTransportWrapper) model.Resolver {
	return &resolverSystem{
		t: WrapDNSTransport(&dnsOverGetaddrinfoTransport{}, wrappers...),
	}
}

// NewSerialUDPResolver creates a new Resolver using DNS-over-UDP
// that performs serial A/AAAA lookups during LookupHost.
//
// Deprecated: use NewParallelResolverUDP.
//
// Arguments:
//
// - logger is the logger to use
//
// - dialer is the dialer to create and connect UDP conns
//
// - address is the server address (e.g., 1.1.1.1:53)
//
// - wrappers is the optional list of wrappers to wrap the underlying
// transport.  Any nil wrapper will be silently ignored.
func NewSerialUDPResolver(logger model.DebugLogger, dialer model.Dialer,
	address string, wrappers ...model.DNSTransportWrapper) model.Resolver {
	return WrapResolver(logger, NewUnwrappedSerialResolver(
		WrapDNSTransport(NewUnwrappedDNSOverUDPTransport(dialer, address), wrappers...),
	))
}

// NewParallelUDPResolver creates a new Resolver using DNS-over-UDP
// that performs parallel A/AAAA lookups during LookupHost.
//
// Arguments:
//
// - logger is the logger to use
//
// - dialer is the dialer to create and connect UDP conns
//
// - address is the server address (e.g., 1.1.1.1:53)
//
// - wrappers is the optional list of wrappers to wrap the underlying
// transport.  Any nil wrapper will be silently ignored.
func NewParallelUDPResolver(logger model.DebugLogger, dialer model.Dialer,
	address string, wrappers ...model.DNSTransportWrapper) model.Resolver {
	return WrapResolver(logger, NewUnwrappedParallelResolver(
		WrapDNSTransport(NewUnwrappedDNSOverUDPTransport(dialer, address), wrappers...),
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
	t model.DNSTransport
}

var _ model.Resolver = &resolverSystem{}

func (r *resolverSystem) LookupHost(ctx context.Context, hostname string) ([]string, error) {
	encoder := &DNSEncoderMiekg{}
	query := encoder.Encode(hostname, dns.TypeANY, false)
	resp, err := r.t.RoundTrip(ctx, query)
	if err != nil {
		return nil, err
	}
	return resp.DecodeLookupHost()
}

func (r *resolverSystem) Network() string {
	return r.t.Network()
}

func (r *resolverSystem) Address() string {
	return r.t.Address()
}

func (r *resolverSystem) CloseIdleConnections() {
	r.t.CloseIdleConnections()
}

func (r *resolverSystem) LookupHTTPS(
	ctx context.Context, domain string) (*model.HTTPSSvc, error) {
	return nil, ErrNoDNSTransport
}

func (r *resolverSystem) LookupNS(
	ctx context.Context, domain string) ([]*net.NS, error) {
	return nil, ErrNoDNSTransport
}

// resolverLogger is a resolver that emits events
type resolverLogger struct {
	Resolver model.Resolver
	Logger   model.DebugLogger
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

func (r *resolverLogger) Address() string {
	return r.Resolver.Address()
}

func (r *resolverLogger) Network() string {
	return r.Resolver.Network()
}

func (r *resolverLogger) CloseIdleConnections() {
	r.Resolver.CloseIdleConnections()
}

func (r *resolverLogger) LookupNS(
	ctx context.Context, domain string) ([]*net.NS, error) {
	prefix := fmt.Sprintf("resolve[NS] %s with %s (%s)", domain, r.Network(), r.Address())
	r.Logger.Debugf("%s...", prefix)
	start := time.Now()
	ns, err := r.Resolver.LookupNS(ctx, domain)
	elapsed := time.Since(start)
	if err != nil {
		r.Logger.Debugf("%s... %s in %s", prefix, err, elapsed)
		return nil, err
	}
	r.Logger.Debugf("%s... %+v in %s", prefix, ns, elapsed)
	return ns, nil
}

// resolverIDNA supports resolving Internationalized Domain Names.
//
// See RFC3492 for more information.
type resolverIDNA struct {
	Resolver model.Resolver
}

var _ model.Resolver = &resolverIDNA{}

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

func (r *resolverIDNA) Network() string {
	return r.Resolver.Network()
}

func (r *resolverIDNA) Address() string {
	return r.Resolver.Address()
}

func (r *resolverIDNA) CloseIdleConnections() {
	r.Resolver.CloseIdleConnections()
}

func (r *resolverIDNA) LookupNS(
	ctx context.Context, domain string) ([]*net.NS, error) {
	host, err := idna.ToASCII(domain)
	if err != nil {
		return nil, err
	}
	return r.Resolver.LookupNS(ctx, host)
}

// resolverShortCircuitIPAddr recognizes when the input hostname is an
// IP address and returns it immediately to the caller.
type resolverShortCircuitIPAddr struct {
	Resolver model.Resolver
}

var _ model.Resolver = &resolverShortCircuitIPAddr{}

func (r *resolverShortCircuitIPAddr) LookupHost(ctx context.Context, hostname string) ([]string, error) {
	if net.ParseIP(hostname) != nil {
		return []string{hostname}, nil
	}
	return r.Resolver.LookupHost(ctx, hostname)
}

func (r *resolverShortCircuitIPAddr) LookupHTTPS(ctx context.Context, hostname string) (*model.HTTPSSvc, error) {
	if net.ParseIP(hostname) != nil {
		https := &model.HTTPSSvc{}
		if isIPv6(hostname) {
			https.IPv6 = append(https.IPv6, hostname)
		} else {
			https.IPv4 = append(https.IPv4, hostname)
		}
		return https, nil
	}
	return r.Resolver.LookupHTTPS(ctx, hostname)
}

func (r *resolverShortCircuitIPAddr) Network() string {
	return r.Resolver.Network()
}

func (r *resolverShortCircuitIPAddr) Address() string {
	return r.Resolver.Address()
}

func (r *resolverShortCircuitIPAddr) CloseIdleConnections() {
	r.Resolver.CloseIdleConnections()
}

// ErrDNSIPAddress indicates that you passed an IP address to a DNS
// function that only works with domain names.
var ErrDNSIPAddress = errors.New("ooresolver: expected domain, found IP address")

func (r *resolverShortCircuitIPAddr) LookupNS(
	ctx context.Context, hostname string) ([]*net.NS, error) {
	if net.ParseIP(hostname) != nil {
		return nil, ErrDNSIPAddress
	}
	return r.Resolver.LookupNS(ctx, hostname)
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

// NullResolver is a resolver that is not capable of resolving
// domain names to IP addresses and always returns ErrNoResolver.
type NullResolver struct{}

func (r *NullResolver) LookupHost(ctx context.Context, hostname string) (addrs []string, err error) {
	return nil, ErrNoResolver
}

func (r *NullResolver) Network() string {
	return "null"
}

func (r *NullResolver) Address() string {
	return ""
}

func (r *NullResolver) CloseIdleConnections() {
	// nothing to do
}

func (r *NullResolver) LookupHTTPS(
	ctx context.Context, domain string) (*model.HTTPSSvc, error) {
	return nil, ErrNoResolver
}

func (r *NullResolver) LookupNS(
	ctx context.Context, domain string) ([]*net.NS, error) {
	return nil, ErrNoResolver
}

// resolverErrWrapper is a Resolver that knows about wrapping errors.
type resolverErrWrapper struct {
	Resolver model.Resolver
}

var _ model.Resolver = &resolverErrWrapper{}

func (r *resolverErrWrapper) LookupHost(ctx context.Context, hostname string) ([]string, error) {
	addrs, err := r.Resolver.LookupHost(ctx, hostname)
	if err != nil {
		return nil, NewErrWrapper(ClassifyResolverError, ResolveOperation, err)
	}
	return addrs, nil
}

func (r *resolverErrWrapper) LookupHTTPS(
	ctx context.Context, domain string) (*model.HTTPSSvc, error) {
	out, err := r.Resolver.LookupHTTPS(ctx, domain)
	if err != nil {
		return nil, NewErrWrapper(ClassifyResolverError, ResolveOperation, err)
	}
	return out, nil
}

func (r *resolverErrWrapper) Network() string {
	return r.Resolver.Network()
}

func (r *resolverErrWrapper) Address() string {
	return r.Resolver.Address()
}

func (r *resolverErrWrapper) CloseIdleConnections() {
	r.Resolver.CloseIdleConnections()
}

func (r *resolverErrWrapper) LookupNS(
	ctx context.Context, domain string) ([]*net.NS, error) {
	out, err := r.Resolver.LookupNS(ctx, domain)
	if err != nil {
		return nil, NewErrWrapper(ClassifyResolverError, ResolveOperation, err)
	}
	return out, nil
}
