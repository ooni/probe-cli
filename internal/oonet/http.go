package oonet

import "net/http"

// ErrHTTPRoundTrip wraps an error occurred during the HTTP round trip.
type ErrHTTPRoundTrip struct {
	error
}

// Unwrap returns the wrapped error.
func (err *ErrHTTPRoundTrip) Unwrap() error {
	return err.error
}

// RoundTrip implements http.RoundTripper.RoundTrip. On failure, this function will
// return an ErrHTTPRoundTrip error. This function will emit log messages with the
// txp.Logger logger. This function will eventually call ContextConfig().RoundTrip, if
// configured, or Transport.DefaultRoundTrip, otherwise.
func (txp *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	log := txp.logger()
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
	if overrides := ContextOverrides(ctx); overrides != nil && overrides.RoundTrip != nil {
		return overrides.RoundTrip(req)
	}
	return txp.DefaultRoundTrip(req)
}

// DefaultRoundTrip is the default implementation of RoundTrip.
func (txp *Transport) DefaultRoundTrip(req *http.Request) (*http.Response, error) {
	return txp.getOrCreateTransport().RoundTrip(req)
}
