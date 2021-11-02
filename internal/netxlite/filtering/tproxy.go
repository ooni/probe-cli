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
	// TProxyPolicyTCPDropSYN only applies to outgoing TCP connections and
	// causes the TCP segment to be dropped.
	TProxyPolicyTCPDropSYN = TProxyPolicy("tcp-drop-syn")

	// TProxyPolicyTCPRejectSYN only applies to outgoing TCP connections and
	// causes the TCP segment to be replied to with RST.
	TProxyPolicyTCPRejectSYN = TProxyPolicy("tcp-reject-syn")

	// TProxyPolicyDropData applies to existing TCP/UDP connections
	// and causes outgoing data to be dropped.
	TProxyPolicyDropData = TProxyPolicy("drop-data")

	// TProxyPolicyHijackDNS causes the dialer to replace the target
	// address with the address of the local censored resolver.
	TProxyPolicyHijackDNS = TProxyPolicy("hijack-dns")

	// TProxyPolicyHijackTLS causes the dialer to replace the target
	// address with the address of the local censored TLS server.
	TProxyPolicyHijackTLS = TProxyPolicy("hijack-tls")

	// TProxyPolicyHijackHTTP causes the dialer to replace the target
	// address with the address of the local censored HTTP server.
	TProxyPolicyHijackHTTP = TProxyPolicy("hijack-http")
)

// TProxyConfig contains configuration for TProxy.
type TProxyConfig struct {
	// Domains contains rules for filtering the lookup of domains. Note
	// that the map MUST contain FQDNs. That is, you need to append
	// a final dot to the domain name (e.g., `example.com.`).  If you
	// use the NewTProxyConfig factory, you don't need to worry about this
	// issue, because the factory will canonicalize non-canonical
	// entries. Otherwise, you can explicitly call the CanonicalizeDNS
	// method _before_ using the TProxy.
	Domains map[string]DNSAction

	// Endpoints contains rules for filtering TCP/UDP endpoints.
	Endpoints map[string]TProxyPolicy

	// SNIs contains rules for filtering TLS SNIs.
	SNIs map[string]TLSAction

	// Hosts contains rules for filtering by HTTP host.
	Hosts map[string]HTTPAction
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
	config.CanonicalizeDNS()
	return &config, nil
}

// CanonicalizeDNS ensures all DNS names are canonicalized. This method
// modifies the TProxyConfig structure in place.
func (c *TProxyConfig) CanonicalizeDNS() {
	domains := make(map[string]DNSAction)
	for domain, policy := range c.Domains {
		domains[dns.CanonicalName(domain)] = policy
	}
	c.Domains = domains
}

// TProxy is a netxlite.TProxable that implements self censorship.
type TProxy struct {
	// config contains settings for TProxy.
	config *TProxyConfig

	// dnsClient is the DNS client we'll internally use.
	dnsClient netxlite.Resolver

	// dnsListener is the DNS listener.
	dnsListener DNSListener

	// httpListener is the HTTP listener.
	httpListener net.Listener

	// listenUDP allows overriding net.ListenUDP calls in tests
	listenUDP func(network string, laddr *net.UDPAddr) (quicx.UDPLikeConn, error)

	// logger is the underlying logger to use.
	logger Logger

	// tlsListener is the TLS listener.
	tlsListener net.Listener
}

//
// Constructor and destructor
//

// NewTProxy creates a new TProxy instance.
func NewTProxy(config *TProxyConfig, logger Logger) (*TProxy, error) {
	return newTProxy(config, logger, "127.0.0.1:0", "127.0.0.1:0", "127.0.0.1:0")
}

func newTProxy(config *TProxyConfig, logger Logger, dnsListenerAddr,
	tlsListenerAddr, httpListenerAddr string) (*TProxy, error) {
	p := &TProxy{
		config: config,
		listenUDP: func(network string, laddr *net.UDPAddr) (quicx.UDPLikeConn, error) {
			return net.ListenUDP(network, laddr)
		},
		logger: logger,
	}
	if err := p.newDNSListener(dnsListenerAddr); err != nil {
		return nil, err
	}
	p.newDNSClient(logger)
	if err := p.newTLSListener(tlsListenerAddr, logger); err != nil {
		p.dnsListener.Close()
		return nil, err
	}
	if err := p.newHTTPListener(httpListenerAddr); err != nil {
		p.dnsListener.Close()
		p.tlsListener.Close()
		return nil, err
	}
	return p, nil
}

func (p *TProxy) newDNSListener(listenAddr string) error {
	var err error
	dnsProxy := &DNSProxy{OnQuery: p.onQuery}
	p.dnsListener, err = dnsProxy.Start(listenAddr)
	return err
}

func (p *TProxy) newDNSClient(logger Logger) {
	dialer := netxlite.NewDialerWithoutResolver(logger)
	p.dnsClient = netxlite.NewResolverUDP(logger, dialer, p.dnsListener.LocalAddr().String())
}

func (p *TProxy) newTLSListener(listenAddr string, logger Logger) error {
	var err error
	tlsProxy := &TLSProxy{OnIncomingSNI: p.onIncomingSNI}
	p.tlsListener, err = tlsProxy.Start(listenAddr)
	return err
}

func (p *TProxy) newHTTPListener(listenAddr string) error {
	var err error
	httpProxy := &HTTPProxy{OnIncomingHost: p.onIncomingHost}
	p.httpListener, err = httpProxy.Start(listenAddr)
	return err
}

// Close closes the resources used by a TProxy.
func (p *TProxy) Close() error {
	p.dnsClient.CloseIdleConnections()
	p.dnsListener.Close()
	p.httpListener.Close()
	p.tlsListener.Close()
	return nil
}

//
// QUIC
//

// ListenUDP implements netxlite.TProxy.ListenUDP.
func (p *TProxy) ListenUDP(network string, laddr *net.UDPAddr) (quicx.UDPLikeConn, error) {
	pconn, err := p.listenUDP(network, laddr)
	if err != nil {
		return nil, err
	}
	return &tProxyUDPLikeConn{UDPLikeConn: pconn, proxy: p}, nil
}

// tProxyUDPLikeConn is a TProxy-aware UDPLikeConn.
type tProxyUDPLikeConn struct {
	// UDPLikeConn is the underlying conn type.
	quicx.UDPLikeConn

	// proxy refers to the TProxy.
	proxy *TProxy
}

// WriteTo implements UDPLikeConn.WriteTo. This function will
// apply the proper tproxy policies, if required.
func (c *tProxyUDPLikeConn) WriteTo(pkt []byte, addr net.Addr) (int, error) {
	endpoint := fmt.Sprintf("%s/%s", addr.String(), addr.Network())
	policy := c.proxy.config.Endpoints[endpoint]
	switch policy {
	case TProxyPolicyDropData:
		c.proxy.logger.Infof("tproxy: WriteTo: %s => %s", endpoint, policy)
		return len(pkt), nil
	default:
		return c.UDPLikeConn.WriteTo(pkt, addr)
	}
}

//
// System resolver
//

// LookupHost implements netxlite.TProxy.LookupHost.
func (p *TProxy) LookupHost(ctx context.Context, domain string) ([]string, error) {
	return p.dnsClient.LookupHost(ctx, domain)
}

//
// Dialer
//

// NewTProxyDialer implements netxlite.TProxy.NewTProxyDialer.
func (p *TProxy) NewTProxyDialer(timeout time.Duration) netxlite.TProxyDialer {
	return &tProxyDialer{
		dialer: &net.Dialer{Timeout: timeout},
		proxy:  p,
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
	case TProxyPolicyTCPDropSYN:
		d.proxy.logger.Infof("tproxy: DialContext: %s/%s => %s", address, network, policy)
		var cancel context.CancelFunc
		const timeout = 70 * time.Second
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
		<-ctx.Done()
		return nil, errors.New("i/o timeout")
	case TProxyPolicyTCPRejectSYN:
		d.proxy.logger.Infof("tproxy: DialContext: %s/%s => %s", address, network, policy)
		return nil, netxlite.ECONNREFUSED
	case TProxyPolicyHijackDNS:
		d.proxy.logger.Infof("tproxy: DialContext: %s/%s => %s", address, network, policy)
		address = d.proxy.dnsListener.LocalAddr().String()
	case TProxyPolicyHijackTLS:
		d.proxy.logger.Infof("tproxy: DialContext: %s/%s => %s", address, network, policy)
		address = d.proxy.tlsListener.Addr().String()
	case TProxyPolicyHijackHTTP:
		d.proxy.logger.Infof("tproxy: DialContext: %s/%s => %s", address, network, policy)
		address = d.proxy.httpListener.Addr().String()
	default:
		// nothing
	}
	conn, err := d.dialer.DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}
	return &tProxyConn{Conn: conn, proxy: d.proxy}, nil
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
		c.proxy.logger.Infof("tproxy: Write: %s => %s", endpoint, policy)
		return len(b), nil
	default:
		return c.Conn.Write(b)
	}
}

//
// Filtering policies implementation
//

// onQuery is called for filtering outgoing DNS queries.
func (p *TProxy) onQuery(domain string) DNSAction {
	policy := p.config.Domains[domain]
	if policy == "" {
		policy = DNSActionPass
	} else {
		p.logger.Infof("tproxy: DNS: %s => %s", domain, policy)
	}
	return policy
}

// onIncomingSNI is called for filtering SNI values.
func (p *TProxy) onIncomingSNI(sni string) TLSAction {
	policy := p.config.SNIs[sni]
	if policy == "" {
		policy = TLSActionPass
	} else {
		p.logger.Infof("tproxy: TLS: %s => %s", sni, policy)
	}
	return policy
}

// onIncomingHost is called for filtering HTTP hosts.
func (p *TProxy) onIncomingHost(host string) HTTPAction {
	policy := p.config.Hosts[host]
	if policy == "" {
		policy = HTTPActionPass
	} else {
		p.logger.Infof("tproxy: HTTP: %s => %s", host, policy)
	}
	return policy
}
