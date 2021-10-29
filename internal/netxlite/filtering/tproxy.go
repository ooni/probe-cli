package filtering

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"time"

	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/netxlite/quicx"
)

// TProxyPolicy is a policy for the transparent proxy.
type TProxyPolicy string

const (
	// TProxyPolicyDropSYN only applies to outgoing TCP connections and
	// causes the TCP segment to be dropped.
	TProxyPolicyDropSYN = TProxyPolicy("drop-syn")

	// TProxyPolicyDropData applies to existing TCP/UDP connections
	// and causes outgoing data to be dropped.
	TProxyPolicyDropData = TProxyPolicy("drop-data")

	// TProxyPolicyHijackDNS only applies to UDP connections and causes
	// the destination address to become the one of the local DNS
	// server, which will apply DNSActions to incoming queries.
	TProxyPolicyHijackDNS = TProxyPolicy("hijack-dns")

	// TProxyPolicyHijackTLS only applies to TCP connections and causes
	// the destination address to become the one of the local TLS
	// server, which will apply DNSActions to ClientHelloes.
	TProxyPolicyHijackTLS = TProxyPolicy("hijack-tls")
)

// TProxyConfig contains configuration for TProxy.
type TProxyConfig struct {
	// Domains contains rules for filtering the lookup of domains.
	Domains map[string]DNSAction

	// Endpoints contains rules for filtering TCP/UDP endpoints.
	Endpoints map[string]TProxyPolicy

	// SNIs contains rules for filtering TLS SNIs.
	SNIs map[string]TLSAction
}

// NewTProxyConfig reads the TProxyConfig from the given file.
func NewTProxyConfig(file string) (*TProxyConfig, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, err
	}
	var config TProxyConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// TProxy implements netxlite.TProxable to implement filtering.
type TProxy struct {
	// config contains settings for TProxy.
	config *TProxyConfig

	// dnsClient is the DNS client we'll internally use.
	dnsClient netxlite.Resolver

	// dnsListener is the DNS listener.
	dnsListener DNSListener

	// logger is the underlying logger to use.
	logger Logger

	// tlsListener is the TLS listener.
	tlsListener net.Listener
}

// NewTProxy creates a new TProxy instance.
func NewTProxy(config *TProxyConfig, logger Logger) (*TProxy, error) {
	var err error
	p := &TProxy{config: config, logger: logger}
	dnsProxy := &DNSProxy{
		OnQuery: p.onQuery,
	}
	p.dnsListener, err = dnsProxy.Start("127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	dialer := netxlite.NewDialerWithoutResolver(logger)
	p.dnsClient = netxlite.NewResolverUDP(logger, dialer, p.dnsListener.LocalAddr().String())
	tlsProxy := &TLSProxy{
		OnIncomingSNI: p.onIncomingSNI,
	}
	p.tlsListener, err = tlsProxy.Start("127.0.0.1:0")
	if err != nil {
		p.dnsListener.Close()
		return nil, err
	}
	p.canonicalizeDNS()
	return p, nil
}

// canonicalizeDNS ensures all DNS names are canonicalized.
func (p *TProxy) canonicalizeDNS() {
	domains := make(map[string]DNSAction)
	for domain, policy := range p.config.Domains {
		domains[dns.CanonicalName(domain)] = policy
	}
	p.config.Domains = domains
}

// Close closes the resources used by a TProxy.
func (p *TProxy) Close() error {
	p.dnsListener.Close()
	return nil
}

// ListenUDP implements netxlite.TProxy.ListenUDP.
func (p *TProxy) ListenUDP(network string, laddr *net.UDPAddr) (quicx.UDPLikeConn, error) {
	pconn, err := net.ListenUDP(network, laddr)
	if err != nil {
		return nil, err
	}
	return &tProxyUDPLikeConn{UDPLikeConn: pconn, proxy: p}, nil
}

// LookupHost implements netxlite.TProxy.LookupHost.
func (p *TProxy) LookupHost(ctx context.Context, domain string) ([]string, error) {
	return p.dnsClient.LookupHost(ctx, domain)
}

// NewTProxyDialer implements netxlite.TProxy.NewTProxyDialer.
func (p *TProxy) NewTProxyDialer(timeout time.Duration) netxlite.TProxyDialer {
	return &tProxyDialer{
		dialer: &net.Dialer{Timeout: timeout},
		proxy:  p,
	}
}

// tProxyUDPLikeConn is a TProxy-aware UDPLikeConn.
type tProxyUDPLikeConn struct {
	// UDPLikeConn is the underlying conn type.
	quicx.UDPLikeConn

	// proxy refers to the TProxy.
	proxy *TProxy
}

// ErrCannotApplyTProxyPolicy means that the policy cannot be applied.
var ErrCannotApplyTProxyPolicy = errors.New("tproxy: cannot apply policy")

// WriteTo implements UDPLikeConn.WriteTo. This function will
// apply the proper tproxy policies, if required.
func (c *tProxyUDPLikeConn) WriteTo(pkt []byte, addr net.Addr) (int, error) {
	endpoint := fmt.Sprintf("%s/%s", addr.String(), addr.Network())
	policy := c.proxy.config.Endpoints[endpoint]
	switch policy {
	case TProxyPolicyDropData:
		// If we're asked to drop this packet, we'll just pretend we've
		// emitted it on the wire without emitting it.
		return len(pkt), nil
	default:
		c.proxy.logger.Infof("tproxy: WriteTo: %s => %s", endpoint, policy)
		return c.UDPLikeConn.WriteTo(pkt, addr)
	}
}

// tProxyDialer is a TProxy-aware Dialer.
type tProxyDialer struct {
	// dialer is the underlying network dialer.
	dialer *net.Dialer

	// proxy refers to the TProxy.
	proxy *TProxy
}

// DialContext behaves like net.Dialer.DialContext. This function will
// apply the proper tproxy policies, if required.
func (d *tProxyDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	endpoint := fmt.Sprintf("%s/%s", address, network)
	policy := d.proxy.config.Endpoints[endpoint]
	switch policy {
	case TProxyPolicyDropSYN:
		// If we're asked to drop SYN segments, then we will just not
		// dial at all and wait for the context to expire. To avoid
		// blocking the dialing operation forever, we'll ensure that
		// there is a large timeout after which we give up.
		switch network {
		case "tcp", "tcp4", "tcp6":
			d.proxy.logger.Infof("tproxy: DialContext: %s/%s => %s", address, network, policy)
			var cancel context.CancelFunc
			const timeout = 70 * time.Second
			ctx, cancel = context.WithTimeout(ctx, timeout)
			defer cancel()
			<-ctx.Done()
			return nil, errors.New("i/o timeout")
		default:
			return nil, ErrCannotApplyTProxyPolicy
		}
	case TProxyPolicyHijackDNS:
		// If we're asked to hijack the DNS, we'll simply replace
		// the destination address with the local DNS server's one
		switch network {
		case "udp", "udp4", "udp6":
			d.proxy.logger.Infof("tproxy: DialContext: %s/%s => %s", address, network, policy)
			address = d.proxy.dnsListener.LocalAddr().String()
		default:
			return nil, ErrCannotApplyTProxyPolicy
		}
	case TProxyPolicyHijackTLS:
		// If we're asked to hijack TLS, we'll simply replace
		// the destination address with the local TLS's one
		switch network {
		case "tcp", "tcp4", "tcp6":
			d.proxy.logger.Infof("tproxy: DialContext: %s/%s => %s", address, network, policy)
			address = d.proxy.tlsListener.Addr().String()
		default:
			return nil, ErrCannotApplyTProxyPolicy
		}
	default:
		// nothing
	}
	conn, err := d.dialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}
	return &tProxyConn{Conn: conn, proxy: d.proxy}, nil
}

// onQuery is called for filtering outgoing DNS queries.
func (p *TProxy) onQuery(domain string) DNSAction {
	policy := p.config.Domains[domain]
	if policy == "" {
		policy = DNSActionPass
	}
	p.logger.Infof("tproxy: DNS: %s => %s", domain, policy)
	return policy
}

// onIncomingSNI is called for filtering SNI values.
func (p *TProxy) onIncomingSNI(sni string) TLSAction {
	policy := p.config.SNIs[sni]
	if policy == "" {
		policy = TLSActionPass
	}
	p.logger.Infof("tproxy: TLS: %s => %s", sni, policy)
	return policy
}

// tProxyConn is a TProxy-aware net.Conn.
type tProxyConn struct {
	// Conn is the underlying conn.
	net.Conn

	// proxy refers to the TProxy.
	proxy *TProxy
}

// Write implements Conn.Write. This function will apply
// the proper tproxy policies, if required.
func (c *tProxyConn) Write(b []byte) (int, error) {
	addr := c.Conn.RemoteAddr()
	endpoint := fmt.Sprintf("%s/%s", addr.String(), addr.Network())
	policy := c.proxy.config.Endpoints[endpoint]
	switch policy {
	case TProxyPolicyDropData:
		// If we're asked to drop this packet, we'll just pretend we've
		// emitted it on the wire without emitting it.
		c.proxy.logger.Infof("tproxy: Write: %s => %s", endpoint, policy)
		return len(b), nil
	default:
		return c.Conn.Write(b)
	}
}
