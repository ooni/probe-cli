package webconnectivity_test

import (
	"io"
	"net/url"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/webconnectivity"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/errorx"
)

func TestDNSAnalysis(t *testing.T) {
	measurementFailure := errorx.FailureDNSNXDOMAINError
	controlFailure := webconnectivity.DNSNameError
	eofFailure := io.EOF.Error()
	type args struct {
		URL         *url.URL
		measurement webconnectivity.DNSLookupResult
		control     webconnectivity.ControlResponse
	}
	tests := []struct {
		name    string
		args    args
		wantOut webconnectivity.DNSAnalysisResult
	}{{
		name: "when the URL contains an IP address",
		args: args{
			URL: &url.URL{
				Host: "10.0.0.1",
			},
			control: webconnectivity.ControlResponse{
				DNS: webconnectivity.ControlDNSResult{
					Failure: &controlFailure,
				},
			},
		},
		wantOut: webconnectivity.DNSAnalysisResult{
			DNSConsistency: &webconnectivity.DNSConsistent,
		},
	}, {
		name: "when the failures are not compatible",
		args: args{
			URL: &url.URL{
				Host: "www.kerneltrap.org",
			},
			measurement: webconnectivity.DNSLookupResult{
				Failure: &eofFailure,
			},
			control: webconnectivity.ControlResponse{
				DNS: webconnectivity.ControlDNSResult{
					Failure: &controlFailure,
				},
			},
		},
		wantOut: webconnectivity.DNSAnalysisResult{
			DNSConsistency: &webconnectivity.DNSInconsistent,
		},
	}, {
		name: "when the failures are compatible",
		args: args{
			URL: &url.URL{
				Host: "www.kerneltrap.org",
			},
			measurement: webconnectivity.DNSLookupResult{
				Failure: &measurementFailure,
			},
			control: webconnectivity.ControlResponse{
				DNS: webconnectivity.ControlDNSResult{
					Failure: &controlFailure,
				},
			},
		},
		wantOut: webconnectivity.DNSAnalysisResult{
			DNSConsistency: &webconnectivity.DNSConsistent,
		},
	}, {
		name: "when the ASNs are equal",
		args: args{
			URL: &url.URL{
				Host: "fancy.dns",
			},
			measurement: webconnectivity.DNSLookupResult{
				Addrs: map[string]int64{
					"1.1.1.1": 15169,
					"8.8.8.8": 13335,
				},
			},
			control: webconnectivity.ControlResponse{
				DNS: webconnectivity.ControlDNSResult{
					ASNs: []int64{13335, 15169},
				},
			},
		},
		wantOut: webconnectivity.DNSAnalysisResult{
			DNSConsistency: &webconnectivity.DNSConsistent,
		},
	}, {
		name: "when the ASNs overlap",
		args: args{
			URL: &url.URL{
				Host: "fancy.dns",
			},
			measurement: webconnectivity.DNSLookupResult{
				Addrs: map[string]int64{
					"1.1.1.1": 15169,
					"8.8.8.8": 13335,
				},
			},
			control: webconnectivity.ControlResponse{
				DNS: webconnectivity.ControlDNSResult{
					ASNs: []int64{13335, 13335},
				},
			},
		},
		wantOut: webconnectivity.DNSAnalysisResult{
			DNSConsistency: &webconnectivity.DNSConsistent,
		},
	}, {
		name: "when the ASNs do not overlap",
		args: args{
			URL: &url.URL{
				Host: "fancy.dns",
			},
			measurement: webconnectivity.DNSLookupResult{
				Addrs: map[string]int64{
					"1.1.1.1": 15169,
					"8.8.8.8": 15169,
				},
			},
			control: webconnectivity.ControlResponse{
				DNS: webconnectivity.ControlDNSResult{
					ASNs: []int64{13335, 13335},
				},
			},
		},
		wantOut: webconnectivity.DNSAnalysisResult{
			DNSConsistency: &webconnectivity.DNSInconsistent,
		},
	}, {
		name: "when ASNs lookup fails but IPs overlap",
		args: args{
			URL: &url.URL{
				Host: "fancy.dns",
			},
			measurement: webconnectivity.DNSLookupResult{
				Addrs: map[string]int64{
					"2001:4860:4860::8844": 0,
					"8.8.4.4":              0,
				},
			},
			control: webconnectivity.ControlResponse{
				DNS: webconnectivity.ControlDNSResult{
					Addrs: []string{"8.8.8.8", "2001:4860:4860::8844"},
					ASNs:  []int64{0, 0},
				},
			},
		},
		wantOut: webconnectivity.DNSAnalysisResult{
			DNSConsistency: &webconnectivity.DNSConsistent,
		},
	}, {
		name: "when ASNs lookup fails and IPs do not overlap",
		args: args{
			URL: &url.URL{
				Host: "fancy.dns",
			},
			measurement: webconnectivity.DNSLookupResult{
				Addrs: map[string]int64{
					"2001:4860:4860::8888": 0,
					"8.8.8.8":              0,
				},
			},
			control: webconnectivity.ControlResponse{
				DNS: webconnectivity.ControlDNSResult{
					Addrs: []string{"8.8.4.4", "2001:4860:4860::8844"},
					ASNs:  []int64{0, 0},
				},
			},
		},
		wantOut: webconnectivity.DNSAnalysisResult{
			DNSConsistency: &webconnectivity.DNSInconsistent,
		},
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotOut := webconnectivity.DNSAnalysis(tt.args.URL, tt.args.measurement, tt.args.control)
			if diff := cmp.Diff(tt.wantOut, gotOut); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
