package webconnectivity

//
// HTTP diff analysis
//

import (
	"net/url"
	"reflect"
	"strings"

	"github.com/ooni/probe-cli/v3/internal/engine/experiment/webconnectivity"
	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// analysisHTTPDiff computes the HTTP diff between the final request-response
// observed by the probe and the TH's result. The caller is responsible of passing
// us a valid probe observation and a valid TH observation with nil failure.
func (tk *TestKeys) analysisHTTPDiff(logger model.Logger,
	probe *model.ArchivalHTTPRequestResult, th *webconnectivity.ControlHTTPRequestResult) {
	// make sure the caller respected the contract
	runtimex.PanicIfTrue(
		probe.Failure != nil || th.Failure != nil,
		"the caller should have passed us successful HTTP observations",
	)

	// if we're dealing with an HTTPS request, don't perform any comparison
	// under the assumption that we're good if we're using TLS
	URL, err := url.Parse(probe.Request.URL)
	if err != nil {
		return // looks like a bug
	}
	if URL.Scheme == "https" {
		logger.Infof("HTTP: HTTPS && no error => #%d is successful", probe.TransactionID)
		tk.BlockingFlags |= analysisFlagSuccess
		return
	}

	// original HTTP diff algorithm adapted for this implementation
	tk.httpDiffBodyLengthChecks(probe, th)
	tk.httpDiffStatusCodeMatch(probe, th)
	tk.httpDiffHeadersMatch(probe, th)
	tk.httpDiffTitleMatch(probe, th)

	if tk.StatusCodeMatch != nil && *tk.StatusCodeMatch {
		if tk.BodyLengthMatch != nil && *tk.BodyLengthMatch {
			logger.Infof(
				"HTTP: statusCodeMatch && bodyLengthMatch => #%d is successful",
				probe.TransactionID,
			)
			tk.BlockingFlags |= analysisFlagSuccess
			return
		}
		if tk.HeadersMatch != nil && *tk.HeadersMatch {
			logger.Infof(
				"HTTP: statusCodeMatch && headersMatch => #%d is successful",
				probe.TransactionID,
			)
			tk.BlockingFlags |= analysisFlagSuccess
			return
		}
		if tk.TitleMatch != nil && *tk.TitleMatch {
			logger.Infof(
				"HTTP: statusCodeMatch && titleMatch => #%d is successful",
				probe.TransactionID,
			)
			tk.BlockingFlags |= analysisFlagSuccess
			return
		}
	}

	tk.BlockingFlags |= analysisFlagHTTPDiff
	logger.Warnf("HTTP: it seems #%d is a case of httpDiff", probe.TransactionID)
}

// httpDiffBodyLengthChecks compares the bodies lengths.
func (tk *TestKeys) httpDiffBodyLengthChecks(
	probe *model.ArchivalHTTPRequestResult, ctrl *webconnectivity.ControlHTTPRequestResult) {
	control := ctrl.BodyLength
	if control <= 0 {
		return // no actual length
	}
	response := probe.Response
	if response.BodyIsTruncated {
		return // cannot trust body length in this case
	}
	measurement := int64(len(response.Body.Value))
	if measurement <= 0 {
		return // no actual length
	}
	const bodyProportionFactor = 0.7
	var proportion float64
	if measurement >= control {
		proportion = float64(control) / float64(measurement)
	} else {
		proportion = float64(measurement) / float64(control)
	}
	good := proportion > bodyProportionFactor
	tk.BodyLengthMatch = &good
}

// httpDiffStatusCodeMatch compares the status codes.
func (tk *TestKeys) httpDiffStatusCodeMatch(
	probe *model.ArchivalHTTPRequestResult, ctrl *webconnectivity.ControlHTTPRequestResult) {
	control := ctrl.StatusCode
	measurement := probe.Response.Code
	if control <= 0 {
		return // no real status code
	}
	if measurement <= 0 {
		return // no real status code
	}
	good := control == measurement
	if !good && control/100 != 2 {
		// Avoid comparison if it seems the TH failed _and_ the two
		// status codes are not equal. Originally, this algorithm was
		// https://github.com/measurement-kit/measurement-kit/blob/b55fbecb205be62c736249b689df0c45ae342804/src/libmeasurement_kit/ooni/web_connectivity.cpp#L60
		// and excluded the case where the TH failed with 5xx.
		//
		// Then, we discovered when implementing websteps a bunch
		// of control failure modes that suggested to be more
		// cautious. See https://github.com/bassosimone/websteps-illustrated/blob/632f27443ab9d94fb05efcf5e0b0c1ce190221e2/internal/engine/experiment/websteps/analysisweb.go#L137.
		//
		// However, it seems a bit retarded to avoid comparison
		// when both the TH and the probe failed equallty. See
		// https://github.com/ooni/probe/issues/2287, which refers
		// to a measurement where both the probe and the TH fail
		// with 404, but we fail to say "status_code_match = true".
		//
		// See https://explorer.ooni.org/measurement/20220911T203447Z_webconnectivity_IT_30722_n1_YDZQZOHAziEJk6o9?input=http%3A%2F%2Fwww.webbox.com%2Findex.php
		// for a measurement where this was fixed.
		return
	}
	tk.StatusCodeMatch = &good
}

// httpDiffHeadersMatch compares the uncommon headers.
func (tk *TestKeys) httpDiffHeadersMatch(
	probe *model.ArchivalHTTPRequestResult, ctrl *webconnectivity.ControlHTTPRequestResult) {
	control := ctrl.Headers
	measurement := probe.Response.Headers
	if len(control) <= 0 || len(measurement) <= 0 {
		return
	}
	// Implementation note: using map because we only care about the
	// keys being different and we ignore the values.
	const (
		inMeasurement = 1 << 0
		inControl     = 1 << 1
		inBoth        = inMeasurement | inControl
	)
	commonHeaders := map[string]bool{
		"date":                      true,
		"content-type":              true,
		"server":                    true,
		"cache-control":             true,
		"vary":                      true,
		"set-cookie":                true,
		"location":                  true,
		"expires":                   true,
		"x-powered-by":              true,
		"content-encoding":          true,
		"last-modified":             true,
		"accept-ranges":             true,
		"pragma":                    true,
		"x-frame-options":           true,
		"etag":                      true,
		"x-content-type-options":    true,
		"age":                       true,
		"via":                       true,
		"p3p":                       true,
		"x-xss-protection":          true,
		"content-language":          true,
		"cf-ray":                    true,
		"strict-transport-security": true,
		"link":                      true,
		"x-varnish":                 true,
	}
	matching := make(map[string]int)
	ours := make(map[string]bool)
	for key := range measurement {
		key = strings.ToLower(key)
		if _, ok := commonHeaders[key]; !ok {
			matching[key] |= inMeasurement
		}
		ours[key] = true
	}
	theirs := make(map[string]bool)
	for key := range control {
		key = strings.ToLower(key)
		if _, ok := commonHeaders[key]; !ok {
			matching[key] |= inControl
		}
		theirs[key] = true
	}
	// if they are equal we're done
	if good := reflect.DeepEqual(ours, theirs); good {
		tk.HeadersMatch = &good
		return
	}
	// compute the intersection of uncommon headers
	found := false
	for _, value := range matching {
		if (value & inBoth) == inBoth {
			found = true
			break
		}
	}
	tk.HeadersMatch = &found
}

// httpDiffTitleMatch compares the titles.
func (tk *TestKeys) httpDiffTitleMatch(
	probe *model.ArchivalHTTPRequestResult, ctrl *webconnectivity.ControlHTTPRequestResult) {
	response := probe.Response
	if response.Code <= 0 {
		return
	}
	if response.BodyIsTruncated {
		return
	}
	if ctrl.StatusCode <= 0 {
		return
	}
	control := ctrl.Title
	measurementBody := response.Body.Value
	measurement := measurexlite.WebGetTitle(measurementBody)
	if control == "" || measurement == "" {
		return
	}
	const (
		inMeasurement = 1 << 0
		inControl     = 1 << 1
		inBoth        = inMeasurement | inControl
	)
	words := make(map[string]int)
	// We don't consider to match words that are shorter than 5
	// characters (5 is the average word length for english)
	//
	// The original implementation considered the word order but
	// considering different languages it seems we could have less
	// false positives by ignoring the word order.
	const minWordLength = 5
	for _, word := range strings.Split(measurement, " ") {
		if len(word) >= minWordLength {
			words[strings.ToLower(word)] |= inMeasurement
		}
	}
	for _, word := range strings.Split(control, " ") {
		if len(word) >= minWordLength {
			words[strings.ToLower(word)] |= inControl
		}
	}
	good := true
	for _, score := range words {
		if (score & inBoth) != inBoth {
			good = false
			break
		}
	}
	tk.TitleMatch = &good
}
