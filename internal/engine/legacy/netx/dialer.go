// Package netx contains OONI's net extensions.
package netx

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net"
	"os"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/legacy/netx/handlers"
	"github.com/ooni/probe-cli/v3/internal/engine/legacy/netx/modelx"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/dialer"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/tlsdialer"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// Dialer performs measurements while dialing.
type Dialer struct {
	Beginning time.Time
	Handler   modelx.Handler
	Resolver  modelx.DNSResolver
	TLSConfig *tls.Config
}

func newDialer(beginning time.Time, handler modelx.Handler) *Dialer {
	return &Dialer{
		Beginning: beginning,
		Handler:   handler,
		Resolver:  newResolverSystem(),
		TLSConfig: new(tls.Config),
	}
}

// NewDialer creates a new Dialer instance.
func NewDialer() *Dialer {
	return newDialer(time.Now(), handlers.NoHandler)
}

// Dial creates a TCP or UDP connection. See net.Dial docs.
func (d *Dialer) Dial(network, address string) (net.Conn, error) {
	return d.DialContext(context.Background(), network, address)
}

func maybeWithMeasurementRoot(
	ctx context.Context, beginning time.Time, handler modelx.Handler,
) context.Context {
	if modelx.ContextMeasurementRoot(ctx) != nil {
		return ctx
	}
	return modelx.WithMeasurementRoot(ctx, &modelx.MeasurementRoot{
		Beginning: beginning,
		Handler:   handler,
	})
}

// newDNSDialer creates a new DNS dialer using the following chain:
//
// - DNSDialer (topmost)
// - EmitterDialer
// - ErrorWrapperDialer
// - ByteCountingDialer
// - dialer.Default
//
// If you have others needs, manually build the chain you need.
func newDNSDialer(resolver dialer.Resolver) dialer.Dialer {
	// Implementation note: we're wrapping the result of dialer.New
	// on the outside, while previously we were puttting the
	// EmitterDialer before the DNSDialer (see the above comment).
	//
	// Yet, this is fine because the only experiment which is
	// using this code is tor, for which it doesn't matter.
	//
	// Also (and I am always scared to write this kind of
	// comments), we should rewrite tor soon.
	return &EmitterDialer{dialer.New(&dialer.Config{
		ContextByteCounting: true,
	}, resolver)}
}

// DialContext is like Dial but the context allows to interrupt a
// pending connection attempt at any time.
func (d *Dialer) DialContext(
	ctx context.Context, network, address string,
) (conn net.Conn, err error) {
	ctx = maybeWithMeasurementRoot(ctx, d.Beginning, d.Handler)
	return newDNSDialer(d.Resolver).DialContext(ctx, network, address)
}

// DialTLS is like Dial, but creates TLS connections.
func (d *Dialer) DialTLS(network, address string) (net.Conn, error) {
	return d.DialTLSContext(context.Background(), network, address)
}

// newTLSDialer creates a new TLSDialer using:
//
// - EmitterTLSHandshaker (topmost)
// - ErrorWrapperTLSHandshaker
// - TimeoutTLSHandshaker
// - SystemTLSHandshaker
//
// If you have others needs, manually build the chain you need.
func newTLSDialer(d dialer.Dialer, config *tls.Config) *netxlite.TLSDialer {
	return &netxlite.TLSDialer{
		Config: config,
		Dialer: d,
		TLSHandshaker: tlsdialer.EmitterTLSHandshaker{
			TLSHandshaker: tlsdialer.ErrorWrapperTLSHandshaker{
				TLSHandshaker: &netxlite.TLSHandshakerConfigurable{},
			},
		},
	}
}

// DialTLSContext is like DialTLS, but with context
func (d *Dialer) DialTLSContext(
	ctx context.Context, network, address string,
) (net.Conn, error) {
	ctx = maybeWithMeasurementRoot(ctx, d.Beginning, d.Handler)
	return newTLSDialer(
		newDNSDialer(d.Resolver),
		d.TLSConfig,
	).DialTLSContext(ctx, network, address)
}

// SetCABundle configures the dialer to use a specific CA bundle. This
// function is not goroutine safe. Make sure you call it before starting
// to use this specific dialer.
func (d *Dialer) SetCABundle(path string) error {
	cert, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM(cert) {
		return errors.New("AppendCertsFromPEM failed")
	}
	d.TLSConfig.RootCAs = pool
	return nil
}

// ForceSpecificSNI forces using a specific SNI.
func (d *Dialer) ForceSpecificSNI(sni string) error {
	d.TLSConfig.ServerName = sni
	return nil
}

// ForceSkipVerify forces to skip certificate verification
func (d *Dialer) ForceSkipVerify() error {
	d.TLSConfig.InsecureSkipVerify = true
	return nil
}

// ConfigureDNS configures the DNS resolver. The network argument
// selects the type of resolver. The address argument indicates the
// resolver address and depends on the network.
//
// This functionality is not goroutine safe. You should only change
// the DNS settings before starting to use the Dialer.
//
// The following is a list of all the possible network values:
//
// - "": behaves exactly like "system"
//
// - "system": this indicates that Go should use the system resolver
// and prevents us from seeing any DNS packet. The value of the
// address parameter is ignored when using "system". If you do
// not ConfigureDNS, this is the default resolver used.
//
// - "udp": indicates that we should send queries using UDP. In this
// case the address is a host, port UDP endpoint.
//
// - "tcp": like "udp" but we use TCP.
//
// - "dot": we use DNS over TLS (DoT). In this case the address is
// the domain name of the DoT server.
//
// - "doh": we use DNS over HTTPS (DoH). In this case the address is
// the URL of the DoH server.
//
// For example:
//
//   d.ConfigureDNS("system", "")
//   d.ConfigureDNS("udp", "8.8.8.8:53")
//   d.ConfigureDNS("tcp", "8.8.8.8:53")
//   d.ConfigureDNS("dot", "dns.quad9.net")
//   d.ConfigureDNS("doh", "https://cloudflare-dns.com/dns-query")
func (d *Dialer) ConfigureDNS(network, address string) error {
	r, err := newResolver(d.Beginning, d.Handler, network, address)
	if err == nil {
		d.Resolver = r
	}
	return err
}

// SetResolver is a more flexible way of configuring a resolver
// that should perhaps be used instead of ConfigureDNS.
func (d *Dialer) SetResolver(r modelx.DNSResolver) {
	d.Resolver = r
}
