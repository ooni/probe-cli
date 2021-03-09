package oobackend

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
)

// strategy is a strategy with which we attempt to
// communicate with the OONI backend.
type strategy interface {
	// Do implements http.Client.Do.
	Do(req *http.Request) (*http.Response, error)

	// Strategy returns the underlying strategy.
	StrategyInfo() *strategyInfo
}

// ErrNoStrategy indicates that we don't support this strategy.
var ErrNoStrategy = errors.New("oobackend: unsupported strategy")

// makestrategy creates a strategy from a strategyInfo.
func (c *Client) makestrategy(si *strategyInfo) (strategy, error) {
	URL, err := url.Parse(si.URL)
	if err != nil {
		return nil, err
	}
	switch URL.Scheme {
	case "https":
		return &httpStrategy{HTTPClient: c.HTTPClientDefault, Info: si}, nil
	case "tunnel":
		return &tunnelStrategy{
			Broker: c.HTTPTunnelBroker, Name: URL.Host, Info: si}, nil
	default:
		return nil, fmt.Errorf("%w: %s", ErrNoStrategy, URL.Scheme)
	}
}
