package probeservices

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"

	"github.com/ooni/probe-cli/v3/internal/model"
)

const (
	// DefaultDataFormatVersion is the default data format version.
	//
	// See https://github.com/ooni/spec/tree/master/data-formats#history.
	DefaultDataFormatVersion = "0.2.0"

	// DefaultFormat is the default format
	DefaultFormat = "json"
)

var (
	// ErrUnsupportedDataFormatVersion indicates that the user provided
	// in input a data format version that we do not support.
	ErrUnsupportedDataFormatVersion = errors.New("Unsupported data format version")

	// ErrUnsupportedFormat indicates that the format is not supported.
	ErrUnsupportedFormat = errors.New("Unsupported format")

	// ErrJSONFormatNotSupported indicates that the collector we're using
	// does not support the JSON report format.
	ErrJSONFormatNotSupported = errors.New("JSON format not supported")
)

// ReportTemplate is the template for opening a report
type ReportTemplate struct {
	// DataFormatVersion is unconditionally set to DefaultDataFormatVersion
	// and you don't need to be concerned about it.
	DataFormatVersion string `json:"data_format_version"`

	// Format is unconditionally set to `json` and you don't need
	// to be concerned about it.
	Format string `json:"format"`

	// ProbeASN is the probe's autonomous system number (e.g. `AS1234`)
	ProbeASN string `json:"probe_asn"`

	// ProbeCC is the probe's country code (e.g. `IT`)
	ProbeCC string `json:"probe_cc"`

	// SoftwareName is the app name (e.g. `measurement-kit`)
	SoftwareName string `json:"software_name"`

	// SoftwareVersion is the app version (e.g. `0.9.1`)
	SoftwareVersion string `json:"software_version"`

	// TestName is the test name (e.g. `ndt`)
	TestName string `json:"test_name"`

	// TestStartTime contains the test start time
	TestStartTime string `json:"test_start_time"`

	// TestVersion is the test version (e.g. `1.0.1`)
	TestVersion string `json:"test_version"`
}

// NewReportTemplate creates a new ReportTemplate from a Measurement.
func NewReportTemplate(m *model.Measurement) ReportTemplate {
	return ReportTemplate{
		DataFormatVersion: DefaultDataFormatVersion,
		Format:            DefaultFormat,
		ProbeASN:          m.ProbeASN,
		ProbeCC:           m.ProbeCC,
		SoftwareName:      m.SoftwareName,
		SoftwareVersion:   m.SoftwareVersion,
		TestName:          m.TestName,
		TestStartTime:     m.TestStartTime,
		TestVersion:       m.TestVersion,
	}
}

type collectorOpenResponse struct {
	ID               string   `json:"report_id"`
	SupportedFormats []string `json:"supported_formats"`
}

type reportChan struct {
	// ID is the report ID
	ID string

	// client is the client that was used.
	client Client

	// tmpl is the template used when opening this report.
	tmpl ReportTemplate
}

// OpenReport opens a new report.
func (c Client) OpenReport(ctx context.Context, rt ReportTemplate) (ReportChannel, error) {
	if rt.DataFormatVersion != DefaultDataFormatVersion {
		return nil, ErrUnsupportedDataFormatVersion
	}
	if rt.Format != DefaultFormat {
		return nil, ErrUnsupportedFormat
	}
	var cor collectorOpenResponse
	if err := c.APIClientTemplate.Build().PostJSON(ctx, "/report", rt, &cor); err != nil {
		return nil, err
	}
	for _, format := range cor.SupportedFormats {
		if format == "json" {
			return &reportChan{ID: cor.ID, client: c, tmpl: rt}, nil
		}
	}
	return nil, ErrJSONFormatNotSupported
}

type collectorUpdateRequest struct {
	// Format is the data format
	Format string `json:"format"`

	// Content is the actual report
	Content interface{} `json:"content"`
}

type collectorUpdateResponse struct {
	// ID is the measurement ID
	ID string `json:"measurement_id"`
}

// CanSubmit returns true whether the provided measurement belongs to
// this report, false otherwise. We say that a given measurement belongs
// to this report if its report template matches the report's one.
func (r reportChan) CanSubmit(m *model.Measurement) bool {
	return reflect.DeepEqual(NewReportTemplate(m), r.tmpl)
}

// SubmitMeasurement submits a measurement belonging to the report
// to the OONI collector. On success, we will modify the measurement
// such that it contains the report ID for which it has been
// submitted. Otherwise, we'll set the report ID to the empty
// string, so that you know which measurements weren't submitted.
func (r reportChan) SubmitMeasurement(ctx context.Context, m *model.Measurement) error {
	var updateResponse collectorUpdateResponse
	m.ReportID = r.ID
	err := r.client.APIClientTemplate.Build().PostJSON(
		ctx, fmt.Sprintf("/report/%s", r.ID), collectorUpdateRequest{
			Format:  "json",
			Content: m,
		}, &updateResponse,
	)
	if err != nil {
		m.ReportID = ""
		return err
	}
	return nil
}

// ReportID returns the report ID.
func (r reportChan) ReportID() string {
	return r.ID
}

// ReportChannel is a channel through which one could submit measurements
// belonging to the same report. The Report struct belongs to this interface.
type ReportChannel interface {
	CanSubmit(m *model.Measurement) bool
	ReportID() string
	SubmitMeasurement(ctx context.Context, m *model.Measurement) error
}

var _ ReportChannel = &reportChan{}

// ReportOpener is any struct that is able to open a new ReportChannel. The
// Client struct belongs to this interface.
type ReportOpener interface {
	OpenReport(ctx context.Context, rt ReportTemplate) (ReportChannel, error)
}

var _ ReportOpener = Client{}

// Submitter is an abstraction allowing you to submit arbitrary measurements
// to a given OONI backend. This implementation will take care of opening
// reports when needed as well as of closing reports when needed. Nonetheless
// you need to remember to call its Close method when done, because there is
// likely an open report that has not been closed yet.
type Submitter struct {
	channel ReportChannel
	logger  model.Logger
	mu      sync.Mutex
	opener  ReportOpener
}

// NewSubmitter creates a new Submitter instance.
func NewSubmitter(opener ReportOpener, logger model.Logger) *Submitter {
	return &Submitter{opener: opener, logger: logger}
}

// Submit submits the current measurement to the OONI backend created using
// the ReportOpener passed to the constructor.
func (sub *Submitter) Submit(ctx context.Context, m *model.Measurement) error {
	var err error
	sub.mu.Lock()
	defer sub.mu.Unlock()
	if sub.channel == nil || !sub.channel.CanSubmit(m) {
		sub.channel, err = sub.opener.OpenReport(ctx, NewReportTemplate(m))
		if err != nil {
			return err
		}
		sub.logger.Infof("New reportID: %s", sub.channel.ReportID())
	}
	return sub.channel.SubmitMeasurement(ctx, m)
}
