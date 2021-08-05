package nwebconnectivity

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
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"golang.org/x/net/http2"
)

var ErrNoConnReuse = errors.New("cannot reuse connection")

// getHTTP3Transport creates am http3.RoundTripper
func GetSingleH3Transport(qsess quic.EarlySession, tlscfg *tls.Config, qcfg *quic.Config) *http3.RoundTripper {
	transport := &http3.RoundTripper{
		DisableCompression: true,
		TLSClientConfig:    tlscfg,
		QuicConfig:         qcfg,
		Dial:               (&SingleDialerH3{qsess: &qsess}).Dial,
	}
	return transport
}

// getTransport determines the appropriate HTTP Transport from the ALPN
func GetSingleTransport(state *tls.ConnectionState, conn net.Conn, config *tls.Config) http.RoundTripper {
	if state == nil {
		return netxlite.NewHTTPTransport(&SingleDialerHTTP1{conn: &conn}, nil, nil)
	}
	// ALPN ?
	switch state.NegotiatedProtocol {
	case "h2":
		// HTTP 2 + TLS.
		return getHTTP2Transport(conn, config)
	default:
		// assume HTTP 1.x + TLS.
		return getHTTPTransport(conn, config)
	}
}

// getHTTPTransport creates an http.Transport
func getHTTPTransport(conn net.Conn, config *tls.Config) (transport http.RoundTripper) {
	transport = &http.Transport{
		DialContext:        (&SingleDialerHTTP1{conn: &conn}).DialContext,
		DialTLSContext:     (&SingleDialerHTTP1{conn: &conn}).DialContext,
		TLSClientConfig:    config,
		DisableCompression: true,
	}
	transport = &netxlite.HTTPTransportLogger{Logger: log.Log, HTTPTransport: transport.(*http.Transport)}
	return transport
}

// getHTTP2Transport creates an http2.Transport
func getHTTP2Transport(conn net.Conn, config *tls.Config) (transport http.RoundTripper) {
	transport = &http2.Transport{
		DialTLS:            (&SingleDialerH2{conn: &conn}).DialTLS,
		TLSClientConfig:    config,
		DisableCompression: true,
	}
	transport = &netxlite.HTTPTransportLogger{Logger: log.Log, HTTPTransport: transport.(*http2.Transport)}
	return transport
}

type SingleDialerHTTP1 struct {
	sync.Mutex
	conn *net.Conn
}

func (s *SingleDialerHTTP1) DialContext(ctx context.Context, network string, addr string) (net.Conn, error) {
	s.Lock()
	defer s.Unlock()
	if s.conn == nil {
		return nil, ErrNoConnReuse
	}
	c := s.conn
	s.conn = nil
	return *c, nil
}

type SingleDialerH2 struct {
	sync.Mutex
	conn *net.Conn
}

func (s *SingleDialerH2) DialTLS(network string, addr string, cfg *tls.Config) (net.Conn, error) {
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
