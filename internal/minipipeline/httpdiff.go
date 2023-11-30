package minipipeline

import (
	"strings"

	"github.com/ooni/probe-cli/v3/internal/optional"
)

// ComputeHTTPDiffBodyProportionFactor computes the body proportion factor.
func ComputeHTTPDiffBodyProportionFactor(measurement, control int64) float64 {
	var proportion float64
	if measurement >= control {
		proportion = float64(control) / float64(measurement)
	} else {
		proportion = float64(measurement) / float64(control)
	}
	return proportion
}

// ComputeHTTPDiffStatusCodeMatch computes whether the status code matches.
func ComputeHTTPDiffStatusCodeMatch(measurement, control int64) optional.Value[bool] {
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
		return optional.None[bool]()
	}
	return optional.Some(good)
}

var httpDiffCommonHeaders = map[string]bool{
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

// ComputeHTTPDiffUncommonHeadersIntersection computes the uncommon header intersection.
func ComputeHTTPDiffUncommonHeadersIntersection(measurement, control map[string]bool) map[string]bool {
	state := make(map[string]bool)

	const (
		byProbe = 1 << iota
		byTH
	)

	matching := make(map[string]int64)
	for key := range measurement {
		key = strings.ToLower(key)
		if _, ok := httpDiffCommonHeaders[key]; !ok {
			matching[key] |= byProbe
		}
	}

	for key := range control {
		key = strings.ToLower(key)
		if _, ok := httpDiffCommonHeaders[key]; !ok {
			matching[key] |= byTH
		}
	}

	// compute the intersection of uncommon headers
	for key, value := range matching {
		if (value & (byProbe | byTH)) == (byProbe | byTH) {
			state[key] = true
		}
	}

	return state
}

// ComputeHTTPDiffTitleDifferentLongWords computes the different long words
// in the title (a long word is a word longer than 5 chars).
func ComputeHTTPDiffTitleDifferentLongWords(measurement, control string) map[string]bool {
	state := make(map[string]bool)

	const (
		byProbe = 1 << iota
		byTH
	)

	// Implementation note
	//
	// We don't consider to match words that are shorter than 5
	// characters (5 is the average word length for english)
	//
	// The original implementation considered the word order but
	// considering different languages it seems we could have less
	// false positives by ignoring the word order.
	words := make(map[string]int64)
	const minWordLength = 5
	for _, word := range strings.Split(measurement, " ") {
		if len(word) >= minWordLength {
			words[strings.ToLower(word)] |= byProbe
		}
	}
	for _, word := range strings.Split(control, " ") {
		if len(word) >= minWordLength {
			words[strings.ToLower(word)] |= byTH
		}
	}

	// compute the list of long words that do not appear in both titles
	for word, score := range words {
		if (score & (byProbe | byTH)) != (byProbe | byTH) {
			state[word] = true
		}
	}

	return state
}
