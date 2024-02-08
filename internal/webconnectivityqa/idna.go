package webconnectivityqa

import (
	"github.com/ooni/probe-cli/v3/internal/netemx"
)

// idnaWithoutCensorshipLowercase verifies that we can handle IDNA with lowercase.
func idnaWithoutCensorshipLowercase() *TestCase {
	return &TestCase{
		Name:  "idnaWithoutCensorshipLowercase",
		Flags: TestCaseFlagNoV04,
		Input: "http://яндекс.рф/",
		Configure: func(env *netemx.QAEnv) {
			// nothing
		},
		ExpectErr: false,
		ExpectTestKeys: &testKeys{
			DNSExperimentFailure:  nil,
			DNSConsistency:        "consistent",
			HTTPExperimentFailure: nil,
			BodyLengthMatch:       true,
			BodyProportion:        1,
			StatusCodeMatch:       true,
			HeadersMatch:          true,
			TitleMatch:            true,
			XBlockingFlags:        32, // AnalysisBlockingFlagSuccess
			Accessible:            true,
			Blocking:              false,
		},
	}
}

// idnaWithoutCensorshipWithFirstLetterUppercase verifies that we can handle IDNA
// with the first letter being uppercase.
func idnaWithoutCensorshipWithFirstLetterUppercase() *TestCase {
	return &TestCase{
		Name:  "idnaWithoutCensorshipWithFirstLetterUppercase",
		Flags: TestCaseFlagNoV04,
		Input: "http://Яндекс.рф/",
		Configure: func(env *netemx.QAEnv) {
			// nothing
		},
		ExpectErr: false,
		ExpectTestKeys: &testKeys{
			DNSExperimentFailure:  nil,
			DNSConsistency:        "consistent",
			HTTPExperimentFailure: nil,
			BodyLengthMatch:       true,
			BodyProportion:        1,
			StatusCodeMatch:       true,
			HeadersMatch:          true,
			TitleMatch:            true,
			XBlockingFlags:        32, // AnalysisBlockingFlagSuccess
			Accessible:            true,
			Blocking:              false,
		},
	}
}
