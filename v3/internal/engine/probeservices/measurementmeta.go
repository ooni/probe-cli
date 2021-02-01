package probeservices

import (
	"context"
	"net/url"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/internal/httpx"
)

// MeasurementMetaConfig contains configuration for GetMeasurementMeta.
type MeasurementMetaConfig struct {
	// ReportID is the mandatory report ID.
	ReportID string

	// Full indicates whether we also want the full measurement body.
	Full bool

	// Input is the optional input.
	Input string
}

// MeasurementMeta contains measurement metadata.
type MeasurementMeta struct {
	// Fields returned by the API server whenever we are
	// calling /api/v1/measurement_meta.
	Anomaly              bool      `json:"anomaly"`
	CategoryCode         string    `json:"category_code"`
	Confirmed            bool      `json:"confirmed"`
	Failure              bool      `json:"failure"`
	Input                *string   `json:"input"`
	MeasurementStartTime time.Time `json:"measurement_start_time"`
	ProbeASN             int64     `json:"probe_asn"`
	ProbeCC              string    `json:"probe_cc"`
	ReportID             string    `json:"report_id"`
	Scores               string    `json:"scores"`
	TestName             string    `json:"test_name"`
	TestStartTime        time.Time `json:"test_start_time"`

	// This field is only included if the user has specified
	// the config.Full option, otherwise it's empty.
	RawMeasurement string `json:"raw_measurement"`
}

// GetMeasurementMeta returns meta information about a measurement.
func (c Client) GetMeasurementMeta(
	ctx context.Context, config MeasurementMetaConfig) (*MeasurementMeta, error) {
	query := url.Values{}
	query.Add("report_id", config.ReportID)
	if config.Input != "" {
		query.Add("input", config.Input)
	}
	if config.Full {
		query.Add("full", "true")
	}
	var response MeasurementMeta
	err := (httpx.Client{
		BaseURL:    c.BaseURL,
		HTTPClient: c.HTTPClient,
		Logger:     c.Logger,
		UserAgent:  c.UserAgent,
	}).GetJSONWithQuery(ctx, "/api/v1/measurement_meta", query, &response)
	if err != nil {
		return nil, err
	}
	return &response, nil
}
