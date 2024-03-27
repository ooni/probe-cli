package probeservices

import (
	"context"
	"fmt"
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// FetchOpenVPNConfig returns valid configuration for the openvpn experiment.
// It accepts the provider label, and the country code for the probe, in case the API wants to
// return different targets to us depending on where we are located.
func (c Client) FetchOpenVPNConfig(ctx context.Context, provider, cc string) (result model.OOAPIVPNProviderConfig, err error) {
	_, auth, err := c.GetCredsAndAuth()
	if err != nil {
		return model.OOAPIVPNProviderConfig{}, err
	}
	s := fmt.Sprintf("Bearer %s", auth.Token)
	client := c.APIClientTemplate.BuildWithAuthorization(s)
	query := url.Values{}
	query.Add("country_code", cc)

	err = client.GetJSONWithQuery(
		ctx,
		fmt.Sprintf("/api/v2/ooniprobe/vpn-config/%s/", provider),
		query,
		&result,
	)
	return
}
