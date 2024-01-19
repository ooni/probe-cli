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

type analysisHTTPDiffValuesProvider interface {
	bodyLengthMatch() optional.Value[bool]
	headersMatch() optional.Value[bool]
	statusCodeMatch() optional.Value[bool]
	titleMatch() optional.Value[bool]
}

var _ analysisHTTPDiffValuesProvider = &analysisHTTPDiffStatus{}

// bodyLengthMatch implements analysisHTTPDiffValuesProvider.
func (hds *analysisHTTPDiffStatus) bodyLengthMatch() optional.Value[bool] {
	return hds.BodyLengthMatch
}

// headersMatch implements analysisHTTPDiffValuesProvider.
func (hds *analysisHTTPDiffStatus) headersMatch() optional.Value[bool] {
	return hds.HeadersMatch
}

// statusCodeMatch implements analysisHTTPDiffValuesProvider.
func (hds *analysisHTTPDiffStatus) statusCodeMatch() optional.Value[bool] {
	return hds.StatusCodeMatch
}

// titleMatch implements analysisHTTPDiffValuesProvider.
func (hds *analysisHTTPDiffStatus) titleMatch() optional.Value[bool] {
	return hds.TitleMatch
}

// analysisHTTPDiffAlgorithm returns whether there's an HTTP diff
func analysisHTTPDiffAlgorithm(p analysisHTTPDiffValuesProvider) bool {
	if !p.statusCodeMatch().IsNone() && p.statusCodeMatch().Unwrap() {
		if !p.bodyLengthMatch().IsNone() && p.bodyLengthMatch().Unwrap() {
			return false
		}
		if !p.headersMatch().IsNone() && p.headersMatch().Unwrap() {
			return false
		}
		if !p.titleMatch().IsNone() && p.titleMatch().Unwrap() {
			return false
		}
		// fallthrough
	}
	return true
}
