package netxlite

import "net/http"

// HTTPTransport is an http.Transport-like structure.
type HTTPTransport interface {
	// RoundTrip performs the HTTP round trip.
	RoundTrip(req *http.Request) (*http.Response, error)

	// CloseIdleConnections closes idle connections.
	CloseIdleConnections()
}

// HTTPTransportLogger is an HTTPTransport with logging.
type HTTPTransportLogger struct {
	// HTTPTransport is the underlying HTTP transport.
	HTTPTransport HTTPTransport

	// Logger is the underlying logger.
	Logger Logger
}

var _ HTTPTransport = &HTTPTransportLogger{}

// RoundTrip implements HTTPTransport.RoundTrip.
func (txp *HTTPTransportLogger) RoundTrip(req *http.Request) (*http.Response, error) {
	host := req.Host
	if host == "" {
		host = req.URL.Host
	}
	req.Header.Set("Host", host) // anticipate what Go would do
	return txp.logTrip(req)
}

// logTrip is an HTTP round trip with logging.
func (txp *HTTPTransportLogger) logTrip(req *http.Request) (*http.Response, error) {
	txp.Logger.Debugf("> %s %s", req.Method, req.URL.String())
	for key, values := range req.Header {
		for _, value := range values {
			txp.Logger.Debugf("> %s: %s", key, value)
		}
	}
	txp.Logger.Debug(">")
	resp, err := txp.HTTPTransport.RoundTrip(req)
	if err != nil {
		txp.Logger.Debugf("< %s", err)
		return nil, err
	}
	txp.Logger.Debugf("< %d", resp.StatusCode)
	for key, values := range resp.Header {
		for _, value := range values {
			txp.Logger.Debugf("< %s: %s", key, value)
		}
	}
	txp.Logger.Debug("<")
	return resp, nil
}

// CloseIdleConnections implement HTTPTransport.CloseIdleConnections.
func (txp *HTTPTransportLogger) CloseIdleConnections() {
	txp.HTTPTransport.CloseIdleConnections()
}
