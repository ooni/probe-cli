package probeservices

//
// measurementmeta.go - GET /api/v1/measurement_meta
//

import (
	"context"
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/httpclientx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/urlx"
)

// GetMeasurementMeta returns meta information about a measurement.
func (c Client) GetMeasurementMeta(
	ctx context.Context, config model.OOAPIMeasurementMetaConfig) (*model.OOAPIMeasurementMeta, error) {
	// construct the query to use
	query := url.Values{}
	query.Add("report_id", config.ReportID)
	if config.Input != "" {
		query.Add("input", config.Input)
	}
	if config.Full {
		query.Add("full", "true")
	}

	// construct the URL to use
	URL, err := urlx.ResolveReference(c.BaseURL, "/api/v1/measurement_meta", query.Encode())
	if err != nil {
		return nil, err
	}

	// get the response
	return httpclientx.GetJSON[*model.OOAPIMeasurementMeta](
		ctx,
		httpclientx.NewEndpoint(URL).WithHostOverride(c.Host),
		&httpclientx.Config{
			Client:    c.HTTPClient,
			Logger:    c.Logger,
			UserAgent: c.UserAgent,
		})
}
