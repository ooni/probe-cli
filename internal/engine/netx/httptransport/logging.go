package httptransport

import "net/http"

// Logger is the logger assumed by this package
type Logger interface {
	Debugf(format string, v ...interface{})
	Debug(message string)
}

// LoggingTransport is a logging transport
type LoggingTransport struct {
	RoundTripper
	Logger Logger
}

// RoundTrip implements RoundTripper.RoundTrip
func (txp LoggingTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	host := req.Host
	if host == "" {
		host = req.URL.Host
	}
	req.Header.Set("Host", host) // anticipate what Go would do
	return txp.logTrip(req)
}

func (txp LoggingTransport) logTrip(req *http.Request) (*http.Response, error) {
	txp.Logger.Debugf("> %s %s", req.Method, req.URL.String())
	for key, values := range req.Header {
		for _, value := range values {
			txp.Logger.Debugf("> %s: %s", key, value)
		}
	}
	txp.Logger.Debug(">")
	resp, err := txp.RoundTripper.RoundTrip(req)
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

var _ RoundTripper = LoggingTransport{}
