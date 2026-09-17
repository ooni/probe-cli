package probeservices

import (
	"context"
	"errors"
	"fmt"
	"log"
	"reflect"
	"sync"

	"github.com/ooni/probe-cli/v3/internal/httpclientx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/urlx"
)

var (
	// ErrUnsupportedDataFormatVersion indicates that the user provided
	// in input a data format version that we do not support.
	ErrUnsupportedDataFormatVersion = errors.New("unsupported data format version")

	// ErrUnsupportedFormat indicates that the format is not supported.
	ErrUnsupportedFormat = errors.New("unsupported format")

	// ErrJSONFormatNotSupported indicates that the collector we're using
	// does not support the JSON report format.
	ErrJSONFormatNotSupported = errors.New("JSON format not supported")
)

// NewReportTemplate creates a new ReportTemplate from a Measurement.
func NewReportTemplate(m *model.Measurement) model.OOAPIReportTemplate {
	return model.OOAPIReportTemplate{
		DataFormatVersion: model.OOAPIReportDefaultDataFormatVersion,
		Format:            model.OOAPIReportDefaultFormat,
		ProbeASN:          m.ProbeASN,
		ProbeCC:           m.ProbeCC,
		SoftwareName:      m.SoftwareName,
		SoftwareVersion:   m.SoftwareVersion,
		TestName:          m.TestName,
		TestStartTime:     m.TestStartTime,
		TestVersion:       m.TestVersion,
	}
}

type reportChan struct {
	// ID is the report ID
	ID string

	// client is the client that was used.
	client Client

	// tmpl is the template used when opening this report.
	tmpl model.OOAPIReportTemplate
}

// OpenReport opens a new report.
func (c Client) OpenReport(ctx context.Context, rt model.OOAPIReportTemplate) (ReportChannel, error) {
	if rt.DataFormatVersion != model.OOAPIReportDefaultDataFormatVersion {
		return nil, ErrUnsupportedDataFormatVersion
	}
	if rt.Format != model.OOAPIReportDefaultFormat {
		return nil, ErrUnsupportedFormat
	}

	URL, err := urlx.ResolveReference(c.BaseURL, "/report", "")
	if err != nil {
		return nil, err
	}

	cor, err := httpclientx.PostJSON[model.OOAPIReportTemplate, *model.OOAPICollectorOpenResponse](
		ctx,
		httpclientx.NewEndpoint(URL).WithHostOverride(c.Host),
		rt,
		&httpclientx.Config{
			Client:    c.HTTPClient,
			Logger:    c.Logger,
			UserAgent: c.UserAgent,
		},
	)

	if err != nil {
		return nil, err
	}

	for _, format := range cor.SupportedFormats {
		if format == "json" {
			return &reportChan{ID: cor.ReportID, client: c, tmpl: rt}, nil
		}
	}
	return nil, ErrJSONFormatNotSupported
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
func (r reportChan) SubmitMeasurement(ctx context.Context, m *model.Measurement) (string, error) {
	// TODO(bassosimone): do we need to prevent measurement submission
	// if the measurement isn't consistent with the orig template?

	m.ReportID = r.ID

	URL, err := urlx.ResolveReference(r.client.BaseURL, fmt.Sprintf("/report/%s", r.ID), "")
	if err != nil {
		return "", err
	}

	apiReq := model.OOAPICollectorUpdateRequest{
		Format:  "json",
		Content: m,
	}

	updateResponse, err := httpclientx.PostJSON[
		model.OOAPICollectorUpdateRequest, *model.OOAPICollectorUpdateResponse](
		ctx,
		httpclientx.NewEndpoint(URL).WithHostOverride(r.client.Host),
		apiReq,
		&httpclientx.Config{
			Client:    r.client.HTTPClient,
			Logger:    r.client.Logger,
			UserAgent: r.client.UserAgent,
		},
	)

	if err != nil {
		m.ReportID = ""
		return "", err
	}

	// TODO(bassosimone): we should use the session logger here but for now this stopgap
	// solution will allow observing the measurement URL for CLI users.
	log.Printf("Measurement URL: https://explorer.ooni.org/m/%s", updateResponse.MeasurementUID)
	return updateResponse.MeasurementUID, nil
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
	SubmitMeasurement(ctx context.Context, m *model.Measurement) (string, error)
}

var _ ReportChannel = &reportChan{}

// ReportOpener is any struct that is able to open a new ReportChannel. The
// Client struct belongs to this interface.
type ReportOpener interface {
	OpenReport(ctx context.Context, rt model.OOAPIReportTemplate) (ReportChannel, error)
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
func (sub *Submitter) Submit(ctx context.Context, m *model.Measurement) (string, error) {
	var err error
	sub.mu.Lock()
	defer sub.mu.Unlock()
	if sub.channel == nil || !sub.channel.CanSubmit(m) {
		sub.channel, err = sub.opener.OpenReport(ctx, NewReportTemplate(m))
		if err != nil {
			return "", err
		}
		sub.logger.Infof("New reportID: %s", sub.channel.ReportID())
	}
	return sub.channel.SubmitMeasurement(ctx, m)
}
