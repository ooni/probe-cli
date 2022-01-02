package measurex

import (
	"net/http"
	"net/url"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// NewTracingHTTPTransport creates a new HTTPTransport
// instance with events tracing.
//
// Arguments:
//
// - logger is the logger to use
//
// - begin is the zero time for measurements
//
// - db is the DB in which to write events that will
// eventually become the measurement
//
// - dialer is the base dialer to establish conns
//
// - resolver is the underlying resolver to use
//
// - handshake is the TLS handshaker to use
func NewTracingHTTPTransport(logger model.Logger, begin time.Time, db WritableDB,
	resolver Resolver, dialer model.Dialer, handshaker TLSHandshaker) *HTTPTransportDB {
	resolver = WrapResolver(begin, db, resolver)
	dialer = netxlite.WrapDialer(logger, resolver, WrapDialer(begin, db, dialer))
	tlsDialer := netxlite.NewTLSDialer(dialer, handshaker)
	return WrapHTTPTransport(
		begin, db, netxlite.NewHTTPTransport(logger, dialer, tlsDialer))
}

// NewTracingHTTPTransportWithDefaultSettings creates a new
// HTTP transport with tracing capabilities and default settings.
//
// Arguments:
//
// - begin is the zero time for measurements
//
// - logger is the logger to use
//
// - db is the DB in which to write events that will
// eventually become the measurement
//
func NewTracingHTTPTransportWithDefaultSettings(
	begin time.Time, logger model.Logger, db WritableDB) *HTTPTransportDB {
	return NewTracingHTTPTransport(logger, begin, db,
		netxlite.NewResolverStdlib(logger),
		netxlite.NewDialerWithoutResolver(logger),
		netxlite.NewTLSHandshakerStdlib(logger))
}

func (mx *Measurer) NewTracingHTTPTransportWithDefaultSettings(
	logger model.Logger, db WritableDB) *HTTPTransportDB {
	return NewTracingHTTPTransport(
		mx.Logger, mx.Begin, db, mx.NewResolverSystem(db, mx.Logger),
		mx.NewDialerWithoutResolver(db, mx.Logger),
		mx.TLSHandshaker)
}

// UnmeasuredHTTPEndpoints returns the endpoints whose IP address
// has been resolved but for which we don't have any measurement
// inside of the given database. The returned list will be
// empty if there is no such endpoint in the DB. This function will
// return an error if the URL is not valid or not HTTP/HTTPS.
func UnmeasuredHTTPEndpoints(db *MeasurementDB, URL string,
	headers http.Header) ([]*HTTPEndpoint, error) {
	parsedURL, err := url.Parse(URL)
	if err != nil {
		return nil, err
	}
	m := &DNSMeasurement{
		Domain:      parsedURL.Hostname(),
		Measurement: db.AsMeasurement(),
	}
	return AllHTTPEndpointsForURL(parsedURL, headers, m)
}
