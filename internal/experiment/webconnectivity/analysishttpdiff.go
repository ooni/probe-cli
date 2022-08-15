package webconnectivity

//
// HTTP diff analysis
//

import (
	"net/url"
	"reflect"
	"strings"

	"github.com/ooni/probe-cli/v3/internal/engine/experiment/webconnectivity"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// analysisHTTPDiff computes the HTTP diff between the final request-response
// observed by the probe and the TH's result. The caller is responsible of passing
// us a valid probe observation and a valid TH observation.
func (tk *TestKeys) analysisHTTPDiff(
	probe *model.ArchivalHTTPRequestResult, th *webconnectivity.ControlHTTPRequestResult) {

	// if we're dealing with an HTTPS request, don't perform any comparison
	// under the assumption that we're good if we're using TLS
	URL, err := url.Parse(probe.Request.URL)
	if err != nil {
		return // looks like a bug
	}
	accessibleTrue := true
	if URL.Scheme == "https" {
		tk.Accessible = &accessibleTrue
		return
	}

	// original HTTP diff algorithm adapted for this implementation
	tk.httpDiffBodyLengthChecks(probe, th)
	tk.httpDiffStatusCodeMatch(probe, th)
	tk.httpDiffHeadersMatch(probe, th)
	tk.httpDiffTitleMatch(probe, th)

	if tk.StatusCodeMatch != nil && *tk.StatusCodeMatch {
		if tk.BodyLengthMatch != nil && *tk.BodyLengthMatch {
			tk.Accessible = &accessibleTrue
			return
		}
		if tk.HeadersMatch != nil && *tk.HeadersMatch {
			tk.Accessible = &accessibleTrue
			return
		}
		if tk.TitleMatch != nil && *tk.TitleMatch {
			tk.Accessible = &accessibleTrue
			return
		}
	}

	tk.BlockingFlags |= analysisBlockingHTTPDiff
	accessibleFalse := false
	tk.Accessible = &accessibleFalse
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
	if control/100 != 2 {
		return // avoid comparison if it seems the TH failed
	}
	good := control == measurement
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
	measurement := webconnectivity.GetTitle(measurementBody)
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
