package probeservices

import (
	"context"
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/httpx"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// GetMeasurementMeta returns meta information about a measurement.
func (c Client) GetMeasurementMeta(
	ctx context.Context, config model.OOAPIMeasurementMetaConfig) (*model.OOAPIMeasurementMeta, error) {
	query := url.Values{}
	query.Add("report_id", config.ReportID)
	if config.Input != "" {
		query.Add("input", config.Input)
	}
	if config.Full {
		query.Add("full", "true")
	}
	var response model.OOAPIMeasurementMeta
	err := (&httpx.APIClientTemplate{
		BaseURL:    c.BaseURL,
		HTTPClient: c.HTTPClient,
		Logger:     c.Logger,
		UserAgent:  c.UserAgent,
	}).WithBodyLogging().Build().GetJSONWithQuery(ctx, "/api/v1/measurement_meta", query, &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}
