package probeservices

import (
	"context"
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/httpx"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// CheckReportID checks whether the given ReportID exists.
func (c Client) CheckReportID(ctx context.Context, reportID string) (bool, error) {
	query := url.Values{}
	query.Add("report_id", reportID)
	var response model.OOAPICheckReportIDResponse
	err := (&httpx.APIClientTemplate{
		BaseURL:    c.BaseURL,
		HTTPClient: c.HTTPClient,
		Logger:     c.Logger,
		UserAgent:  c.UserAgent,
	}).WithBodyLogging().Build().GetJSONWithQuery(ctx, "/api/_/check_report_id", query, &response)
	if err != nil {
		return false, err
	}
	return response.Found, nil
}
