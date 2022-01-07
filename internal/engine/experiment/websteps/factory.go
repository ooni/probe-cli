package websteps

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"net/url"
	"sync"

	"github.com/lucas-clemente/quic-go"
	"github.com/lucas-clemente/quic-go/http3"
	oohttp "github.com/ooni/oohttp"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

var ErrNoConnReuse = errors.New("cannot reuse connection")

func NewRequest(ctx context.Context, URL *url.URL, headers http.Header) *http.Request {
	req, err := http.NewRequestWithContext(ctx, "GET", URL.String(), nil)
	runtimex.PanicOnError(err, "NewRequestWithContect failed")
	for k, vs := range headers {
		for _, v := range vs {
			req.Header.Add(k, v)
		}
	}
	return req
}

// NewDialerResolver contructs a new dialer for TCP connections,
// with default, errorwrapping and resolve functionalities
func NewDialerResolver(resolver netxlite.ResolverLegacy) model.Dialer {
	var d model.Dialer = netxlite.DefaultDialer
	d = &netxlite.ErrorWrapperDialer{Dialer: d}
	d = &netxlite.DialerResolver{
		Resolver: netxlite.NewResolverLegacyAdapter(resolver),
		Dialer:   d,
	}
	return d
}

// NewQUICDialerResolver creates a new QUICDialerResolver
// with default, errorwrapping and resolve functionalities
func NewQUICDialerResolver(resolver netxlite.ResolverLegacy) model.QUICDialer {
	var ql model.QUICListener = &netxlite.QUICListenerStdlib{}
	ql = &netxlite.ErrorWrapperQUICListener{QUICListener: ql}
	var dialer model.QUICDialer = &netxlite.QUICDialerQUICGo{
		QUICListener: ql,
	}
	dialer = &netxlite.ErrorWrapperQUICDialer{QUICDialer: dialer}
	dialer = &netxlite.QUICDialerResolver{
		Resolver: netxlite.NewResolverLegacyAdapter(resolver),
		Dialer:   dialer,
	}
	return dialer
}

// NewSingleH3Transport creates an http3.RoundTripper.
func NewSingleH3Transport(qsess quic.EarlySession, tlscfg *tls.Config, qcfg *quic.Config) http.RoundTripper {
	transport := &http3.RoundTripper{
		DisableCompression: true,
		TLSClientConfig:    tlscfg,
		QuicConfig:         qcfg,
		Dial:               (&SingleDialerH3{qsess: &qsess}).Dial,
	}
	return transport
}

// NewSingleTransport creates a new HTTP transport with a single-use dialer.
func NewSingleTransport(conn net.Conn) http.RoundTripper {
	singledialer := &SingleDialer{conn: &conn}
	transport := newBaseTransport()
	transport.DialContext = singledialer.DialContext
	transport.DialTLSContext = singledialer.DialContext
	return transport
}

// NewSingleTransport creates a new HTTP transport with a custom dialer and handshaker.
func NewTransportWithDialer(dialer netxlite.DialerLegacy, tlsConfig *tls.Config, handshaker model.TLSHandshaker) http.RoundTripper {
	transport := newBaseTransport()
	transport.DialContext = dialer.DialContext
	transport.DialTLSContext = (&netxlite.TLSDialerLegacy{
		Config:        tlsConfig,
		Dialer:        netxlite.NewDialerLegacyAdapter(dialer),
		TLSHandshaker: handshaker,
	}).DialTLSContext
	return transport
}

// newBaseTransport creates a new HTTP transport with the default dialer.
func newBaseTransport() (transport *oohttp.StdlibTransport) {
	base := oohttp.DefaultTransport.(*oohttp.Transport).Clone()
	base.DisableCompression = true
	base.MaxConnsPerHost = 1
	transport = &oohttp.StdlibTransport{Transport: base}
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
