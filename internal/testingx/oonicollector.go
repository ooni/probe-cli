package testingx

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/google/go-cmp/cmp"
	"github.com/google/uuid"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/must"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// OONICollector implements the OONI collector for testing.
//
// The zero value is ready to use.
//
// This struct methods panics for several errors. Only use for testing purposes!
type OONICollector struct {
	// EditOpenReportResponse is an OPTIONAL callback to edit the response
	// before the server actually sends it to the client.
	EditOpenReportResponse func(resp *model.OOAPICollectorOpenResponse)

	// EditUpdateResponse is an OPTIONAL callback to edit the response
	// before the server actually sends it to the client.
	EditUpdateResponse func(resp *model.OOAPICollectorUpdateResponse)

	// ValidateMeasurement is an OPTIONAL callback to validate the incoming measurement
	// beyond checks that ensure it is consistent with the original template.
	ValidateMeasurement func(meas *model.Measurement) error

	// ValidateReportTemplate is an OPTIONAL callback to validate the incoming report
	// template beyond the data format version and format fields values.
	ValidateReportTemplate func(rt *model.OOAPIReportTemplate) error

	// mu provides mutual exclusion.
	mu sync.Mutex

	// reports contains the open reports.
	reports map[string]*model.OOAPIReportTemplate
}

// OpenReport opens a report for the given report ID and template.
//
// This method is safe to call concurrently with other methods.
func (oc *OONICollector) OpenReport(reportID string, template *model.OOAPIReportTemplate) {
	oc.mu.Lock()
	if oc.reports == nil {
		oc.reports = make(map[string]*model.OOAPIReportTemplate)
	}
	oc.reports[reportID] = template
	oc.mu.Unlock()
}

// ServeHTTP implements [http.Handler].
//
// This method is safe to call concurrently with other methods.
func (oc *OONICollector) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// make sure that the method is POST
	if r.Method != "POST" {
		log.Printf("OONICollector: invalid method")
		w.WriteHeader(http.StatusNotImplemented)
		return
	}

	// make sure the URL path starts with /report
	if !strings.HasPrefix(r.URL.Path, "/report") {
		log.Printf("OONICollector: invalid URL path prefix")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// make sure that the content-type is application/json
	if r.Header.Get("Content-Type") != "application/json" {
		log.Printf("OONICollector: missing content-type header")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// read the raw request body or panic if we cannot read it
	body := runtimex.Try1(io.ReadAll(r.Body))

	log.Printf("OONICollector: URLPath %+v", r.URL.Path)
	log.Printf("OONICollector: request body %s", string(body))

	// handle the case where the user wants to open a new report
	if r.URL.Path == "/report" {
		log.Printf("OONICollector: opening new report")
		oc.openReport(w, body)
		return
	}

	// handle the case where the user wants to append to an existing report
	log.Printf("OONICollector: updating existing report")
	oc.updateReport(w, r.URL.Path, body)
}

// openReport handles opening a new OONI report.
func (oc *OONICollector) openReport(w http.ResponseWriter, body []byte) {
	// make sure we can parse the incoming request
	var template model.OOAPIReportTemplate
	if err := json.Unmarshal(body, &template); err != nil {
		log.Printf("OONICollector: cannot unmarshal JSON: %s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// make sure the data format version is OK
	if template.DataFormatVersion != model.OOAPIReportDefaultDataFormatVersion {
		log.Printf("OONICollector: invalid data format version")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// make sure the format is also OK
	if template.Format != model.OOAPIReportDefaultFormat {
		log.Printf("OONICollector: invalid format")
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// optionally allow the user to validate the report template
	if oc.ValidateReportTemplate != nil {
		if err := oc.ValidateReportTemplate(&template); err != nil {
			log.Printf("OONICollector: invalid report template: %s", err.Error())
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	// create the response
	response := &model.OOAPICollectorOpenResponse{
		BackendVersion: "1.3.0",
		ReportID:       uuid.Must(uuid.NewRandom()).String(),
		SupportedFormats: []string{
			model.OOAPIReportDefaultFormat,
		},
	}

	// optionally allow the user to modify the response
	if oc.EditOpenReportResponse != nil {
		oc.EditOpenReportResponse(response)
	}

	// make sure we know that this report ID now exists - note that this must
	// happen after the client code has edited the response
	oc.OpenReport(response.ReportID, &template)

	// set the content-type header
	w.Header().Set("Content-Type", "application/json")

	// serialize and send
	w.Write(must.MarshalJSON(response))
}

// updateReport handles updating an existing OONI report.
func (oc *OONICollector) updateReport(w http.ResponseWriter, urlpath string, body []byte) {
	// get the report ID
	reportID := strings.TrimPrefix(urlpath, "/report/")

	// obtain the report template
	oc.mu.Lock()
	template := oc.reports[reportID]
	oc.mu.Unlock()

	// handle the case of missing template
	if template == nil {
		log.Printf("OONICollector: the report does not exist: %s", reportID)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// make sure we can parse the incoming request
	var request model.OOAPICollectorUpdateRequest
	if err := json.Unmarshal(body, &request); err != nil {
		log.Printf("OONICollector: cannot unmarshal JSON: %s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// make sure the measurement is encoded as JSON
	if request.Format != "json" {
		log.Printf("OONICollector: invalid request format: %s", request.Format)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// make sure we can parse the content
	var measurement model.Measurement
	if err := json.Unmarshal(must.MarshalJSON(request.Content), &measurement); err != nil {
		log.Printf("OONICollector: cannot unmarshal JSON: %s", err.Error())
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// make sure all the required fields match
	mt := &model.OOAPIReportTemplate{
		DataFormatVersion: measurement.DataFormatVersion,
		Format:            request.Format,
		ProbeASN:          measurement.ProbeASN,
		ProbeCC:           measurement.ProbeCC,
		SoftwareName:      measurement.SoftwareName,
		SoftwareVersion:   measurement.SoftwareVersion,
		TestName:          measurement.TestName,
		TestStartTime:     measurement.TestStartTime,
		TestVersion:       measurement.TestVersion,
	}
	if diff := cmp.Diff(template, mt); diff != "" {
		log.Printf("OONICollector: measurement differs from template %s", diff)
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// give the user a chance to validate the measurement
	if oc.ValidateMeasurement != nil {
		if err := oc.ValidateMeasurement(&measurement); err != nil {
			log.Printf("OONICollector: invalid measurement: %s", err.Error())
			w.WriteHeader(http.StatusBadRequest)
			return
		}
	}

	// create the response
	response := &model.OOAPICollectorUpdateResponse{
		MeasurementUID: uuid.Must(uuid.NewRandom()).String(),
	}

	// optionally allow the user to modify the response
	if oc.EditUpdateResponse != nil {
		oc.EditUpdateResponse(response)
	}

	// set the content-type header
	w.Header().Set("Content-Type", "application/json")

	// serialize and send
	w.Write(must.MarshalJSON(response))
}
