package minipipeline

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/optional"
)

func TestHTTPDiffBodyProportionFactor(t *testing.T) {
	type testcase struct {
		name                          string
		ControlHTTPResponseBodyLength optional.Value[int64]
		HTTPResponseIsFinal           optional.Value[bool]
		HTTPResponseBodyLength        optional.Value[int64]
		HTTPResponseBodyIsTruncated   optional.Value[bool]
		ExpectReturnValue             int64
		ExpectBodyProportionFactor    optional.Value[float64]
	}

	allcases := []testcase{{
		name:                          "with missing information on whether the WebObservation is final",
		ControlHTTPResponseBodyLength: optional.None[int64](),
		HTTPResponseIsFinal:           optional.None[bool](),
		HTTPResponseBodyLength:        optional.None[int64](),
		HTTPResponseBodyIsTruncated:   optional.None[bool](),
		ExpectReturnValue:             -1,
		ExpectBodyProportionFactor:    optional.None[float64](),
	}, {
		name:                          "with non-final WebObservation",
		ControlHTTPResponseBodyLength: optional.None[int64](),
		HTTPResponseIsFinal:           optional.Some[bool](false),
		HTTPResponseBodyLength:        optional.None[int64](),
		HTTPResponseBodyIsTruncated:   optional.None[bool](),
		ExpectReturnValue:             -1,
		ExpectBodyProportionFactor:    optional.None[float64](),
	}, {
		name:                          "with missing response body length",
		ControlHTTPResponseBodyLength: optional.None[int64](),
		HTTPResponseIsFinal:           optional.Some(true),
		HTTPResponseBodyLength:        optional.None[int64](),
		HTTPResponseBodyIsTruncated:   optional.None[bool](),
		ExpectReturnValue:             -2,
		ExpectBodyProportionFactor:    optional.None[float64](),
	}, {
		name:                          "with response body length being negative",
		ControlHTTPResponseBodyLength: optional.None[int64](),
		HTTPResponseIsFinal:           optional.Some(true),
		HTTPResponseBodyLength:        optional.Some[int64](-1),
		HTTPResponseBodyIsTruncated:   optional.None[bool](),
		ExpectReturnValue:             -2,
		ExpectBodyProportionFactor:    optional.None[float64](),
	}, {
		name:                          "with response body length being zero",
		ControlHTTPResponseBodyLength: optional.None[int64](),
		HTTPResponseIsFinal:           optional.Some(true),
		HTTPResponseBodyLength:        optional.Some[int64](0),
		HTTPResponseBodyIsTruncated:   optional.None[bool](),
		ExpectReturnValue:             -2,
		ExpectBodyProportionFactor:    optional.None[float64](),
	}, {
		name:                          "with no information on whether the body is truncated",
		ControlHTTPResponseBodyLength: optional.None[int64](),
		HTTPResponseIsFinal:           optional.Some(true),
		HTTPResponseBodyLength:        optional.Some[int64](1024),
		HTTPResponseBodyIsTruncated:   optional.None[bool](),
		ExpectReturnValue:             -3,
		ExpectBodyProportionFactor:    optional.None[float64](),
	}, {
		name:                          "with truncated response body",
		ControlHTTPResponseBodyLength: optional.None[int64](),
		HTTPResponseIsFinal:           optional.Some(true),
		HTTPResponseBodyLength:        optional.Some[int64](1024),
		HTTPResponseBodyIsTruncated:   optional.Some(true),
		ExpectReturnValue:             -3,
		ExpectBodyProportionFactor:    optional.None[float64](),
	}, {
		name:                          "with missing control response body length",
		ControlHTTPResponseBodyLength: optional.None[int64](),
		HTTPResponseIsFinal:           optional.Some(true),
		HTTPResponseBodyLength:        optional.Some[int64](1024),
		HTTPResponseBodyIsTruncated:   optional.Some(false),
		ExpectReturnValue:             -4,
		ExpectBodyProportionFactor:    optional.None[float64](),
	}, {
		name:                          "with control response body length being negative",
		ControlHTTPResponseBodyLength: optional.Some[int64](-1),
		HTTPResponseIsFinal:           optional.Some(true),
		HTTPResponseBodyLength:        optional.Some[int64](1024),
		HTTPResponseBodyIsTruncated:   optional.Some(false),
		ExpectReturnValue:             -4,
		ExpectBodyProportionFactor:    optional.None[float64](),
	}, {
		name:                          "with control response body length being zero",
		ControlHTTPResponseBodyLength: optional.Some[int64](0),
		HTTPResponseIsFinal:           optional.Some(true),
		HTTPResponseBodyLength:        optional.Some[int64](1024),
		HTTPResponseBodyIsTruncated:   optional.Some(false),
		ExpectReturnValue:             -4,
		ExpectBodyProportionFactor:    optional.None[float64](),
	}, {
		name:                          "successful case",
		ControlHTTPResponseBodyLength: optional.Some[int64](2048),
		HTTPResponseIsFinal:           optional.Some(true),
		HTTPResponseBodyLength:        optional.Some[int64](1024),
		HTTPResponseBodyIsTruncated:   optional.Some[bool](false),
		ExpectReturnValue:             0,
		ExpectBodyProportionFactor:    optional.Some(0.5),
	}}

	for _, tc := range allcases {
		t.Run(tc.name, func(t *testing.T) {
			obs := &WebObservation{
				HTTPResponseBodyLength:        tc.HTTPResponseBodyLength,
				HTTPResponseBodyIsTruncated:   tc.HTTPResponseBodyIsTruncated,
				HTTPResponseIsFinal:           tc.HTTPResponseIsFinal,
				ControlHTTPResponseBodyLength: tc.ControlHTTPResponseBodyLength,
			}

			wa := &WebAnalysis{}
			retval := wa.httpDiffBodyProportionFactor(obs)
			if diff := cmp.Diff(tc.ExpectReturnValue, retval); diff != "" {
				t.Fatal(diff)
			}

			result := wa.HTTPFinalResponseDiffBodyProportionFactor
			switch {
			case tc.ExpectBodyProportionFactor.IsNone() && result.IsNone():
				// all good here
			case tc.ExpectBodyProportionFactor.IsNone() && !result.IsNone():
				t.Fatal("expected none, got", result.Unwrap())
			case !tc.ExpectBodyProportionFactor.IsNone() && result.IsNone():
				t.Fatal("expected", tc.ExpectBodyProportionFactor.Unwrap(), "got none")
			case !tc.ExpectBodyProportionFactor.IsNone() && !result.IsNone():
				if diff := cmp.Diff(tc.ExpectBodyProportionFactor.Unwrap(), result.Unwrap()); diff != "" {
					t.Fatal(diff)
				}
			}
		})
	}
}

func TestHTTPDiffStatusCodeMatch(t *testing.T) {
	type testcase struct {
		name                          string
		ControlHTTPResponseStatusCode optional.Value[int64]
		HTTPResponseIsFinal           optional.Value[bool]
		HTTPResponseStatusCode        optional.Value[int64]
		ExpectReturnValue             int64
		ExpectStatusCodeMatch         optional.Value[bool]
	}

	allcases := []testcase{{
		name:                          "with missing information on whether the WebObservation is final",
		ControlHTTPResponseStatusCode: optional.None[int64](),
		HTTPResponseIsFinal:           optional.None[bool](),
		HTTPResponseStatusCode:        optional.None[int64](),
		ExpectReturnValue:             -1,
		ExpectStatusCodeMatch:         optional.None[bool](),
	}, {
		name:                          "with non-final WebObservation",
		ControlHTTPResponseStatusCode: optional.None[int64](),
		HTTPResponseIsFinal:           optional.Some(false),
		HTTPResponseStatusCode:        optional.None[int64](),
		ExpectReturnValue:             -1,
		ExpectStatusCodeMatch:         optional.None[bool](),
	}, {
		name:                          "with missing response status code",
		ControlHTTPResponseStatusCode: optional.None[int64](),
		HTTPResponseIsFinal:           optional.Some(true),
		HTTPResponseStatusCode:        optional.None[int64](),
		ExpectReturnValue:             -2,
		ExpectStatusCodeMatch:         optional.None[bool](),
	}, {
		name:                          "with negative response status code",
		ControlHTTPResponseStatusCode: optional.None[int64](),
		HTTPResponseIsFinal:           optional.Some(true),
		HTTPResponseStatusCode:        optional.Some[int64](-1),
		ExpectReturnValue:             -2,
		ExpectStatusCodeMatch:         optional.None[bool](),
	}, {
		name:                          "with zero response status code",
		ControlHTTPResponseStatusCode: optional.None[int64](),
		HTTPResponseIsFinal:           optional.Some(true),
		HTTPResponseStatusCode:        optional.Some[int64](0),
		ExpectReturnValue:             -2,
		ExpectStatusCodeMatch:         optional.None[bool](),
	}, {
		name:                          "with missing control response status code",
		ControlHTTPResponseStatusCode: optional.None[int64](),
		HTTPResponseIsFinal:           optional.Some(true),
		HTTPResponseStatusCode:        optional.Some[int64](200),
		ExpectReturnValue:             -3,
		ExpectStatusCodeMatch:         optional.None[bool](),
	}, {
		name:                          "with negative control response status code",
		ControlHTTPResponseStatusCode: optional.Some[int64](-1),
		HTTPResponseIsFinal:           optional.Some(true),
		HTTPResponseStatusCode:        optional.Some[int64](200),
		ExpectReturnValue:             -3,
		ExpectStatusCodeMatch:         optional.None[bool](),
	}, {
		name:                          "with zero control response status code",
		ControlHTTPResponseStatusCode: optional.Some[int64](0),
		HTTPResponseIsFinal:           optional.Some(true),
		HTTPResponseStatusCode:        optional.Some[int64](200),
		ExpectReturnValue:             -3,
		ExpectStatusCodeMatch:         optional.None[bool](),
	}, {
		name:                          "successful case with match",
		ControlHTTPResponseStatusCode: optional.Some[int64](200),
		HTTPResponseIsFinal:           optional.Some(true),
		HTTPResponseStatusCode:        optional.Some[int64](200),
		ExpectReturnValue:             0,
		ExpectStatusCodeMatch:         optional.Some[bool](true),
	}, {
		name:                          "successful case with mismatch",
		ControlHTTPResponseStatusCode: optional.Some[int64](200),
		HTTPResponseIsFinal:           optional.Some(true),
		HTTPResponseStatusCode:        optional.Some[int64](500),
		ExpectReturnValue:             0,
		ExpectStatusCodeMatch:         optional.Some[bool](false),
	}, {
		name:                          "successful case with mismatch and 5xx control",
		ControlHTTPResponseStatusCode: optional.Some[int64](500),
		HTTPResponseIsFinal:           optional.Some(true),
		HTTPResponseStatusCode:        optional.Some[int64](403),
		ExpectReturnValue:             0,
		ExpectStatusCodeMatch:         optional.None[bool](),
	}}

	for _, tc := range allcases {
		t.Run(tc.name, func(t *testing.T) {
			obs := &WebObservation{
				HTTPResponseStatusCode:        tc.HTTPResponseStatusCode,
				HTTPResponseIsFinal:           tc.HTTPResponseIsFinal,
				ControlHTTPResponseStatusCode: tc.ControlHTTPResponseStatusCode,
			}

			wa := &WebAnalysis{}
			retval := wa.httpDiffStatusCodeMatch(obs)
			if diff := cmp.Diff(tc.ExpectReturnValue, retval); diff != "" {
				t.Fatal(diff)
			}

			result := wa.HTTPFinalResponseDiffStatusCodeMatch
			switch {
			case tc.ExpectStatusCodeMatch.IsNone() && result.IsNone():
				// all good here
			case tc.ExpectStatusCodeMatch.IsNone() && !result.IsNone():
				t.Fatal("expected none, got", result.Unwrap())
			case !tc.ExpectStatusCodeMatch.IsNone() && result.IsNone():
				t.Fatal("expected", tc.ExpectStatusCodeMatch.Unwrap(), "got none")
			case !tc.ExpectStatusCodeMatch.IsNone() && !result.IsNone():
				if diff := cmp.Diff(tc.ExpectStatusCodeMatch.Unwrap(), result.Unwrap()); diff != "" {
					t.Fatal(diff)
				}
			}
		})
	}
}

func TestHTTPDiffUncommonHeadersIntersection(t *testing.T) {
	type testcase struct {
		name                          string
		ControlHTTPResponseStatusCode optional.Value[int64]
		ControlHTTPResponseHeaderKeys optional.Value[map[string]bool]
		HTTPResponseIsFinal           optional.Value[bool]
		HTTPResponseHeaderKeys        optional.Value[map[string]bool]
		ExpectReturnValue             int64
		ExpectHeadersIntersection     optional.Value[map[string]bool]
	}

	allcases := []testcase{{
		name:                          "when we don't know whether the WebObservation is final",
		ControlHTTPResponseStatusCode: optional.None[int64](),
		ControlHTTPResponseHeaderKeys: optional.None[map[string]bool](),
		HTTPResponseIsFinal:           optional.None[bool](),
		HTTPResponseHeaderKeys:        optional.None[map[string]bool](),
		ExpectReturnValue:             -1,
		ExpectHeadersIntersection:     optional.None[map[string]bool](),
	}, {
		name:                          "when the WebObservation is not final",
		ControlHTTPResponseStatusCode: optional.None[int64](),
		ControlHTTPResponseHeaderKeys: optional.None[map[string]bool](),
		HTTPResponseIsFinal:           optional.Some[bool](false),
		HTTPResponseHeaderKeys:        optional.None[map[string]bool](),
		ExpectReturnValue:             -1,
		ExpectHeadersIntersection:     optional.None[map[string]bool](),
	}, {
		name:                          "when we don't know the control status code",
		ControlHTTPResponseStatusCode: optional.None[int64](),
		ControlHTTPResponseHeaderKeys: optional.None[map[string]bool](),
		HTTPResponseIsFinal:           optional.Some[bool](true),
		HTTPResponseHeaderKeys:        optional.None[map[string]bool](),
		ExpectReturnValue:             -2,
		ExpectHeadersIntersection:     optional.None[map[string]bool](),
	}, {
		name:                          "when the control status code is negative",
		ControlHTTPResponseStatusCode: optional.Some[int64](-1),
		ControlHTTPResponseHeaderKeys: optional.None[map[string]bool](),
		HTTPResponseIsFinal:           optional.Some[bool](true),
		HTTPResponseHeaderKeys:        optional.None[map[string]bool](),
		ExpectReturnValue:             -2,
		ExpectHeadersIntersection:     optional.None[map[string]bool](),
	}, {
		name:                          "when the control status code is zero",
		ControlHTTPResponseStatusCode: optional.Some[int64](0),
		ControlHTTPResponseHeaderKeys: optional.None[map[string]bool](),
		HTTPResponseIsFinal:           optional.Some[bool](true),
		HTTPResponseHeaderKeys:        optional.None[map[string]bool](),
		ExpectReturnValue:             -2,
		ExpectHeadersIntersection:     optional.None[map[string]bool](),
	}, {
		name:                          "with missing headers information",
		ControlHTTPResponseStatusCode: optional.Some[int64](200),
		ControlHTTPResponseHeaderKeys: optional.None[map[string]bool](),
		HTTPResponseIsFinal:           optional.Some[bool](true),
		HTTPResponseHeaderKeys:        optional.None[map[string]bool](),
		ExpectReturnValue:             0,
		ExpectHeadersIntersection:     optional.Some(map[string]bool{}),
	}, {
		name:                          "with no headers intersection",
		ControlHTTPResponseStatusCode: optional.Some[int64](200),
		ControlHTTPResponseHeaderKeys: optional.Some(map[string]bool{"A": true}), // uncommon header
		HTTPResponseIsFinal:           optional.Some[bool](true),
		HTTPResponseHeaderKeys:        optional.Some(map[string]bool{"B": true}), // uncommon header
		ExpectReturnValue:             0,
		ExpectHeadersIntersection:     optional.Some(map[string]bool{}),
	}, {
		name:                          "with headers intersection",
		ControlHTTPResponseStatusCode: optional.Some[int64](200),
		ControlHTTPResponseHeaderKeys: optional.Some(map[string]bool{"C": true}), // uncommon header
		HTTPResponseIsFinal:           optional.Some[bool](true),
		HTTPResponseHeaderKeys:        optional.Some(map[string]bool{"C": true}), // uncommon header
		ExpectReturnValue:             1,
		ExpectHeadersIntersection:     optional.Some(map[string]bool{"c": true}), // lowercase b/c it's normalized
	}}

	for _, tc := range allcases {
		t.Run(tc.name, func(t *testing.T) {
			obs := &WebObservation{
				ControlHTTPResponseStatusCode:  tc.ControlHTTPResponseStatusCode,
				ControlHTTPResponseHeadersKeys: tc.ControlHTTPResponseHeaderKeys,
				HTTPResponseIsFinal:            tc.HTTPResponseIsFinal,
				HTTPResponseHeadersKeys:        tc.HTTPResponseHeaderKeys,
			}

			wa := &WebAnalysis{}
			retval := wa.httpDiffUncommonHeadersIntersection(obs)
			if diff := cmp.Diff(tc.ExpectReturnValue, retval); diff != "" {
				t.Fatal(diff)
			}

			result := wa.HTTPFinalResponseDiffUncommonHeadersIntersection
			switch {
			case tc.ExpectHeadersIntersection.IsNone() && result.IsNone():
				// all good here
			case tc.ExpectHeadersIntersection.IsNone() && !result.IsNone():
				t.Fatal("expected none, got", result.Unwrap())
			case !tc.ExpectHeadersIntersection.IsNone() && result.IsNone():
				t.Fatal("expected", tc.ExpectHeadersIntersection.Unwrap(), "got none")
			case !tc.ExpectHeadersIntersection.IsNone() && !result.IsNone():
				if diff := cmp.Diff(tc.ExpectHeadersIntersection.Unwrap(), result.Unwrap()); diff != "" {
					t.Fatal(diff)
				}
			}
		})
	}
}

func TestHTTPDiffTitleDifferentLongWords(t *testing.T) {
	type testcase struct {
		name                          string
		ControlHTTPResponseStatusCode optional.Value[int64]
		ControlHTTPResponseTitle      optional.Value[string]
		HTTPResponseIsFinal           optional.Value[bool]
		HTTPResponseTitle             optional.Value[string]
		ExpectReturnValue             int64
		ExpectDifferentLongWords      optional.Value[map[string]bool]
	}

	allcases := []testcase{{
		name:                          "with no information on whether the observation is final",
		ControlHTTPResponseStatusCode: optional.None[int64](),
		ControlHTTPResponseTitle:      optional.None[string](),
		HTTPResponseIsFinal:           optional.None[bool](),
		HTTPResponseTitle:             optional.None[string](),
		ExpectReturnValue:             -1,
		ExpectDifferentLongWords:      optional.None[map[string]bool](),
	}, {
		name:                          "with non-final response",
		ControlHTTPResponseStatusCode: optional.None[int64](),
		ControlHTTPResponseTitle:      optional.None[string](),
		HTTPResponseIsFinal:           optional.Some(false),
		HTTPResponseTitle:             optional.None[string](),
		ExpectReturnValue:             -1,
		ExpectDifferentLongWords:      optional.None[map[string]bool](),
	}, {
		name:                          "with no information on control status code",
		ControlHTTPResponseStatusCode: optional.None[int64](),
		ControlHTTPResponseTitle:      optional.None[string](),
		HTTPResponseIsFinal:           optional.Some(true),
		HTTPResponseTitle:             optional.None[string](),
		ExpectReturnValue:             -2,
		ExpectDifferentLongWords:      optional.None[map[string]bool](),
	}, {
		name:                          "with negative control status code",
		ControlHTTPResponseStatusCode: optional.Some[int64](-1),
		ControlHTTPResponseTitle:      optional.None[string](),
		HTTPResponseIsFinal:           optional.Some(true),
		HTTPResponseTitle:             optional.None[string](),
		ExpectReturnValue:             -2,
		ExpectDifferentLongWords:      optional.None[map[string]bool](),
	}, {
		name:                          "with zero control status code",
		ControlHTTPResponseStatusCode: optional.Some[int64](0),
		ControlHTTPResponseTitle:      optional.None[string](),
		HTTPResponseIsFinal:           optional.Some(true),
		HTTPResponseTitle:             optional.None[string](),
		ExpectReturnValue:             -2,
		ExpectDifferentLongWords:      optional.None[map[string]bool](),
	}, {
		name:                          "with no titles",
		ControlHTTPResponseStatusCode: optional.Some[int64](200),
		ControlHTTPResponseTitle:      optional.None[string](),
		HTTPResponseIsFinal:           optional.Some(true),
		HTTPResponseTitle:             optional.None[string](),
		ExpectReturnValue:             0,
		ExpectDifferentLongWords:      optional.Some(map[string]bool{}),
	}, {
		name:                          "with no different long words",
		ControlHTTPResponseStatusCode: optional.Some[int64](200),
		ControlHTTPResponseTitle:      optional.Some("Antani Mascetti Melandri"),
		HTTPResponseIsFinal:           optional.Some(true),
		HTTPResponseTitle:             optional.Some("Mascetti Melandri Antani"),
		ExpectReturnValue:             0,
		ExpectDifferentLongWords:      optional.Some(map[string]bool{}),
	}, {
		name:                          "with different long words",
		ControlHTTPResponseStatusCode: optional.Some[int64](200),
		ControlHTTPResponseTitle:      optional.Some("Antani Mascetti Melandri"),
		HTTPResponseIsFinal:           optional.Some(true),
		HTTPResponseTitle:             optional.Some("Forbidden Mascetti"),
		ExpectReturnValue:             3,
		ExpectDifferentLongWords: optional.Some(map[string]bool{
			"antani":    true,
			"forbidden": true,
			"melandri":  true,
		}),
	}}

	for _, tc := range allcases {
		t.Run(tc.name, func(t *testing.T) {
			obs := &WebObservation{
				ControlHTTPResponseStatusCode: tc.ControlHTTPResponseStatusCode,
				ControlHTTPResponseTitle:      tc.ControlHTTPResponseTitle,
				HTTPResponseIsFinal:           tc.HTTPResponseIsFinal,
				HTTPResponseTitle:             tc.HTTPResponseTitle,
			}

			wa := &WebAnalysis{}
			retval := wa.httpDiffTitleDifferentLongWords(obs)
			if diff := cmp.Diff(tc.ExpectReturnValue, retval); diff != "" {
				t.Fatal(diff)
			}

			result := wa.HTTPFinalResponseDiffTitleDifferentLongWords
			switch {
			case tc.ExpectDifferentLongWords.IsNone() && result.IsNone():
				// all good here
			case tc.ExpectDifferentLongWords.IsNone() && !result.IsNone():
				t.Fatal("expected none, got", result.Unwrap())
			case !tc.ExpectDifferentLongWords.IsNone() && result.IsNone():
				t.Fatal("expected", tc.ExpectDifferentLongWords.Unwrap(), "got none")
			case !tc.ExpectDifferentLongWords.IsNone() && !result.IsNone():
				if diff := cmp.Diff(tc.ExpectDifferentLongWords.Unwrap(), result.Unwrap()); diff != "" {
					t.Fatal(diff)
				}
			}
		})
	}
}
