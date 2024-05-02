package probeservices

import (
	"context"
	"fmt"
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/httpclientx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/urlx"
)

// FetchOpenVPNConfig returns valid configuration for the openvpn experiment.
// It accepts the provider label, and the country code for the probe, in case the API wants to
// return different targets to us depending on where we are located.
func (c Client) FetchOpenVPNConfig(ctx context.Context, provider, cc string) (result model.OOAPIVPNProviderConfig, err error) {
	// create query string
	query := url.Values{}
	query.Add("country_code", cc)

	URL, err := urlx.ResolveReference(c.BaseURL,
		fmt.Sprintf("/api/v2/ooniprobe/vpn-config/%svpn/", provider),
		query.Encode())
	if err != nil {
		return
	}

	// get response
	//
	// use a model.DiscardLogger to avoid logging bridges
	return httpclientx.GetJSON[model.OOAPIVPNProviderConfig](ctx, URL, &httpclientx.Config{
		Client:    c.HTTPClient,
		Host:      c.Host,
		Logger:    model.DiscardLogger,
		UserAgent: c.UserAgent,
	})
}
