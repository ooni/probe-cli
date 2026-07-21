package probeservices

import (
	"context"
	"encoding/json"

	"github.com/ooni/probe-cli/v3/internal/httpclientx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/urlx"
)

// Submit submits a measurement to the submit_measurement endpoint without a
// credential and returns the measurement UID.
func (c Client) Submit(ctx context.Context, m *model.Measurement) (string, error) {
	content, err := json.Marshal(m)
	if err != nil {
		return "", err
	}

	URL, err := urlx.ResolveReference(c.BaseURL, "/api/v1/submit_measurement", "")
	if err != nil {
		return "", err
	}

	resp, err := httpclientx.PostJSON[*model.OOAPISubmitMeasurementRequest, *model.OOAPISubmitMeasurementResponse](
		ctx,
		httpclientx.NewEndpoint(URL).WithHostOverride(c.Host),
		&model.OOAPISubmitMeasurementRequest{Format: "json", Content: string(content)},
		&httpclientx.Config{
			Client:    c.HTTPClient,
			Logger:    c.Logger,
			UserAgent: c.UserAgent,
		},
	)
	if err != nil {
		return "", err
	}

	c.Logger.Infof("Measurement URL: https://explorer.ooni.org/m/%s", resp.MeasurementUID)
	return resp.MeasurementUID, nil
}
