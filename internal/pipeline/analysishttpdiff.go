package pipeline

import (
	"strings"

	"github.com/ooni/probe-cli/v3/internal/optional"
)

func (ax *Analysis) httpDiffHelper(db *DB, fx func(probeFR *WebEndpointObservation, thFR *WebObservationTH)) {
	// skip if there's no final request
	if db.WebFinalRequest.IsNone() {
		return
	}
	probeFR := db.WebFinalRequest.Unwrap()

	// skip if the HTTP failure is not defined (bug?)
	if probeFR.HTTPFailure.IsNone() {
		return
	}

	// skip if the final request failed
	probeFailure := probeFR.HTTPFailure.Unwrap()
	if probeFailure != "" {
		return
	}

	// skip if the final request is not defined for the TH
	if db.THWeb.IsNone() {
		return
	}
	thFR := db.THWeb.Unwrap()

	// skip if the failure is not defined for the TH
	if thFR.HTTPFailure.IsNone() {
		return
	}

	// skip if also the TH's HTTP request failed
	thFailure := thFR.HTTPFailure.Unwrap()
	if thFailure != "" {
		return
	}

	// invoke user defined func
	fx(probeFR, thFR)
}

// ComputeHTTPDiffBodyProportionFactor computes HTTPDiffBodyProportionFactor.
func (ax *Analysis) ComputeHTTPDiffBodyProportionFactor(db *DB) {
	ax.httpDiffHelper(db, func(probeFT *WebEndpointObservation, thFR *WebObservationTH) {
		// skip if there's no length for the TH
		if thFR.HTTPResponseBodyLength.IsNone() {
			return
		}

		// skip if the length has not been computed by the TH
		control := thFR.HTTPResponseBodyLength.Unwrap()
		if control <= 0 {
			return
		}

		// skip if we don't know whether the body was truncated
		if probeFT.HTTPResponseBodyIsTruncated.IsNone() {
			return
		}
		truncated := probeFT.HTTPResponseBodyIsTruncated.Unwrap()

		// skip if the body was truncated (we cannot trust length in this case)
		if truncated {
			return
		}

		// skip if we don't know the body length
		if probeFT.HTTPResponseBodyLength.IsNone() {
			return
		}
		measurement := probeFT.HTTPResponseBodyLength.Unwrap()

		// skip if the length is zero or negative (which doesn't make sense and seems a bug)
		if measurement <= 0 {
			return
		}

		// compute the body proportion factor
		var proportion float64
		if measurement >= control {
			proportion = float64(control) / float64(measurement)
		} else {
			proportion = float64(measurement) / float64(control)
		}

		// save the body proportion factor
		ax.HTTPDiffBodyProportionFactor = optional.Some(proportion)
	})
}

// ComputeHTTPDiffStatusCodeMatch computes HTTPDiffStatusCodeMatch.
func (ax *Analysis) ComputeHTTPDiffStatusCodeMatch(db *DB) {
	ax.httpDiffHelper(db, func(probeFR *WebEndpointObservation, thFR *WebObservationTH) {
		// skip if we don't know the control status
		if thFR.HTTPResponseStatusCode.IsNone() {
			return
		}
		control := thFR.HTTPResponseStatusCode.Unwrap()

		// skip the control is invalid
		if control <= 0 {
			return
		}

		// skip if we don't know the probe status
		if probeFR.HTTPResponseStatusCode.IsNone() {
			return
		}
		measurement := probeFR.HTTPResponseStatusCode.Unwrap()

		// skip if the meaasurement is invalid
		if measurement <= 0 {
			return
		}

		// compute whether there's a match including caveats
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
			// when both the TH and the probe failed equally. See
			// https://github.com/ooni/probe/issues/2287, which refers
			// to a measurement where both the probe and the TH fail
			// with 404, but we fail to say "status_code_match = true".
			//
			// See https://explorer.ooni.org/measurement/20220911T203447Z_webconnectivity_IT_30722_n1_YDZQZOHAziEJk6o9?input=http%3A%2F%2Fwww.webbox.com%2Findex.php
			// for a measurement where this was fixed.
			return
		}

		// store the algorithm result
		ax.HTTPDiffStatusCodeMatch = optional.Some(good)
	})
}

// ComputeHTTPDiffUncommonHeadersMatch computes HTTPDiffUncommonHeadersMatch.
func (ax *Analysis) ComputeHTTPDiffUncommonHeadersMatch(db *DB) {
	ax.httpDiffHelper(db, func(probeFR *WebEndpointObservation, thFR *WebObservationTH) {
		// skip if we don't know the control headers keys
		if len(thFR.HTTPResponseHeadersKeys) <= 0 {
			return
		}
		control := thFR.HTTPResponseHeadersKeys

		// skip if we don't know the probe headers keys
		if len(probeFR.HTTPResponseHeadersKeys) <= 0 {
			return
		}
		measurement := probeFR.HTTPResponseHeadersKeys

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

		matching := make(map[string]Origin)
		for key := range measurement {
			key = strings.ToLower(key)
			if _, ok := commonHeaders[key]; !ok {
				matching[key] |= OriginProbe
			}
		}

		for key := range control {
			key = strings.ToLower(key)
			if _, ok := commonHeaders[key]; !ok {
				matching[key] |= OriginTH
			}
		}

		// compute the intersection of uncommon headers
		found := false
		for _, value := range matching {
			if (value & (OriginProbe | OriginTH)) == (OriginProbe | OriginTH) {
				found = true
				break
			}
		}

		// store the result
		ax.HTTPDiffUncommonHeadersMatch = optional.Some(found)
	})
}

// ComputeHTTPDiffTitleMatch computes HTTPDiffTitleMatch.
func (ax *Analysis) ComputeHTTPDiffTitleMatch(db *DB) {
	ax.httpDiffHelper(db, func(probeFR *WebEndpointObservation, thFR *WebObservationTH) {
		// skip if we don't know the TH title
		if thFR.HTTPResponseTitle.IsNone() {
			return
		}
		control := thFR.HTTPResponseTitle.Unwrap()

		// skip if we don't know the probe title
		if probeFR.HTTPResponseTitle.IsNone() {
			return
		}
		measurement := probeFR.HTTPResponseTitle.Unwrap()

		if control == "" || measurement == "" {
			return
		}

		words := make(map[string]Origin)
		// We don't consider to match words that are shorter than 5
		// characters (5 is the average word length for english)
		//
		// The original implementation considered the word order but
		// considering different languages it seems we could have less
		// false positives by ignoring the word order.
		const minWordLength = 5
		for _, word := range strings.Split(measurement, " ") {
			if len(word) >= minWordLength {
				words[strings.ToLower(word)] |= OriginProbe
			}
		}
		for _, word := range strings.Split(control, " ") {
			if len(word) >= minWordLength {
				words[strings.ToLower(word)] |= OriginTH
			}
		}

		// check whether there's a long word that does not match
		good := true
		for _, score := range words {
			if (score & (OriginProbe | OriginTH)) != (OriginProbe | OriginTH) {
				good = false
				break
			}
		}

		// store the results
		ax.HTTPDiffTitleMatch = optional.Some(good)
	})
}
