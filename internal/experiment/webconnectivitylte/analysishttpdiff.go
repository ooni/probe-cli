package webconnectivitylte

//
// HTTP diff analysis
//

import (
	"github.com/ooni/probe-cli/v3/internal/minipipeline"
	"github.com/ooni/probe-cli/v3/internal/optional"
)

// analysisHTTPDiffStatus contains the status relevant to compute HTTP diff.
type analysisHTTPDiffStatus struct {
	BodyProportion  optional.Value[float64] `json:"body_proportion"`
	BodyLengthMatch optional.Value[bool]    `json:"body_length_match"`
	HeadersMatch    optional.Value[bool]    `json:"headers_match"`
	StatusCodeMatch optional.Value[bool]    `json:"status_code_match"`
	TitleMatch      optional.Value[bool]    `json:"title_match"`
}

// newAnalysisHTTPDiffStatus constructs a new [*analysisHTTPDiffStatus].
func newAnalysisHTTPDiffStatus(analysis *minipipeline.WebAnalysis) *analysisHTTPDiffStatus {
	hds := &analysisHTTPDiffStatus{}

	// BodyProportion & BodyLengthMatch
	const bodyProportionFactor = 0.7
	if !analysis.HTTPFinalResponseDiffBodyProportionFactor.IsNone() {
		hds.BodyProportion = analysis.HTTPFinalResponseDiffBodyProportionFactor
		value := hds.BodyProportion.Unwrap() > bodyProportionFactor
		hds.BodyLengthMatch = optional.Some(value)
	}

	// HeadersMatch
	if !analysis.HTTPFinalResponseDiffUncommonHeadersIntersection.IsNone() {
		value := len(analysis.HTTPFinalResponseDiffUncommonHeadersIntersection.Unwrap()) > 0
		hds.HeadersMatch = optional.Some(value)
	}

	// StatusCodeMatch
	if !analysis.HTTPFinalResponseDiffStatusCodeMatch.IsNone() {
		value := analysis.HTTPFinalResponseDiffStatusCodeMatch.Unwrap()
		hds.StatusCodeMatch = optional.Some(value)
	}

	// TitleMatch
	if !analysis.HTTPFinalResponseDiffTitleDifferentLongWords.IsNone() {
		value := len(analysis.HTTPFinalResponseDiffTitleDifferentLongWords.Unwrap()) <= 0
		hds.TitleMatch = optional.Some(value)
	}

	return hds
}

// httpDiff computes whether there is HTTP diff.
func (hds *analysisHTTPDiffStatus) httpDiff() bool {
	if !hds.StatusCodeMatch.IsNone() && hds.StatusCodeMatch.Unwrap() {
		if !hds.BodyLengthMatch.IsNone() && hds.BodyLengthMatch.Unwrap() {
			return false
		}
		if !hds.HeadersMatch.IsNone() && hds.HeadersMatch.Unwrap() {
			return false
		}
		if !hds.TitleMatch.IsNone() && hds.TitleMatch.Unwrap() {
			return false
		}
		// fallthrough
	}
	return true
}
