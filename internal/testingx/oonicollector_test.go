package testingx

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/url"
	"slices"
	"sync"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/must"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// This function tests the OONICollector type.
func TestOONICollector(t *testing.T) {
	t.Run("common: when method is not POST", func(t *testing.T) {
		// create and expose the testing collector
		collector := &OONICollector{}
		srv := MustNewHTTPServer(collector)
		defer srv.Close()

		// create request
		req := runtimex.Try1(http.NewRequest("GET", srv.URL, nil))

		// issue the request
		resp, err := http.DefaultClient.Do(req)

		// we don't expect error
		if err != nil {
			t.Fatal(err)
		}

		// make sure we close the body
		defer resp.Body.Close()

		// we expect to see 501
		if resp.StatusCode != http.StatusNotImplemented {
			t.Fatal("unexpected status code")
		}
	})

	t.Run("common: when the URL path does not start with report", func(t *testing.T) {
		// create and expose the testing collector
		collector := &OONICollector{}
		srv := MustNewHTTPServer(collector)
		defer srv.Close()

		// create request
		//
		// note: the server URL has / as its path so this URL is good to go
		req := runtimex.Try1(http.NewRequest("POST", srv.URL, nil))

		// issue the request
		resp, err := http.DefaultClient.Do(req)

		// we don't expect error
		if err != nil {
			t.Fatal(err)
		}

		// make sure we close the body
		defer resp.Body.Close()

		// we expect to see 400
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatal("unexpected status code")
		}
	})

	t.Run("common: when the Content-Type header is missing", func(t *testing.T) {
		// create and expose the testing collector
		collector := &OONICollector{}
		srv := MustNewHTTPServer(collector)
		defer srv.Close()

		// rewrite the URL to have a path starting with /report
		URL := runtimex.Try1(url.Parse(srv.URL))
		URL.Path = "/report"

		// create request
		//
		// note: the request has no Content-Type so we should be good here
		req := runtimex.Try1(http.NewRequest("POST", URL.String(), nil))

		// issue the request
		resp, err := http.DefaultClient.Do(req)

		// we don't expect error
		if err != nil {
			t.Fatal(err)
		}

		// make sure we close the body
		defer resp.Body.Close()

		// we expect to see 400
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatal("unexpected status code")
		}
	})

	t.Run("openReport: when we cannot unmarshal the body", func(t *testing.T) {
		// create and expose the testing collector
		collector := &OONICollector{}
		srv := MustNewHTTPServer(collector)
		defer srv.Close()

		// rewrite the URL to be exactly /report
		URL := runtimex.Try1(url.Parse(srv.URL))
		URL.Path = "/report"

		// create request
		//
		// note: the body is empty so parsing should fail
		req := runtimex.Try1(http.NewRequest("POST", URL.String(), nil))

		// make sure there's content-type
		req.Header.Set("Content-Type", "application/json")

		// issue the request
		resp, err := http.DefaultClient.Do(req)

		// we don't expect error
		if err != nil {
			t.Fatal(err)
		}

		// make sure we close the body
		defer resp.Body.Close()

		// we expect to see 400
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatal("unexpected status code")
		}
	})

	t.Run("openReport: with invalid data format version", func(t *testing.T) {
		// create and expose the testing collector
		collector := &OONICollector{}
		srv := MustNewHTTPServer(collector)
		defer srv.Close()

		// rewrite the URL to be exactly /report
		URL := runtimex.Try1(url.Parse(srv.URL))
		URL.Path = "/report"

		// create the request body
		//
		// note: we're using an invalid data format version to trigger error
		const invalidDataFormatVersion = "0.3.0"
		request := &model.OOAPIReportTemplate{
			DataFormatVersion: invalidDataFormatVersion,
		}
		rawreqbody := must.MarshalJSON(request)

		// create request
		req := runtimex.Try1(http.NewRequest("POST", URL.String(), bytes.NewReader(rawreqbody)))

		// make sure there's content-type
		req.Header.Set("Content-Type", "application/json")

		// issue the request
		resp, err := http.DefaultClient.Do(req)

		// we don't expect error
		if err != nil {
			t.Fatal(err)
		}

		// make sure we close the body
		defer resp.Body.Close()

		// we expect to see 400
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatal("unexpected status code")
		}
	})

	t.Run("openReport: with invalid format", func(t *testing.T) {
		// create and expose the testing collector
		collector := &OONICollector{}
		srv := MustNewHTTPServer(collector)
		defer srv.Close()

		// rewrite the URL to be exactly /report
		URL := runtimex.Try1(url.Parse(srv.URL))
		URL.Path = "/report"

		// create the request body
		//
		// note: we're using an invalid format to trigger error
		const validDataFormatVersion = "0.2.0"
		request := &model.OOAPIReportTemplate{
			DataFormatVersion: validDataFormatVersion,
			Format:            "yaml",
		}
		rawreqbody := must.MarshalJSON(request)

		// create request
		req := runtimex.Try1(http.NewRequest("POST", URL.String(), bytes.NewReader(rawreqbody)))

		// make sure there's content-type
		req.Header.Set("Content-Type", "application/json")

		// issue the request
		resp, err := http.DefaultClient.Do(req)

		// we don't expect error
		if err != nil {
			t.Fatal(err)
		}

		// make sure we close the body
		defer resp.Body.Close()

		// we expect to see 400
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatal("unexpected status code")
		}
	})

	t.Run("openReport: we can invoke a callback to see the incoming template", func(t *testing.T) {
		// create and expose the testing collector
		collector := &OONICollector{}
		srv := MustNewHTTPServer(collector)
		defer srv.Close()

		// rewrite the URL to be exactly /report
		URL := runtimex.Try1(url.Parse(srv.URL))
		URL.Path = "/report"

		// create the request body
		//
		// note: we're using the fake filler to randomly fill and then we're
		// editing to avoid failures, but this means we have a request that we
		// can later compare to using google/cmp-go/cmp.Diff.
		const validDataFormatVersion = "0.2.0"
		request := &model.OOAPIReportTemplate{}
		ff := &FakeFiller{}
		ff.Fill(&request)
		request.DataFormatVersion = validDataFormatVersion
		request.Format = "json"
		rawreqbody := must.MarshalJSON(request)

		// create request
		req := runtimex.Try1(http.NewRequest("POST", URL.String(), bytes.NewReader(rawreqbody)))

		// make sure there's content-type
		req.Header.Set("Content-Type", "application/json")

		// prepare to obtain the incoming report template
		mu := &sync.Mutex{}
		var incoming *model.OOAPIReportTemplate

		// set the callback to verify the request
		//
		// we save the original request and return an error to trigger failure
		collector.ValidateReportTemplate = func(rt *model.OOAPIReportTemplate) error {
			mu.Lock()
			incoming = rt
			mu.Unlock()
			return errors.New("mocked error")
		}

		// issue the request
		resp, err := http.DefaultClient.Do(req)

		// we don't expect error
		if err != nil {
			t.Fatal(err)
		}

		// make sure we close the body
		defer resp.Body.Close()

		// we expect to see 400
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatal("unexpected status code")
		}

		// make sure we got what we sent
		if diff := cmp.Diff(request, incoming); diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("openReport: we can edit the outgoing response", func(t *testing.T) {
		// create and expose the testing collector
		collector := &OONICollector{}
		srv := MustNewHTTPServer(collector)
		defer srv.Close()

		// rewrite the URL to be exactly /report
		URL := runtimex.Try1(url.Parse(srv.URL))
		URL.Path = "/report"

		// create the request body
		const validDataFormatVersion = "0.2.0"
		request := &model.OOAPIReportTemplate{}
		ff := &FakeFiller{}
		ff.Fill(&request)
		request.DataFormatVersion = validDataFormatVersion
		request.Format = "json"
		rawreqbody := must.MarshalJSON(request)

		// create request
		req := runtimex.Try1(http.NewRequest("POST", URL.String(), bytes.NewReader(rawreqbody)))

		// make sure there's content-type
		req.Header.Set("Content-Type", "application/json")

		// make sure we can edit the response
		collector.EditOpenReportResponse = func(resp *model.OOAPICollectorOpenResponse) {
			resp.BackendVersion = "antani-antani-antani"
		}

		// issue the request
		resp, err := http.DefaultClient.Do(req)

		// we don't expect error
		if err != nil {
			t.Fatal(err)
		}

		// make sure we close the body
		defer resp.Body.Close()

		// we expect to see 200
		if resp.StatusCode != http.StatusOK {
			t.Fatal("unexpected status code")
		}

		// read response body
		rawrespbody := runtimex.Try1(io.ReadAll(resp.Body))
		var response model.OOAPICollectorOpenResponse
		must.UnmarshalJSON(rawrespbody, &response)

		// make sure we've got the edited BackendVersion
		if response.BackendVersion != "antani-antani-antani" {
			t.Fatal("did not edit the response")
		}
	})

	t.Run("openReport: we get a reportID back and format=json", func(t *testing.T) {
		// create and expose the testing collector
		collector := &OONICollector{}
		srv := MustNewHTTPServer(collector)
		defer srv.Close()

		// rewrite the URL to be exactly /report
		URL := runtimex.Try1(url.Parse(srv.URL))
		URL.Path = "/report"

		// create the request body
		const validDataFormatVersion = "0.2.0"
		request := &model.OOAPIReportTemplate{}
		ff := &FakeFiller{}
		ff.Fill(&request)
		request.DataFormatVersion = validDataFormatVersion
		request.Format = "json"
		rawreqbody := must.MarshalJSON(request)

		// create request
		req := runtimex.Try1(http.NewRequest("POST", URL.String(), bytes.NewReader(rawreqbody)))

		// make sure there's content-type
		req.Header.Set("Content-Type", "application/json")

		// issue the request
		resp, err := http.DefaultClient.Do(req)

		// we don't expect error
		if err != nil {
			t.Fatal(err)
		}

		// make sure we close the body
		defer resp.Body.Close()

		// we expect to see 200
		if resp.StatusCode != http.StatusOK {
			t.Fatal("unexpected status code")
		}

		// read response body
		rawrespbody := runtimex.Try1(io.ReadAll(resp.Body))
		var response model.OOAPICollectorOpenResponse
		must.UnmarshalJSON(rawrespbody, &response)

		// make sure the fields are okay
		if response.BackendVersion != "1.3.0" {
			t.Fatal("unexpected backend version")
		}
		if response.ReportID == "" {
			t.Fatal("empty report ID")
		}
		if !slices.Contains(response.SupportedFormats, "json") {
			t.Fatal("SupportedFormats does not contain the json format")
		}
	})

	// This is a convenience function to open a report before submitting
	openreport := func(t *testing.T, stringURL string) (*model.OOAPIReportTemplate, *model.OOAPICollectorOpenResponse) {
		// rewrite the URL to be exactly /report
		URL := runtimex.Try1(url.Parse(stringURL))
		URL.Path = "/report"

		// create the request body
		const validDataFormatVersion = "0.2.0"
		request := &model.OOAPIReportTemplate{}
		ff := &FakeFiller{}
		ff.Fill(&request)
		request.DataFormatVersion = validDataFormatVersion
		request.Format = "json"
		rawreqbody := must.MarshalJSON(request)

		// create request
		req := runtimex.Try1(http.NewRequest("POST", URL.String(), bytes.NewReader(rawreqbody)))

		// make sure there's content-type
		req.Header.Set("Content-Type", "application/json")

		// issue the request
		resp, err := http.DefaultClient.Do(req)

		// we don't expect error
		if err != nil {
			t.Fatal(err)
		}

		// make sure we close the body
		defer resp.Body.Close()

		// we expect to see 200
		if resp.StatusCode != http.StatusOK {
			t.Fatal("unexpected status code")
		}

		// read response body
		rawrespbody := runtimex.Try1(io.ReadAll(resp.Body))
		var response model.OOAPICollectorOpenResponse
		must.UnmarshalJSON(rawrespbody, &response)

		return request, &response
	}

	t.Run("submit: when the report ID does not exist", func(t *testing.T) {
		// create and expose the testing collector
		collector := &OONICollector{}
		srv := MustNewHTTPServer(collector)
		defer srv.Close()

		// rewrite the URL to be exactly /report/${reportID}
		URL := runtimex.Try1(url.Parse(srv.URL))
		URL.Path = "/report/" + "blah"

		// create request
		//
		// note: the body is empty so parsing should fail
		req := runtimex.Try1(http.NewRequest("POST", URL.String(), nil))

		// make sure there's content-type
		req.Header.Set("Content-Type", "application/json")

		// issue the request
		resp, err := http.DefaultClient.Do(req)

		// we don't expect error
		if err != nil {
			t.Fatal(err)
		}

		// make sure we close the body
		defer resp.Body.Close()

		// we expect to see 400
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatal("unexpected status code")
		}
	})

	t.Run("submit: when we cannot unmarshal the body", func(t *testing.T) {
		// create and expose the testing collector
		collector := &OONICollector{}
		srv := MustNewHTTPServer(collector)
		defer srv.Close()

		// first of all let's open a report
		_, reportInfo := openreport(t, srv.URL)

		// rewrite the URL to be exactly /report/${reportID}
		URL := runtimex.Try1(url.Parse(srv.URL))
		URL.Path = "/report/" + reportInfo.ReportID

		// create request
		//
		// note: the body is empty so parsing should fail
		req := runtimex.Try1(http.NewRequest("POST", URL.String(), nil))

		// make sure there's content-type
		req.Header.Set("Content-Type", "application/json")

		// issue the request
		resp, err := http.DefaultClient.Do(req)

		// we don't expect error
		if err != nil {
			t.Fatal(err)
		}

		// make sure we close the body
		defer resp.Body.Close()

		// we expect to see 400
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatal("unexpected status code")
		}
	})

	t.Run("submit: with invalid format", func(t *testing.T) {
		// create and expose the testing collector
		collector := &OONICollector{}
		srv := MustNewHTTPServer(collector)
		defer srv.Close()

		// first of all let's open a report
		_, reportInfo := openreport(t, srv.URL)

		// rewrite the URL to be exactly /report/${reportID}
		URL := runtimex.Try1(url.Parse(srv.URL))
		URL.Path = "/report/" + reportInfo.ReportID

		// create the request body
		//
		// note: the YAML format here is invalid
		request := &model.OOAPICollectorUpdateRequest{
			Format:  "yaml",
			Content: nil,
		}
		rawrequestbody := must.MarshalJSON(request)

		// create request
		req := runtimex.Try1(http.NewRequest("POST", URL.String(), bytes.NewReader(rawrequestbody)))

		// make sure there's content-type
		req.Header.Set("Content-Type", "application/json")

		// issue the request
		resp, err := http.DefaultClient.Do(req)

		// we don't expect error
		if err != nil {
			t.Fatal(err)
		}

		// make sure we close the body
		defer resp.Body.Close()

		// we expect to see 400
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatal("unexpected status code")
		}
	})

	t.Run("submit: when we cannot unmarshal the nested inside body", func(t *testing.T) {
		// create and expose the testing collector
		collector := &OONICollector{}
		srv := MustNewHTTPServer(collector)
		defer srv.Close()

		// first of all let's open a report
		_, reportInfo := openreport(t, srv.URL)

		// rewrite the URL to be exactly /report/${reportID}
		URL := runtimex.Try1(url.Parse(srv.URL))
		URL.Path = "/report/" + reportInfo.ReportID

		// create the request body
		//
		// note: the content is empty so we cannot parse a JSON
		request := &model.OOAPICollectorUpdateRequest{
			Format:  "json",
			Content: "",
		}
		rawrequestbody := must.MarshalJSON(request)

		// create request
		req := runtimex.Try1(http.NewRequest("POST", URL.String(), bytes.NewReader(rawrequestbody)))

		// make sure there's content-type
		req.Header.Set("Content-Type", "application/json")

		// issue the request
		resp, err := http.DefaultClient.Do(req)

		// we don't expect error
		if err != nil {
			t.Fatal(err)
		}

		// make sure we close the body
		defer resp.Body.Close()

		// we expect to see 400
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatal("unexpected status code")
		}
	})

	t.Run("submit: when the template does not match the expectations", func(t *testing.T) {
		// create and expose the testing collector
		collector := &OONICollector{}
		srv := MustNewHTTPServer(collector)
		defer srv.Close()

		// first of all let's open a report
		template, reportInfo := openreport(t, srv.URL)

		// rewrite the URL to be exactly /report/${reportID}
		URL := runtimex.Try1(url.Parse(srv.URL))
		URL.Path = "/report/" + reportInfo.ReportID

		// create the measurement
		//
		// note: we're changing the test name here
		measurement := &model.Measurement{
			DataFormatVersion:    template.DataFormatVersion,
			InputHashes:          []string{},
			MeasurementStartTime: template.TestStartTime,
			ProbeASN:             template.ProbeASN,
			ProbeCC:              template.ProbeCC,
			ReportID:             reportInfo.ReportID,
			TestKeys:             nil,
			TestName:             template.TestName + "blahblah",
			TestStartTime:        template.TestStartTime,
			TestVersion:          template.TestVersion,
		}

		// create the request body
		request := &model.OOAPICollectorUpdateRequest{
			Format:  "json",
			Content: measurement,
		}
		rawrequestbody := must.MarshalJSON(request)

		// create request
		req := runtimex.Try1(http.NewRequest("POST", URL.String(), bytes.NewReader(rawrequestbody)))

		// make sure there's content-type
		req.Header.Set("Content-Type", "application/json")

		// issue the request
		resp, err := http.DefaultClient.Do(req)

		// we don't expect error
		if err != nil {
			t.Fatal(err)
		}

		// make sure we close the body
		defer resp.Body.Close()

		// we expect to see 400
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatal("unexpected status code")
		}
	})

	t.Run("submit: we can invoke a callback to further validate the measurement", func(t *testing.T) {
		// create and expose the testing collector
		collector := &OONICollector{}
		srv := MustNewHTTPServer(collector)
		defer srv.Close()

		// first of all let's open a report
		template, reportInfo := openreport(t, srv.URL)

		// rewrite the URL to be exactly /report/${reportID}
		URL := runtimex.Try1(url.Parse(srv.URL))
		URL.Path = "/report/" + reportInfo.ReportID

		// create the measurement
		measurement := &model.Measurement{
			DataFormatVersion:    template.DataFormatVersion,
			MeasurementStartTime: template.TestStartTime,
			ProbeASN:             template.ProbeASN,
			ProbeCC:              template.ProbeCC,
			ReportID:             reportInfo.ReportID,
			SoftwareName:         template.SoftwareName,
			SoftwareVersion:      template.SoftwareVersion,
			TestKeys:             nil,
			TestName:             template.TestName,
			TestStartTime:        template.TestStartTime,
			TestVersion:          template.TestVersion,
		}

		// create the request body
		request := &model.OOAPICollectorUpdateRequest{
			Format:  "json",
			Content: measurement,
		}
		rawrequestbody := must.MarshalJSON(request)

		// create request
		req := runtimex.Try1(http.NewRequest("POST", URL.String(), bytes.NewReader(rawrequestbody)))

		// make sure there's content-type
		req.Header.Set("Content-Type", "application/json")

		// prepare to obtain the incoming measurement
		mu := &sync.Mutex{}
		var incoming *model.Measurement

		// setup a callback so we can get the incoming measurement
		//
		// note: here we return an error to make the API fail but we save the measurement for later
		collector.ValidateMeasurement = func(meas *model.Measurement) error {
			mu.Lock()
			incoming = meas
			mu.Unlock()
			return errors.New("mocked error")
		}

		// issue the request
		resp, err := http.DefaultClient.Do(req)

		// we don't expect error
		if err != nil {
			t.Fatal(err)
		}

		// make sure we close the body
		defer resp.Body.Close()

		// we expect to see 400
		if resp.StatusCode != http.StatusBadRequest {
			t.Fatal("unexpected status code")
		}

		// make sure the measurement received by the API is the expected one
		if diff := cmp.Diff(measurement, incoming); diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("submit: we can edit the outgoing response", func(t *testing.T) {
		// create and expose the testing collector
		collector := &OONICollector{}
		srv := MustNewHTTPServer(collector)
		defer srv.Close()

		// first of all let's open a report
		template, reportInfo := openreport(t, srv.URL)

		// rewrite the URL to be exactly /report/${reportID}
		URL := runtimex.Try1(url.Parse(srv.URL))
		URL.Path = "/report/" + reportInfo.ReportID

		// create the measurement
		measurement := &model.Measurement{
			DataFormatVersion:    template.DataFormatVersion,
			MeasurementStartTime: template.TestStartTime,
			ProbeASN:             template.ProbeASN,
			ProbeCC:              template.ProbeCC,
			ReportID:             reportInfo.ReportID,
			SoftwareName:         template.SoftwareName,
			SoftwareVersion:      template.SoftwareVersion,
			TestKeys:             nil,
			TestName:             template.TestName,
			TestStartTime:        template.TestStartTime,
			TestVersion:          template.TestVersion,
		}

		// create the request body
		request := &model.OOAPICollectorUpdateRequest{
			Format:  "json",
			Content: measurement,
		}
		rawrequestbody := must.MarshalJSON(request)

		// create request
		req := runtimex.Try1(http.NewRequest("POST", URL.String(), bytes.NewReader(rawrequestbody)))

		// make sure there's content-type
		req.Header.Set("Content-Type", "application/json")

		// make sure we can edit the response
		collector.EditUpdateResponse = func(resp *model.OOAPICollectorUpdateResponse) {
			resp.MeasurementUID = "blablah"
		}

		// issue the request
		resp, err := http.DefaultClient.Do(req)

		// we don't expect error
		if err != nil {
			t.Fatal(err)
		}

		// make sure we close the body
		defer resp.Body.Close()

		// we expect to see 200
		if resp.StatusCode != http.StatusOK {
			t.Fatal("unexpected status code")
		}

		// read and parse response body
		rawrespbody := runtimex.Try1(io.ReadAll(resp.Body))
		var response model.OOAPICollectorUpdateResponse
		must.UnmarshalJSON(rawrespbody, &response)

		// make sure the measurement UID has been edited
		if response.MeasurementUID != "blablah" {
			t.Fatal("did not exit the measurement UID")
		}
	})

	t.Run("submit: we get a measurement ID back", func(t *testing.T) {
		// create and expose the testing collector
		collector := &OONICollector{}
		srv := MustNewHTTPServer(collector)
		defer srv.Close()

		// first of all let's open a report
		template, reportInfo := openreport(t, srv.URL)

		// rewrite the URL to be exactly /report/${reportID}
		URL := runtimex.Try1(url.Parse(srv.URL))
		URL.Path = "/report/" + reportInfo.ReportID

		// create the measurement
		measurement := &model.Measurement{
			DataFormatVersion:    template.DataFormatVersion,
			MeasurementStartTime: template.TestStartTime,
			ProbeASN:             template.ProbeASN,
			ProbeCC:              template.ProbeCC,
			ReportID:             reportInfo.ReportID,
			SoftwareName:         template.SoftwareName,
			SoftwareVersion:      template.SoftwareVersion,
			TestKeys:             nil,
			TestName:             template.TestName,
			TestStartTime:        template.TestStartTime,
			TestVersion:          template.TestVersion,
		}

		// create the request body
		request := &model.OOAPICollectorUpdateRequest{
			Format:  "json",
			Content: measurement,
		}
		rawrequestbody := must.MarshalJSON(request)

		// create request
		req := runtimex.Try1(http.NewRequest("POST", URL.String(), bytes.NewReader(rawrequestbody)))

		// make sure there's content-type
		req.Header.Set("Content-Type", "application/json")

		// issue the request
		resp, err := http.DefaultClient.Do(req)

		// we don't expect error
		if err != nil {
			t.Fatal(err)
		}

		// make sure we close the body
		defer resp.Body.Close()

		// we expect to see 200
		if resp.StatusCode != http.StatusOK {
			t.Fatal("unexpected status code")
		}

		// read and parse response body
		rawrespbody := runtimex.Try1(io.ReadAll(resp.Body))
		var response model.OOAPICollectorUpdateResponse
		must.UnmarshalJSON(rawrespbody, &response)

		// make sure the measurement UID is not empty
		if response.MeasurementUID == "" {
			t.Fatal("the measurement UID is unexpectedly empty")
		}
	})
}
