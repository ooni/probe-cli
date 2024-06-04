package probeservices

import (
	"context"
	"fmt"
	"net/url"
	"strings"

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

	// TODO(ainghazal): remove temporary fix
	if !strings.HasSuffix(provider, "vpn") {
		provider = provider + "vpn"
	}

	URL, err := urlx.ResolveReference(c.BaseURL,
		fmt.Sprintf("/api/v2/ooniprobe/vpn-config/%s", provider),
		query.Encode())
	if err != nil {
		return
	}

	// get response
	//
	// use a model.DiscardLogger to avoid logging config
	return httpclientx.GetJSON[model.OOAPIVPNProviderConfig](
		ctx,
		httpclientx.NewEndpoint(URL).WithHostOverride(c.Host),
		&httpclientx.Config{
			Client:    c.HTTPClient,
			Logger:    model.DiscardLogger,
			UserAgent: c.UserAgent,
		})
}
