package mocks

//
// Mocks for model.Trace
//

import (
	"crypto/tls"
	"time"

	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// Trace allows mocking model.Trace.
type Trace struct {
	MockTimeNow func() time.Time

	MockOnDNSRoundTripForLookupHost func(started time.Time, reso model.Resolver, query model.DNSQuery,
		response model.DNSResponse, addrs []string, err error, finished time.Time)

	MockOnConnectDone func(
		started time.Time, network, domain, remoteAddr string, err error, finished time.Time)

	MockOnTLSHandshakeStart func(now time.Time, remoteAddr string, config *tls.Config)

	MockOnTLSHandshakeDone func(started time.Time, remoteAddr string, config *tls.Config,
		state tls.ConnectionState, err error, finished time.Time)

	MockOnQUICHandshakeStart func(now time.Time, remoteAddrs string, config *quic.Config)

	MockOnQUICHandshakeDone func(started time.Time, remoteAddr string, qconn quic.EarlyConnection,
		config *tls.Config, err error, finished time.Time)
}

var _ model.Trace = &Trace{}

func (t *Trace) TimeNow() time.Time {
	return t.MockTimeNow()
}

func (t *Trace) OnDNSRoundTripForLookupHost(started time.Time, reso model.Resolver, query model.DNSQuery,
	response model.DNSResponse, addrs []string, err error, finished time.Time) {
	t.MockOnDNSRoundTripForLookupHost(started, reso, query, response, addrs, err, finished)
}

func (t *Trace) OnConnectDone(
	started time.Time, network, domain, remoteAddr string, err error, finished time.Time) {
	t.MockOnConnectDone(started, network, domain, remoteAddr, err, finished)
}

func (t *Trace) OnTLSHandshakeStart(now time.Time, remoteAddr string, config *tls.Config) {
	t.MockOnTLSHandshakeStart(now, remoteAddr, config)
}

func (t *Trace) OnTLSHandshakeDone(started time.Time, remoteAddr string, config *tls.Config,
	state tls.ConnectionState, err error, finished time.Time) {
	t.MockOnTLSHandshakeDone(started, remoteAddr, config, state, err, finished)
}

func (t *Trace) OnQUICHandshakeStart(now time.Time, remoteAddr string, config *quic.Config) {
	t.MockOnQUICHandshakeStart(now, remoteAddr, config)
}

func (t *Trace) OnQUICHandshakeDone(started time.Time, remoteAddr string, qconn quic.EarlyConnection,
	config *tls.Config, err error, finished time.Time) {
	t.MockOnQUICHandshakeDone(started, remoteAddr, qconn, config, err, finished)
}
