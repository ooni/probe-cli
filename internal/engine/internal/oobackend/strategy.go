package oobackend

import (
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

// readstrategies returns all the available strategies
// sorted from the best one to the worst one, in most
// cases. With low probability, this function will instead
// reset the strategies score to their default.
func (c *Client) readstrategies(seed int64) []strategy {
	in := c.readstatemaybedefault(seed)
	var out []strategy
	for _, sg := range in {
		URL, err := url.Parse(sg.URL)
		if err != nil {
			// TODO(bassosimone): should we log this error?
			continue
		}
		switch URL.Scheme {
		case "https":
			out = append(out, &httpStrategy{
				HTTPClient: c.HTTPClientDefault, Info: sg})
		case "tunnel":
			out = append(out, &tunnelStrategy{
				Broker: c.HTTPTunnelBroker, Name: URL.Host, Info: sg,
			})
		}
	}
	return out
}

// writestrategies writes the available strategies on
// disk using the client's kvstore.
func (c *Client) writestrategies(sg []strategy) error {
	var out []*strategyInfo
	for _, e := range sg {
		out = append(out, e.StrategyInfo())
	}
	return c.writestate(out)
}
