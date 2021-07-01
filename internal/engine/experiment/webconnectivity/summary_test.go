package webconnectivity_test

import (
	"io"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/webconnectivity"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/archival"
	"github.com/ooni/probe-cli/v3/internal/errorsx"
)

func TestSummarize(t *testing.T) {
	var (
		genericFailure         = io.EOF.Error()
		dns                    = "dns"
		falseValue             = false
		httpDiff               = "http-diff"
		httpFailure            = "http-failure"
		nilstring              *string
		probeConnectionRefused = errorsx.FailureConnectionRefused
		probeConnectionReset   = errorsx.FailureConnectionReset
		probeEOFError          = errorsx.FailureEOFError
		probeNXDOMAIN          = errorsx.FailureDNSNXDOMAINError
		probeTimeout           = errorsx.FailureGenericTimeoutError
		probeSSLInvalidHost    = errorsx.FailureSSLInvalidHostname
		probeSSLInvalidCert    = errorsx.FailureSSLInvalidCertificate
		probeSSLUnknownAuth    = errorsx.FailureSSLUnknownAuthority
		tcpIP                  = "tcp_ip"
		trueValue              = true
	)
	type args struct {
		tk *webconnectivity.TestKeys
	}
	tests := []struct {
		name    string
		args    args
		wantOut webconnectivity.Summary
	}{{
		name: "with an HTTPS request with no failure",
		args: args{
			tk: &webconnectivity.TestKeys{
				Requests: []archival.RequestEntry{{
					Request: archival.HTTPRequest{
						URL: "https://www.kernel.org/",
					},
					Failure: nil,
				}},
			},
		},
		wantOut: webconnectivity.Summary{
			BlockingReason: nil,
			Blocking:       false,
			Accessible:     &trueValue,
			Status:         webconnectivity.StatusSuccessSecure,
		},
	}, {
		name: "with failure in contacting the control",
		args: args{
			tk: &webconnectivity.TestKeys{
				ControlFailure: &genericFailure,
			},
		},
		wantOut: webconnectivity.Summary{
			BlockingReason: nil,
			Blocking:       nilstring,
			Accessible:     nil,
			Status:         webconnectivity.StatusAnomalyControlUnreachable,
		},
	}, {
		name: "with non-existing website",
		args: args{
			tk: &webconnectivity.TestKeys{
				DNSExperimentFailure: &probeNXDOMAIN,
				DNSAnalysisResult: webconnectivity.DNSAnalysisResult{
					DNSConsistency: &webconnectivity.DNSConsistent,
				},
			},
		},
		wantOut: webconnectivity.Summary{
			BlockingReason: nil,
			Blocking:       false,
			Accessible:     &trueValue,
			Status: webconnectivity.StatusSuccessNXDOMAIN |
				webconnectivity.StatusExperimentDNS,
		},
	}, {
		name: "with NXDOMAIN measured only by the probe",
		args: args{
			tk: &webconnectivity.TestKeys{
				DNSExperimentFailure: &probeNXDOMAIN,
				DNSAnalysisResult: webconnectivity.DNSAnalysisResult{
					DNSConsistency: &webconnectivity.DNSInconsistent,
				},
			},
		},
		wantOut: webconnectivity.Summary{
			BlockingReason: &dns,
			Blocking:       &dns,
			Accessible:     &falseValue,
			Status: webconnectivity.StatusAnomalyDNS |
				webconnectivity.StatusExperimentDNS,
		},
	}, {
		name: "with TCP total failure and consistent DNS",
		args: args{
			tk: &webconnectivity.TestKeys{
				DNSAnalysisResult: webconnectivity.DNSAnalysisResult{
					DNSConsistency: &webconnectivity.DNSConsistent,
				},
				TCPConnectAttempts:  7,
				TCPConnectSuccesses: 0,
			},
		},
		wantOut: webconnectivity.Summary{
			BlockingReason: &tcpIP,
			Blocking:       &tcpIP,
			Accessible:     &falseValue,
			Status: webconnectivity.StatusAnomalyConnect |
				webconnectivity.StatusExperimentConnect,
		},
	}, {
		name: "with TCP total failure and inconsistent DNS",
		args: args{
			tk: &webconnectivity.TestKeys{
				DNSAnalysisResult: webconnectivity.DNSAnalysisResult{
					DNSConsistency: &webconnectivity.DNSInconsistent,
				},
				TCPConnectAttempts:  7,
				TCPConnectSuccesses: 0,
			},
		},
		wantOut: webconnectivity.Summary{
			BlockingReason: &dns,
			Blocking:       &dns,
			Accessible:     &falseValue,
			Status: webconnectivity.StatusAnomalyConnect |
				webconnectivity.StatusExperimentConnect |
				webconnectivity.StatusAnomalyDNS,
		},
	}, {
		name: "with TCP total failure and unexpected DNS consistency",
		args: args{
			tk: &webconnectivity.TestKeys{
				DNSAnalysisResult: webconnectivity.DNSAnalysisResult{
					DNSConsistency: func() *string {
						s := "ANTANI"
						return &s
					}(),
				},
				TCPConnectAttempts:  7,
				TCPConnectSuccesses: 0,
			},
		},
		wantOut: webconnectivity.Summary{
			BlockingReason: nil,
			Blocking:       nilstring,
			Accessible:     nil,
			Status: webconnectivity.StatusAnomalyConnect |
				webconnectivity.StatusExperimentConnect |
				webconnectivity.StatusAnomalyUnknown,
		},
	}, {
		name: "with failed control HTTP request",
		args: args{
			tk: &webconnectivity.TestKeys{
				Control: webconnectivity.ControlResponse{
					HTTPRequest: webconnectivity.ControlHTTPRequestResult{
						Failure: &genericFailure,
					},
				},
			},
		},
		wantOut: webconnectivity.Summary{
			BlockingReason: nil,
			Blocking:       nilstring,
			Accessible:     nil,
			Status:         webconnectivity.StatusAnomalyControlFailure,
		},
	}, {
		name: "with less that one request entry",
		args: args{
			tk: &webconnectivity.TestKeys{},
		},
		wantOut: webconnectivity.Summary{
			BlockingReason: nil,
			Blocking:       nilstring,
			Accessible:     nil,
			Status:         webconnectivity.StatusBugNoRequests,
		},
	}, {
		name: "with connection refused",
		args: args{
			tk: &webconnectivity.TestKeys{
				Requests: []archival.RequestEntry{{
					Failure: &probeConnectionRefused,
				}},
			},
		},
		wantOut: webconnectivity.Summary{
			BlockingReason: &httpFailure,
			Blocking:       &httpFailure,
			Accessible:     &falseValue,
			Status: webconnectivity.StatusExperimentHTTP |
				webconnectivity.StatusAnomalyConnect,
		},
	}, {
		name: "with connection reset",
		args: args{
			tk: &webconnectivity.TestKeys{
				Requests: []archival.RequestEntry{{
					Failure: &probeConnectionReset,
				}},
			},
		},
		wantOut: webconnectivity.Summary{
			BlockingReason: &httpFailure,
			Blocking:       &httpFailure,
			Accessible:     &falseValue,
			Status: webconnectivity.StatusExperimentHTTP |
				webconnectivity.StatusAnomalyReadWrite,
		},
	}, {
		name: "with NXDOMAIN",
		args: args{
			tk: &webconnectivity.TestKeys{
				Requests: []archival.RequestEntry{{
					Failure: &probeNXDOMAIN,
				}},
			},
		},
		wantOut: webconnectivity.Summary{
			BlockingReason: &dns,
			Blocking:       &dns,
			Accessible:     &falseValue,
			Status: webconnectivity.StatusExperimentHTTP |
				webconnectivity.StatusAnomalyDNS,
		},
	}, {
		name: "with EOF",
		args: args{
			tk: &webconnectivity.TestKeys{
				Requests: []archival.RequestEntry{{
					Failure: &probeEOFError,
				}},
			},
		},
		wantOut: webconnectivity.Summary{
			BlockingReason: &httpFailure,
			Blocking:       &httpFailure,
			Accessible:     &falseValue,
			Status: webconnectivity.StatusExperimentHTTP |
				webconnectivity.StatusAnomalyReadWrite,
		},
	}, {
		name: "with timeout",
		args: args{
			tk: &webconnectivity.TestKeys{
				Requests: []archival.RequestEntry{{
					Failure: &probeTimeout,
				}},
			},
		},
		wantOut: webconnectivity.Summary{
			BlockingReason: &httpFailure,
			Blocking:       &httpFailure,
			Accessible:     &falseValue,
			Status: webconnectivity.StatusExperimentHTTP |
				webconnectivity.StatusAnomalyUnknown,
		},
	}, {
		name: "with SSL invalid hostname",
		args: args{
			tk: &webconnectivity.TestKeys{
				Requests: []archival.RequestEntry{{
					Failure: &probeSSLInvalidHost,
				}},
			},
		},
		wantOut: webconnectivity.Summary{
			BlockingReason: &httpFailure,
			Blocking:       &httpFailure,
			Accessible:     &falseValue,
			Status: webconnectivity.StatusExperimentHTTP |
				webconnectivity.StatusAnomalyTLSHandshake,
		},
	}, {
		name: "with SSL invalid cert",
		args: args{
			tk: &webconnectivity.TestKeys{
				Requests: []archival.RequestEntry{{
					Failure: &probeSSLInvalidCert,
				}},
			},
		},
		wantOut: webconnectivity.Summary{
			BlockingReason: &httpFailure,
			Blocking:       &httpFailure,
			Accessible:     &falseValue,
			Status: webconnectivity.StatusExperimentHTTP |
				webconnectivity.StatusAnomalyTLSHandshake,
		},
	}, {
		name: "with SSL unknown auth",
		args: args{
			tk: &webconnectivity.TestKeys{
				Requests: []archival.RequestEntry{{
					Failure: &probeSSLUnknownAuth,
				}},
			},
		},
		wantOut: webconnectivity.Summary{
			BlockingReason: &httpFailure,
			Blocking:       &httpFailure,
			Accessible:     &falseValue,
			Status: webconnectivity.StatusExperimentHTTP |
				webconnectivity.StatusAnomalyTLSHandshake,
		},
	}, {
		name: "with SSL unknown auth _and_ untrustworthy DNS",
		args: args{
			tk: &webconnectivity.TestKeys{
				DNSAnalysisResult: webconnectivity.DNSAnalysisResult{
					DNSConsistency: &webconnectivity.DNSInconsistent,
				},
				Requests: []archival.RequestEntry{{
					Failure: &probeSSLUnknownAuth,
				}},
			},
		},
		wantOut: webconnectivity.Summary{
			BlockingReason: &dns,
			Blocking:       &dns,
			Accessible:     &falseValue,
			Status: webconnectivity.StatusExperimentHTTP |
				webconnectivity.StatusAnomalyTLSHandshake |
				webconnectivity.StatusAnomalyDNS,
		},
	}, {
		name: "with SSL unknown auth _and_ untrustworthy DNS _and_ a longer chain",
		args: args{
			tk: &webconnectivity.TestKeys{
				DNSAnalysisResult: webconnectivity.DNSAnalysisResult{
					DNSConsistency: &webconnectivity.DNSInconsistent,
				},
				Requests: []archival.RequestEntry{{
					Failure: &probeSSLUnknownAuth,
				}, {}},
			},
		},
		wantOut: webconnectivity.Summary{
			BlockingReason: &httpFailure,
			Blocking:       &httpFailure,
			Accessible:     &falseValue,
			Status: webconnectivity.StatusExperimentHTTP |
				webconnectivity.StatusAnomalyTLSHandshake,
		},
	}, {
		name: "with status code and body length matching",
		args: args{
			tk: &webconnectivity.TestKeys{
				HTTPAnalysisResult: webconnectivity.HTTPAnalysisResult{
					StatusCodeMatch: &trueValue,
					BodyLengthMatch: &trueValue,
				},
				Requests: []archival.RequestEntry{{}},
			},
		},
		wantOut: webconnectivity.Summary{
			BlockingReason: nil,
			Blocking:       falseValue,
			Accessible:     &trueValue,
			Status:         webconnectivity.StatusSuccessCleartext,
		},
	}, {
		name: "with status code and headers matching",
		args: args{
			tk: &webconnectivity.TestKeys{
				HTTPAnalysisResult: webconnectivity.HTTPAnalysisResult{
					StatusCodeMatch: &trueValue,
					HeadersMatch:    &trueValue,
				},
				Requests: []archival.RequestEntry{{}},
			},
		},
		wantOut: webconnectivity.Summary{
			BlockingReason: nil,
			Blocking:       falseValue,
			Accessible:     &trueValue,
			Status:         webconnectivity.StatusSuccessCleartext,
		},
	}, {
		name: "with status code and title matching",
		args: args{
			tk: &webconnectivity.TestKeys{
				HTTPAnalysisResult: webconnectivity.HTTPAnalysisResult{
					StatusCodeMatch: &trueValue,
					TitleMatch:      &trueValue,
				},
				Requests: []archival.RequestEntry{{}},
			},
		},
		wantOut: webconnectivity.Summary{
			BlockingReason: nil,
			Blocking:       falseValue,
			Accessible:     &trueValue,
			Status:         webconnectivity.StatusSuccessCleartext,
		},
	}, {
		name: "with suspect http-diff and inconsistent DNS",
		args: args{
			tk: &webconnectivity.TestKeys{
				HTTPAnalysisResult: webconnectivity.HTTPAnalysisResult{
					StatusCodeMatch: &falseValue,
					TitleMatch:      &trueValue,
				},
				Requests: []archival.RequestEntry{{}},
				DNSAnalysisResult: webconnectivity.DNSAnalysisResult{
					DNSConsistency: &webconnectivity.DNSInconsistent,
				},
			},
		},
		wantOut: webconnectivity.Summary{
			BlockingReason: &dns,
			Blocking:       &dns,
			Accessible:     &falseValue,
			Status: webconnectivity.StatusAnomalyHTTPDiff |
				webconnectivity.StatusAnomalyDNS,
		},
	}, {
		name: "with suspect http-diff and consistent DNS",
		args: args{
			tk: &webconnectivity.TestKeys{
				HTTPAnalysisResult: webconnectivity.HTTPAnalysisResult{
					StatusCodeMatch: &falseValue,
					TitleMatch:      &trueValue,
				},
				Requests: []archival.RequestEntry{{}},
				DNSAnalysisResult: webconnectivity.DNSAnalysisResult{
					DNSConsistency: &webconnectivity.DNSConsistent,
				},
			},
		},
		wantOut: webconnectivity.Summary{
			BlockingReason: &httpDiff,
			Blocking:       &httpDiff,
			Accessible:     &falseValue,
			Status:         webconnectivity.StatusAnomalyHTTPDiff,
		},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOut := webconnectivity.Summarize(tt.args.tk)
			if diff := cmp.Diff(tt.wantOut, gotOut); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
