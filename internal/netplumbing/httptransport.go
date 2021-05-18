package netplumbing

import "net/http"

// ErrHTTPRoundTrip wraps an error occurred during the HTTP round trip.
type ErrHTTPRoundTrip struct {
	error
}

// Unwrap returns the wrapped error.
func (err *ErrHTTPRoundTrip) Unwrap() error {
	return err.error
}

// RoundTrip implements http.RoundTripper.RoundTrip.
func (txp *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	log := txp.logger(req.Context())
	log.Debugf("> %s %s", req.Method, req.URL)
	for key, values := range req.Header {
		for _, value := range values {
			log.Debugf("> %s: %s", key, value)
		}
	}
	log.Debug(">")
	resp, err := txp.routeRoundTrip(req)
	if err != nil {
		log.Debugf("< %s", err)
		return nil, &ErrHTTPRoundTrip{err}
	}
	log.Debugf("< %d", resp.StatusCode)
	for key, values := range resp.Header {
		for _, value := range values {
			log.Debugf("< %s: %s", key, value)
		}
	}
	log.Debug("<")
	return resp, nil
}

// routeRoundTrip routes the RoundTrip call.
func (txp *Transport) routeRoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	if config := ContextConfig(ctx); config != nil && config.HTTPTransport != nil {
		return config.HTTPTransport.RoundTrip(req)
	}
	return txp.RoundTripper.RoundTrip(req)
}

// CloseIdleConnections closes idle connections.
func (txp *Transport) CloseIdleConnections() {
	txp.RoundTripper.CloseIdleConnections()
}
