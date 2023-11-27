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
					DNSTransactionIDs: optional.Some([]int64{1}),
					DNSDomain:         optional.None[string](), // explicitly set
					DNSLookupFailure:  optional.Some("dns_no_answer"),
					DNSQueryType:      optional.Some("A"),
					DNSEngine:         optional.Some("getaddrinfo"),
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
					DNSTransactionIDs: optional.Some([]int64{1}),
					DNSDomain:         optional.Some("dns.google.com"),
					DNSLookupFailure:  optional.Some("dns_no_answer"),
					DNSQueryType:      optional.Some("A"),
					DNSEngine:         optional.Some("getaddrinfo"),
					ControlDNSDomain:  optional.Some("dns.google"),
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
					DNSTransactionIDs: optional.Some([]int64{1}),
					DNSDomain:         optional.Some("dns.google.com"),
					DNSLookupFailure:  optional.Some("dns_no_answer"),
					DNSQueryType:      optional.Some("AAAA"),
					DNSEngine:         optional.Some("getaddrinfo"),
					ControlDNSDomain:  optional.Some("dns.google.com"),
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
