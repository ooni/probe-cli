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
)

var ErrNoConnReuse = errors.New("cannot reuse connection")

// GetSingleH3Transport creates am http3.RoundTripper
func GetSingleH3Transport(qsess quic.EarlySession, tlscfg *tls.Config, qcfg *quic.Config) *http3.RoundTripper {
	transport := &http3.RoundTripper{
		DisableCompression: true,
		TLSClientConfig:    tlscfg,
		QuicConfig:         qcfg,
		Dial:               (&SingleDialerH3{qsess: &qsess}).Dial,
	}
	return transport
}

// GetSingleTransport determines the appropriate HTTP Transport from the ALPN
func GetSingleTransport(conn net.Conn, config *tls.Config) (transport http.RoundTripper) {
	transport = &http.Transport{
		DialContext:        (&SingleDialer{conn: &conn}).DialContext,
		DialTLSContext:     (&SingleDialer{conn: &conn}).DialContext,
		TLSClientConfig:    config,
		DisableCompression: true,
	}
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
