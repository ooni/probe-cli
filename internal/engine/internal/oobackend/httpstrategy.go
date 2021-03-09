package oobackend

import (
	"net/http"
	"net/url"
)

// httpStrategy is a strategy where we use http. Specifically, we
// will rewrite the outgoing request to use the base URL configured
// inside of the mandatory Info field.
type httpStrategy struct {
	HTTPClient HTTPClient    //optional
	Info       *strategyInfo // mandatory
}

// Do implements strategy.Do.
func (s *httpStrategy) Do(req *http.Request) (*http.Response, error) {
	resp, err := s.do(req)
	s.Info.updatescore(err) // track the strategy score
	return resp, err
}

// do calls HTTPClient.Do with a clone of req using the URL configured
// in the strategy as the base URL.
func (s *httpStrategy) do(req *http.Request) (*http.Response, error) {
	clnt := s.HTTPClient
	if clnt == nil {
		clnt = http.DefaultClient // as promised in the docs
	}
	URL, err := url.Parse(s.Info.URL)
	if err != nil {
		return nil, err
	}
	// Note: Client.do guarantees that we're operating on
	// a deep copy of the original request.
	req.URL.Host = URL.Host     // replace the host
	req.URL.Scheme = URL.Scheme // replace the scheme
	return clnt.Do(req)
}

// StrategyInfo implements strategy.StrategyInfo.
func (s *httpStrategy) StrategyInfo() *strategyInfo {
	return s.Info
}
