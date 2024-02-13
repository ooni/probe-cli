package minipipeline

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/randx"
)

func TestNormalizeDNSLookupResults(t *testing.T) {
	type testcase struct {
		name     string
		inputGen func() []*model.ArchivalDNSLookupResult
		expect   []*model.ArchivalDNSLookupResult
	}

	cases := []testcase{{
		name: "with nil input",
		inputGen: func() []*model.ArchivalDNSLookupResult {
			return nil
		},
		expect: nil,
	}, {
		name: "with empty input",
		inputGen: func() []*model.ArchivalDNSLookupResult {
			return []*model.ArchivalDNSLookupResult{}
		},
		expect: []*model.ArchivalDNSLookupResult{},
	}, {
		name: "with plausible input",
		inputGen: func() []*model.ArchivalDNSLookupResult {
			return []*model.ArchivalDNSLookupResult{{
				Engine:      "udp",
				RawResponse: []byte("0xdeadbeef"),
				T0:          0.11,
				T:           0.4,
			}, {
				Engine:      "doh",
				RawResponse: []byte("0xdeadbeef"),
				T0:          0.5,
				T:           0.66,
			}}
		},
		expect: []*model.ArchivalDNSLookupResult{{
			Engine:          "udp",
			ResolverAddress: "1.1.1.1:53",
		}, {
			Engine:          "doh",
			ResolverAddress: "https://dns.google/dns-query",
		}},
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			values := tc.inputGen()

			NormalizeDNSLookupResults(values)

			if diff := cmp.Diff(tc.expect, values); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestNormalizeNetworkEvents(t *testing.T) {
	type testcase struct {
		name     string
		inputGen func() []*model.ArchivalNetworkEvent
		expect   []*model.ArchivalNetworkEvent
	}

	cases := []testcase{{
		name: "with nil input",
		inputGen: func() []*model.ArchivalNetworkEvent {
			return nil
		},
		expect: nil,
	}, {
		name: "with empty input",
		inputGen: func() []*model.ArchivalNetworkEvent {
			return []*model.ArchivalNetworkEvent{}
		},
		expect: []*model.ArchivalNetworkEvent{},
	}, {
		name: "with plausible input",
		inputGen: func() []*model.ArchivalNetworkEvent {
			return []*model.ArchivalNetworkEvent{{
				T0: 0.11,
				T:  0.4,
			}, {
				T0: 0.5,
				T:  0.66,
			}}
		},
		expect: []*model.ArchivalNetworkEvent{{}, {}},
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			values := tc.inputGen()

			NormalizeNetworkEvents(values)

			if diff := cmp.Diff(tc.expect, values); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestNormalizeTCPConnectResults(t *testing.T) {
	type testcase struct {
		name     string
		inputGen func() []*model.ArchivalTCPConnectResult
		expect   []*model.ArchivalTCPConnectResult
	}

	cases := []testcase{{
		name: "with nil input",
		inputGen: func() []*model.ArchivalTCPConnectResult {
			return nil
		},
		expect: nil,
	}, {
		name: "with empty input",
		inputGen: func() []*model.ArchivalTCPConnectResult {
			return []*model.ArchivalTCPConnectResult{}
		},
		expect: []*model.ArchivalTCPConnectResult{},
	}, {
		name: "with plausible input",
		inputGen: func() []*model.ArchivalTCPConnectResult {
			return []*model.ArchivalTCPConnectResult{{
				T0: 0.11,
				T:  0.4,
			}, {
				T0: 0.5,
				T:  0.66,
			}}
		},
		expect: []*model.ArchivalTCPConnectResult{{}, {}},
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			values := tc.inputGen()

			NormalizeTCPConnectResults(values)

			if diff := cmp.Diff(tc.expect, values); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestNormalizeTLSHandshakeResults(t *testing.T) {
	type testcase struct {
		name     string
		inputGen func() []*model.ArchivalTLSOrQUICHandshakeResult
		expect   []*model.ArchivalTLSOrQUICHandshakeResult
	}

	cases := []testcase{{
		name: "with nil input",
		inputGen: func() []*model.ArchivalTLSOrQUICHandshakeResult {
			return nil
		},
		expect: nil,
	}, {
		name: "with empty input",
		inputGen: func() []*model.ArchivalTLSOrQUICHandshakeResult {
			return []*model.ArchivalTLSOrQUICHandshakeResult{}
		},
		expect: []*model.ArchivalTLSOrQUICHandshakeResult{},
	}, {
		name: "with plausible input",
		inputGen: func() []*model.ArchivalTLSOrQUICHandshakeResult {
			return []*model.ArchivalTLSOrQUICHandshakeResult{{
				PeerCertificates: []model.ArchivalBinaryData{[]byte("0xdeadbeef")},
				T0:               0.11,
				T:                0.4,
			}, {
				PeerCertificates: []model.ArchivalBinaryData{[]byte("0xdeadbeef")},
				T0:               0.5,
				T:                0.66,
			}}
		},
		expect: []*model.ArchivalTLSOrQUICHandshakeResult{{}, {}},
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			values := tc.inputGen()

			NormalizeTLSHandshakeResults(values)

			if diff := cmp.Diff(tc.expect, values); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}

func TestNormalizeHTTPRequestResults(t *testing.T) {
	type testcase struct {
		name     string
		inputGen func() []*model.ArchivalHTTPRequestResult
		expect   []*model.ArchivalHTTPRequestResult
	}

	cases := []testcase{{
		name: "with nil input",
		inputGen: func() []*model.ArchivalHTTPRequestResult {
			return nil
		},
		expect: nil,
	}, {
		name: "with empty input",
		inputGen: func() []*model.ArchivalHTTPRequestResult {
			return []*model.ArchivalHTTPRequestResult{}
		},
		expect: []*model.ArchivalHTTPRequestResult{},
	}, {
		name: "with plausible input",
		inputGen: func() []*model.ArchivalHTTPRequestResult {
			return []*model.ArchivalHTTPRequestResult{{
				T0: 0.11,
				T:  0.4,
			}, {
				Response: model.ArchivalHTTPResponse{
					Body: model.ArchivalScrubbedMaybeBinaryString("1234567"),
				},
				T0: 0.5,
				T:  0.66,
			}, {
				Response: model.ArchivalHTTPResponse{
					Body: model.ArchivalScrubbedMaybeBinaryString(randx.Letters(1 << 19)),
				},
				T0: 0.7,
				T:  0.88,
			}}
		},
		expect: []*model.ArchivalHTTPRequestResult{
			{
				// empty
			},
			{
				Response: model.ArchivalHTTPResponse{
					Body: model.ArchivalScrubbedMaybeBinaryString("1234567"),
				},
			},
			{
				Response: model.ArchivalHTTPResponse{
					Body:            model.ArchivalScrubbedMaybeBinaryString(""),
					BodyIsTruncated: true,
				},
			},
		},
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			values := tc.inputGen()

			NormalizeHTTPRequestResults(values)

			if diff := cmp.Diff(tc.expect, values); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
