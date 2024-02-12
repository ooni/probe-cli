package minipipeline

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/model"
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
				T0: 0.11,
				T:  0.4,
			}, {
				T0: 0.5,
				T:  0.66,
			}}
		},
		expect: []*model.ArchivalDNSLookupResult{{}, {}},
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
				T0: 0.11,
				T:  0.4,
			}, {
				T0: 0.5,
				T:  0.66,
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
