package webconnectivity

import (
	"reflect"
	"strings"

	"github.com/ooni/probe-cli/v3/internal/experiment/urlgetter"
	"github.com/ooni/probe-cli/v3/internal/experiment/webconnectivity/internal"
	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// HTTPAnalysisResult contains the results of the analysis performed on the
// client. We obtain it by comparing the measurement and the control.
type HTTPAnalysisResult struct {
	BodyLengthMatch *bool   `json:"body_length_match"`
	BodyProportion  float64 `json:"body_proportion"`
	StatusCodeMatch *bool   `json:"status_code_match"`
	HeadersMatch    *bool   `json:"headers_match"`
	TitleMatch      *bool   `json:"title_match"`
}

// Log logs the results of the analysis
func (har HTTPAnalysisResult) Log(logger model.Logger) {
	logger.Infof("BodyLengthMatch: %+v", internal.BoolPointerToString(har.BodyLengthMatch))
	logger.Infof("BodyProportion: %+v", har.BodyProportion)
	logger.Infof("StatusCodeMatch: %+v", internal.BoolPointerToString(har.StatusCodeMatch))
	logger.Infof("HeadersMatch: %+v", internal.BoolPointerToString(har.HeadersMatch))
	logger.Infof("TitleMatch: %+v", internal.BoolPointerToString(har.TitleMatch))
}

// HTTPAnalysis performs follow-up analysis on the webconnectivity measurement by
// comparing the measurement test keys and the control.
func HTTPAnalysis(tk urlgetter.TestKeys, ctrl ControlResponse) (out HTTPAnalysisResult) {
	out.BodyLengthMatch, out.BodyProportion = HTTPBodyLengthChecks(tk, ctrl)
	out.StatusCodeMatch = HTTPStatusCodeMatch(tk, ctrl)
	out.HeadersMatch = HTTPHeadersMatch(tk, ctrl)
	out.TitleMatch = HTTPTitleMatch(tk, ctrl)
	return
}

// HTTPBodyLengthChecks returns whether the measured body is reasonably
// long as much as the control body as well as the proportion between
// the two bodies. This check may return nil, nil when such a
// comparison would actually not be applicable.
func HTTPBodyLengthChecks(
	tk urlgetter.TestKeys, ctrl ControlResponse) (match *bool, proportion float64) {
	control := ctrl.HTTPRequest.BodyLength
	if control <= 0 {
		return
	}
	if len(tk.Requests) <= 0 {
		return
	}
	response := tk.Requests[0].Response
	if response.BodyIsTruncated {
		return
	}
	measurement := int64(len(response.Body.Value))
	if measurement <= 0 {
		return
	}
	const bodyProportionFactor = 0.7
	if measurement >= control {
		proportion = float64(control) / float64(measurement)
	} else {
		proportion = float64(measurement) / float64(control)
	}
	v := proportion > bodyProportionFactor
	match = &v
	return
}

// HTTPStatusCodeMatch returns whether the status code of the measurement
// matches the status code of the control, or nil if such comparison
// is actually not applicable.
func HTTPStatusCodeMatch(tk urlgetter.TestKeys, ctrl ControlResponse) (out *bool) {
	control := ctrl.HTTPRequest.StatusCode
	if len(tk.Requests) < 1 {
		return // no real status code
	}
	measurement := tk.Requests[0].Response.Code
	if control <= 0 {
		return // no real status code
	}
	if measurement <= 0 {
		return // no real status code
	}
	value := control == measurement
	if value {
		// if the status codes are equal, they clearly match
		out = &value
		return
	}
	// This fix is part of Web Connectivity in MK and in Python since
	// basically forever; my recollection is that we want to work around
	// cases where the test helper is failing(?!). Unlike previous
	// implementations, this implementation avoids a false positive
	// when both measurement and control statuses are 500.
	if control/100 == 5 {
		return
	}
	out = &value
	return
}

// HTTPHeadersMatch returns whether uncommon headers match between control and
// measurement, or nil if check is not applicable.
func HTTPHeadersMatch(tk urlgetter.TestKeys, ctrl ControlResponse) *bool {
	if len(tk.Requests) <= 0 {
		return nil
	}
	if tk.Requests[0].Response.Code <= 0 {
		return nil
	}
	if ctrl.HTTPRequest.StatusCode <= 0 {
		return nil
	}
	control := ctrl.HTTPRequest.Headers
	// Implementation note: using map because we only care about the
	// keys being different and we ignore the values.
	measurement := tk.Requests[0].Response.Headers
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
		return &good
	}
	// compute the intersection of uncommon headers
	var intersection int
	for _, value := range matching {
		if (value & inBoth) == inBoth {
			intersection++
		}
	}
	good := intersection > 0
	return &good
}

// HTTPTitleMatch returns whether the measurement and the control titles
// reasonably match, or nil if not applicable.
func HTTPTitleMatch(tk urlgetter.TestKeys, ctrl ControlResponse) (out *bool) {
	if len(tk.Requests) <= 0 {
		return
	}
	response := tk.Requests[0].Response
	if response.Code <= 0 {
		return
	}
	if response.BodyIsTruncated {
		return
	}
	if ctrl.HTTPRequest.StatusCode <= 0 {
		return
	}
	control := ctrl.HTTPRequest.Title
	measurementBody := response.Body.Value
	measurement := measurexlite.WebGetTitle(measurementBody)
	if measurement == "" {
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
	return &good
}
