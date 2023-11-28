package minipipeline

import (
	"testing"

	"github.com/ooni/probe-cli/v3/internal/optional"
)

func TestWebAnalysisComputeDNSExperimentFailure(t *testing.T) {
	t.Run("when there's no DNSDomain", func(t *testing.T) {
		container := &WebObservationsContainer{
			DNSLookupFailures: map[int64]*WebObservation{
				1: {
					DNSTransactionID: optional.Some(int64(1)),
					DNSDomain:        optional.None[string](), // explicitly set
					DNSLookupFailure: optional.Some("dns_no_answer"),
					DNSQueryType:     optional.Some("A"),
					DNSEngine:        optional.Some("getaddrinfo"),
				},
			},
		}

		wa := &WebAnalysis{}
		wa.ComputeDNSExperimentFailure(container)

		if !wa.DNSExperimentFailure.IsNone() {
			t.Fatal("DNSExperimentFailure is not none")
		}
	})

	t.Run("when DNSDomain does not match ControlDNSDomain", func(t *testing.T) {
		container := &WebObservationsContainer{
			DNSLookupFailures: map[int64]*WebObservation{
				1: {
					DNSTransactionID: optional.Some(int64(1)),
					DNSDomain:        optional.Some("dns.google.com"),
					DNSLookupFailure: optional.Some("dns_no_answer"),
					DNSQueryType:     optional.Some("A"),
					DNSEngine:        optional.Some("getaddrinfo"),
					ControlDNSDomain: optional.Some("dns.google"),
				},
			},
		}

		wa := &WebAnalysis{}
		wa.ComputeDNSExperimentFailure(container)

		if !wa.DNSExperimentFailure.IsNone() {
			t.Fatal("DNSExperimentFailure is not none")
		}
	})

	t.Run("when the failure is dns_no_answer for AAAA", func(t *testing.T) {
		container := &WebObservationsContainer{
			DNSLookupFailures: map[int64]*WebObservation{
				1: {
					DNSTransactionID: optional.Some(int64(1)),
					DNSDomain:        optional.Some("dns.google.com"),
					DNSLookupFailure: optional.Some("dns_no_answer"),
					DNSQueryType:     optional.Some("AAAA"),
					DNSEngine:        optional.Some("getaddrinfo"),
					ControlDNSDomain: optional.Some("dns.google.com"),
				},
			},
		}

		wa := &WebAnalysis{}
		wa.ComputeDNSExperimentFailure(container)

		if !wa.DNSExperimentFailure.IsNone() {
			t.Fatal("DNSExperimentFailure is not none")
		}
	})
}

func TestWebAnalysisComputeDNSTransactionsWithBogons(t *testing.T) {
	t.Run("when there's no IPAddressBogon", func(t *testing.T) {
		container := &WebObservationsContainer{
			DNSLookupSuccesses: []*WebObservation{
				{
					/* empty */
				},
			},
		}

		wa := &WebAnalysis{}
		wa.ComputeDNSTransactionsWithBogons(container)

		if v := wa.DNSTransactionsWithBogons.UnwrapOr(nil); len(v) != 0 {
			t.Fatal("DNSTransactionsWithBogons is not none")
		}
	})
}

func TestWebAnalysisComputeTCPTransactionsWithUnexpectedHTTPFailures(t *testing.T) {
	t.Run("when both measurement and control fail", func(t *testing.T) {
		container := &WebObservationsContainer{
			KnownTCPEndpoints: map[int64]*WebObservation{
				1: {
					HTTPFailure:        optional.Some("connection_reset"),
					ControlHTTPFailure: optional.Some("connection_reset"),
				},
			},
		}

		wa := &WebAnalysis{}
		wa.ComputeTCPTransactionsWithUnexpectedHTTPFailures(container)

		result := wa.TCPTransactionsWithUnexpectedHTTPFailures.Unwrap()
		if len(result) != 0 {
			t.Fatal("should not have added any entry")
		}
	})
}

func TestWebAnalysisComputeHTTPDiffBodyProportionFactor(t *testing.T) {
	t.Run("when there is no probe response body length", func(t *testing.T) {
		container := &WebObservationsContainer{
			KnownTCPEndpoints: map[int64]*WebObservation{
				1: {
					HTTPResponseIsFinal:    optional.Some(true),
					HTTPResponseBodyLength: optional.None[int64](),
				},
			},
		}

		wa := &WebAnalysis{}
		wa.ComputeHTTPDiffBodyProportionFactor(container)

		if !wa.HTTPDiffBodyProportionFactor.IsNone() {
			t.Fatal("should still be none")
		}
	})

	t.Run("when the probe response body length is negative", func(t *testing.T) {
		container := &WebObservationsContainer{
			KnownTCPEndpoints: map[int64]*WebObservation{
				1: {
					HTTPResponseIsFinal:    optional.Some(true),
					HTTPResponseBodyLength: optional.Some[int64](-1),
				},
			},
		}

		wa := &WebAnalysis{}
		wa.ComputeHTTPDiffBodyProportionFactor(container)

		if !wa.HTTPDiffBodyProportionFactor.IsNone() {
			t.Fatal("should still be none")
		}
	})

	t.Run("when the probe response body length is zero", func(t *testing.T) {
		container := &WebObservationsContainer{
			KnownTCPEndpoints: map[int64]*WebObservation{
				1: {
					HTTPResponseIsFinal:    optional.Some(true),
					HTTPResponseBodyLength: optional.Some[int64](0),
				},
			},
		}

		wa := &WebAnalysis{}
		wa.ComputeHTTPDiffBodyProportionFactor(container)

		if !wa.HTTPDiffBodyProportionFactor.IsNone() {
			t.Fatal("should still be none")
		}
	})

	t.Run("when the probe response body is truncated", func(t *testing.T) {
		container := &WebObservationsContainer{
			KnownTCPEndpoints: map[int64]*WebObservation{
				1: {
					HTTPResponseIsFinal:         optional.Some(true),
					HTTPResponseBodyLength:      optional.Some[int64](11),
					HTTPResponseBodyIsTruncated: optional.Some(true),
				},
			},
		}

		wa := &WebAnalysis{}
		wa.ComputeHTTPDiffBodyProportionFactor(container)

		if !wa.HTTPDiffBodyProportionFactor.IsNone() {
			t.Fatal("should still be none")
		}
	})
}

func TestWebAnalysisComputeHTTPDiffStatusCodeMatch(t *testing.T) {
	t.Run("when there is no probe status code", func(t *testing.T) {
		container := &WebObservationsContainer{
			KnownTCPEndpoints: map[int64]*WebObservation{
				1: {
					HTTPResponseIsFinal:    optional.Some(true),
					HTTPResponseStatusCode: optional.None[int64](),
				},
			},
		}

		wa := &WebAnalysis{}
		wa.ComputeHTTPDiffStatusCodeMatch(container)

		if !wa.HTTPDiffStatusCodeMatch.IsNone() {
			t.Fatal("should still be none")
		}
	})

	t.Run("when the probe status code is negative", func(t *testing.T) {
		container := &WebObservationsContainer{
			KnownTCPEndpoints: map[int64]*WebObservation{
				1: {
					HTTPResponseIsFinal:    optional.Some(true),
					HTTPResponseStatusCode: optional.Some[int64](-1),
				},
			},
		}

		wa := &WebAnalysis{}
		wa.ComputeHTTPDiffStatusCodeMatch(container)

		if !wa.HTTPDiffStatusCodeMatch.IsNone() {
			t.Fatal("should still be none")
		}
	})

	t.Run("when the probe status code is zero", func(t *testing.T) {
		container := &WebObservationsContainer{
			KnownTCPEndpoints: map[int64]*WebObservation{
				1: {
					HTTPResponseIsFinal:    optional.Some(true),
					HTTPResponseStatusCode: optional.Some[int64](0),
				},
			},
		}

		wa := &WebAnalysis{}
		wa.ComputeHTTPDiffStatusCodeMatch(container)

		if !wa.HTTPDiffStatusCodeMatch.IsNone() {
			t.Fatal("should still be none")
		}
	})

	t.Run("when there's status code mismatch and the control is not like 2xx", func(t *testing.T) {
		container := &WebObservationsContainer{
			KnownTCPEndpoints: map[int64]*WebObservation{
				1: {
					HTTPResponseIsFinal:           optional.Some(true),
					HTTPResponseStatusCode:        optional.Some[int64](403),
					ControlHTTPResponseStatusCode: optional.Some[int64](500),
				},
			},
		}

		wa := &WebAnalysis{}
		wa.ComputeHTTPDiffStatusCodeMatch(container)

		if !wa.HTTPDiffStatusCodeMatch.IsNone() {
			t.Fatal("should still be none")
		}
	})
}

func TestWebAnalysisComputeHTTPDiffTitleDifferentLongWords(t *testing.T) {
	t.Run("when there is no probe title", func(t *testing.T) {
		container := &WebObservationsContainer{
			KnownTCPEndpoints: map[int64]*WebObservation{
				1: {
					HTTPResponseIsFinal: optional.Some(true),
					HTTPResponseTitle:   optional.None[string](),
				},
			},
		}

		wa := &WebAnalysis{}
		wa.ComputeHTTPDiffTitleDifferentLongWords(container)

		if !wa.HTTPDiffTitleDifferentLongWords.IsNone() {
			t.Fatal("should still be none")
		}
	})

	t.Run("when the probe title is empty", func(t *testing.T) {
		container := &WebObservationsContainer{
			KnownTCPEndpoints: map[int64]*WebObservation{
				1: {
					HTTPResponseIsFinal: optional.Some(true),
					HTTPResponseTitle:   optional.Some[string](""),
				},
			},
		}

		wa := &WebAnalysis{}
		wa.ComputeHTTPDiffTitleDifferentLongWords(container)

		if !wa.HTTPDiffTitleDifferentLongWords.IsNone() {
			t.Fatal("should still be none")
		}
	})
}

func TestWebAnalysisComputeHTTPFinalResponses(t *testing.T) {
	t.Run("when there is no endpoint transaction ID", func(t *testing.T) {
		container := &WebObservationsContainer{
			KnownTCPEndpoints: map[int64]*WebObservation{
				1: {
					EndpointTransactionID: optional.None[int64](),
				},
			},
		}

		wa := &WebAnalysis{}
		wa.ComputeHTTPFinalResponses(container)

		if v := wa.HTTPFinalResponses.UnwrapOr(nil); len(v) > 0 {
			t.Fatal("should be empty")
		}
	})

	t.Run("when the endpoint transaction ID is negative", func(t *testing.T) {
		container := &WebObservationsContainer{
			KnownTCPEndpoints: map[int64]*WebObservation{
				1: {
					EndpointTransactionID: optional.Some[int64](-1),
				},
			},
		}

		wa := &WebAnalysis{}
		wa.ComputeHTTPFinalResponses(container)

		if v := wa.HTTPFinalResponses.UnwrapOr(nil); len(v) > 0 {
			t.Fatal("should be empty")
		}
	})

	t.Run("when the endpoint transaction ID is zero", func(t *testing.T) {
		container := &WebObservationsContainer{
			KnownTCPEndpoints: map[int64]*WebObservation{
				1: {
					EndpointTransactionID: optional.Some[int64](0),
				},
			},
		}

		wa := &WebAnalysis{}
		wa.ComputeHTTPFinalResponses(container)

		if v := wa.HTTPFinalResponses.UnwrapOr(nil); len(v) > 0 {
			t.Fatal("should be empty")
		}
	})
}

func TestWebAnalysisComputeTCPTransactionsWithUnexplainedUnexpectedFailures(t *testing.T) {
	t.Run("when we don't have a transaction ID", func(t *testing.T) {
		container := &WebObservationsContainer{
			KnownTCPEndpoints: map[int64]*WebObservation{
				1: {
					EndpointTransactionID: optional.None[int64](),
				},
			},
		}

		wa := &WebAnalysis{}
		wa.ComputeTCPTransactionsWithUnexplainedUnexpectedFailures(container)

		if v := wa.TCPTransactionsWithUnexplainedUnexpectedFailures.UnwrapOr(nil); len(v) > 0 {
			t.Fatal("should be empty")
		}
	})
}

func TestWebAnalysisComputeHTTPFinalResponsesWithTLS(t *testing.T) {
	t.Run("when there is no endpoint transaction ID", func(t *testing.T) {
		container := &WebObservationsContainer{
			KnownTCPEndpoints: map[int64]*WebObservation{
				1: {
					EndpointTransactionID: optional.None[int64](),
				},
			},
		}

		wa := &WebAnalysis{}
		wa.ComputeHTTPFinalResponsesWithTLS(container)

		if v := wa.HTTPFinalResponsesWithTLS.UnwrapOr(nil); len(v) > 0 {
			t.Fatal("should be empty")
		}
	})

	t.Run("when the endpoint transaction ID is negative", func(t *testing.T) {
		container := &WebObservationsContainer{
			KnownTCPEndpoints: map[int64]*WebObservation{
				1: {
					EndpointTransactionID: optional.Some[int64](-1),
				},
			},
		}

		wa := &WebAnalysis{}
		wa.ComputeHTTPFinalResponsesWithTLS(container)

		if v := wa.HTTPFinalResponsesWithTLS.UnwrapOr(nil); len(v) > 0 {
			t.Fatal("should be empty")
		}
	})

	t.Run("when the endpoint transaction ID is zero", func(t *testing.T) {
		container := &WebObservationsContainer{
			KnownTCPEndpoints: map[int64]*WebObservation{
				1: {
					EndpointTransactionID: optional.Some[int64](0),
				},
			},
		}

		wa := &WebAnalysis{}
		wa.ComputeHTTPFinalResponsesWithTLS(container)

		if v := wa.HTTPFinalResponsesWithTLS.UnwrapOr(nil); len(v) > 0 {
			t.Fatal("should be empty")
		}
	})
}
