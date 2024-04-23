package probeservices

//
// measurementmeta.go - GET /api/v1/measurement_meta
//

import (
	"context"
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/httpclientx"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// GetMeasurementMeta returns meta information about a measurement.
func (c Client) GetMeasurementMeta(
	ctx context.Context, input model.OOAPIMeasurementMetaConfig) (*model.OOAPIMeasurementMeta, error) {
	// construct the query to use
	query := url.Values{}
	query.Add("report_id", input.ReportID)
	if input.Input != "" {
		query.Add("input", input.Input)
	}
	if input.Full {
		query.Add("full", "true")
	}

	// construct the URL to use
	URL, err := url.Parse(c.BaseURL)
	if err != nil {
		return nil, err
	}
	URL.Path = "/api/v1/measurement_meta"
	URL.RawQuery = query.Encode()

	// get the response
	return httpclientx.GetJSON[*model.OOAPIMeasurementMeta](ctx, URL.String(), &httpclientx.Config{
		Client:    c.HTTPClient,
		Logger:    c.Logger,
		UserAgent: c.UserAgent,
	})
}
