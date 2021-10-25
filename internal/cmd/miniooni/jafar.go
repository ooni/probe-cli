package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/netxlite/quicx"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// JafarSpec is the specification for Jafar.
type JafarSpec struct {
	// Domains contains blocked domains information.
	Domains map[string]string

	// Endpoints contains blocked endpoints information.
	Endpoints map[string]string
}

// MaybeParseJafarSpec will parse the jafar spec and install jafar
// overrides iff the file name is not empty. In case it's not possible
// to read or parse the file, this function will panic. In case the
// file name is empty, this function will do nothing.
// will panic if we cannot parse the spec.
func MaybeParseJafarSpec(file string) {
	if file == "" {
		return
	}
	data, err := os.ReadFile(file)
	runtimex.PanicOnError(err, "cannot read jafar spec")
	var spec JafarSpec
	err = json.Unmarshal(data, &spec)
	runtimex.PanicOnError(err, "cannot parse jafar spec")
	netxlite.TProxy = &spec
}

// endpointBlockingPolicy returns the blocking policy of the endpoint
// represented by the given network (e.g., "tcp") and address
// (e.g., "8.8.8.8:443"). By convention an empty string is returned
// when the endpoint should not be blocked.
func (s *JafarSpec) endpointBlockingPolicy(network, address string) string {
	epnt := fmt.Sprintf("%s/%s", address, network)
	return s.Endpoints[epnt]
}

// domainBlockingPolicy is like endpointBlockingPolicy but for a domain.
func (s *JafarSpec) domainBlockingPolicy(domain string) string {
	return s.Domains[domain]
}

// ListenUDP creates a new quicx.UDPLikeConn conn.
func (s *JafarSpec) ListenUDP(network string, laddr *net.UDPAddr) (quicx.UDPLikeConn, error) {
	conn, err := net.ListenUDP(network, laddr)
	if err != nil {
		return nil, err
	}
	return &jafarUDPLikeConn{UDPLikeConn: conn, spec: s}, nil
}

// jafarUDPLikeConn wraps UDPLikeConn to implement blocking policies.
type jafarUDPLikeConn struct {
	spec *JafarSpec
	quicx.UDPLikeConn
}

func (c *jafarUDPLikeConn) WriteTo(p []byte, addr net.Addr) (int, error) {
	if c.spec.endpointBlockingPolicy(addr.Network(), addr.String()) != "" {
		return len(p), nil
	}
	return c.UDPLikeConn.WriteTo(p, addr)
}

// LookupHost lookups a domain using the stdlib resolver.
func (s *JafarSpec) LookupHost(ctx context.Context, domain string) ([]string, error) {
	switch s.domainBlockingPolicy(domain) {
	case "":
		return net.DefaultResolver.LookupHost(ctx, domain)
	case "nxdomain":
		return nil, netxlite.ErrOODNSNoSuchHost
	case "refused":
		return nil, netxlite.ErrOODNSRefused
	case "no_answer":
		return nil, netxlite.ErrOODNSNoAnswer
	default:
		<-ctx.Done()
		return nil, ctx.Err()
	}
}

// NewTProxyDialer returns a new TProxyDialer.
func (s *JafarSpec) NewTProxyDialer(timeout time.Duration) netxlite.TProxyDialer {
	return &jafarTProxyDialer{
		dialer: &net.Dialer{Timeout: timeout},
		spec:   s,
	}
}

// jafarTProxyDialer is jafar's TProxyDialer.
type jafarTProxyDialer struct {
	dialer *net.Dialer
	spec   *JafarSpec
}

func (d *jafarTProxyDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	if d.spec.endpointBlockingPolicy(network, address) != "" {
		<-ctx.Done()
		return nil, ctx.Err()
	}
	return d.dialer.DialContext(ctx, network, address)
}
