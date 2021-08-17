package websteps

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"sync"

	"github.com/apex/log"
	"github.com/lucas-clemente/quic-go"
	"github.com/lucas-clemente/quic-go/http3"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/quicdialer"
	"github.com/ooni/probe-cli/v3/internal/errorsx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

var ErrNoConnReuse = errors.New("cannot reuse connection")

// NewDialerResolver contructs a new dialer for TCP connections,
// with default, errorwrapping and resolve functionalities
func NewDialerResolver(resolver netxlite.Resolver) netxlite.Dialer {
	var d netxlite.Dialer = netxlite.DefaultDialer
	d = &errorsx.ErrorWrapperDialer{Dialer: d}
	d = &netxlite.DialerResolver{Resolver: resolver, Dialer: d}
	return d
}

// NewQUICDialerResolver creates a new QUICDialerResolver
// with default, errorwrapping and resolve functionalities
func NewQUICDialerResolver(resolver netxlite.Resolver) netxlite.QUICContextDialer {
	var ql quicdialer.QUICListener = &netxlite.QUICListenerStdlib{}
	ql = &errorsx.ErrorWrapperQUICListener{QUICListener: ql}
	var dialer netxlite.QUICContextDialer = &netxlite.QUICDialerQUICGo{
		QUICListener: ql,
	}
	dialer = &errorsx.ErrorWrapperQUICDialer{Dialer: dialer}
	dialer = &netxlite.QUICDialerResolver{Resolver: resolver, Dialer: dialer}
	return dialer
}

// NewSingleH3Transport creates an http3.RoundTripper
func NewSingleH3Transport(qsess quic.EarlySession, tlscfg *tls.Config, qcfg *quic.Config) *http3.RoundTripper {
	transport := &http3.RoundTripper{
		DisableCompression: true,
		TLSClientConfig:    tlscfg,
		QuicConfig:         qcfg,
		Dial:               (&SingleDialerH3{qsess: &qsess}).Dial,
	}
	return transport
}

// NewSingleTransport determines the appropriate HTTP Transport from the ALPN
func NewSingleTransport(conn net.Conn) (transport http.RoundTripper) {
	singledialer := &SingleDialer{conn: &conn}
	transport = http.DefaultTransport.(*http.Transport).Clone()
	transport.(*http.Transport).DialContext = singledialer.DialContext
	transport.(*http.Transport).DialTLSContext = singledialer.DialContext
	transport.(*http.Transport).DisableCompression = true
	transport.(*http.Transport).MaxConnsPerHost = 1
	transport = &netxlite.HTTPTransportLogger{Logger: log.Log, HTTPTransport: transport.(*http.Transport)}
	return transport
}

type SingleDialer struct {
	sync.Mutex
	conn *net.Conn
}

func (s *SingleDialer) DialContext(ctx context.Context, network string, addr string) (net.Conn, error) {
	s.Lock()
	defer s.Unlock()
	if s.conn == nil {
		return nil, ErrNoConnReuse
	}
	c := s.conn
	s.conn = nil
	return *c, nil
}

type SingleDialerH3 struct {
	sync.Mutex
	qsess *quic.EarlySession
}

func (s *SingleDialerH3) Dial(network, addr string, tlsCfg *tls.Config, cfg *quic.Config) (quic.EarlySession, error) {
	s.Lock()
	defer s.Unlock()
	if s.qsess == nil {
		return nil, ErrNoConnReuse
	}
	qs := s.qsess
	s.qsess = nil
	return *qs, nil
}
