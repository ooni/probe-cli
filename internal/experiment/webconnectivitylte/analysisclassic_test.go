package webconnectivitylte

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/minipipeline"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/tailscale/hujson"
)

// This test is related to https://github.com/ooni/probe/issues/2517.
//
// It shows how blocking flags could vary depending on the ASN distribution based on
// a real case where we observed false positives caused by that.
func TestTestKeys_analysisDNSToplevel(t *testing.T) {

	// testcase is a test case in this test
	type testcase struct {
		// name is the name of the test case
		name string

		// tkFile is the name of the JSONC file containing the test keys
		tkFile string

		// geoInfo contains a static mapping of geoip info
		geoInfo map[string]*model.LocationASN

		// expectBlockingFlags contains the expected BlockingFlags
		expecteBlockingFlags int64
	}

	testcases := []testcase{{
		name:   "https://github.com/ooni/probe/issues/2517",
		tkFile: filepath.Join("testdata", "20230706183840.201925_PK_webconnectivity_19f5e0d803cbaea7.jsonc"),
		geoInfo: map[string]*model.LocationASN{
			"172.224.19.10":        {ASNumber: 36183, Organization: "Akamai Technologies, Inc."},
			"172.224.19.5":         {ASNumber: 36183, Organization: "Akamai Technologies, Inc."},
			"172.224.19.9":         {ASNumber: 36183, Organization: "Akamai Technologies, Inc."},
			"17.248.248.101":       {ASNumber: 714, Organization: "Apple Inc."},
			"2a01:b740:a41:212::8": {ASNumber: 714, Organization: "Apple Inc."},
			"2a01:b740:a41:212::7": {ASNumber: 714, Organization: "Apple Inc."},
			"2a01:b740:a41:213::7": {ASNumber: 714, Organization: "Apple Inc."},
			"172.224.19.3":         {ASNumber: 36183, Organization: "Akamai Technologies, Inc."},
			"172.224.19.12":        {ASNumber: 36183, Organization: "Akamai Technologies, Inc."},
			"17.248.248.103":       {ASNumber: 714, Organization: "Apple Inc."},
			"17.248.248.119":       {ASNumber: 714, Organization: "Apple Inc."},
			"2a01:b740:a41:213::5": {ASNumber: 714, Organization: "Apple Inc."},
			"172.224.19.4":         {ASNumber: 36183, Organization: "Akamai Technologies, Inc."},
			"172.224.19.6":         {ASNumber: 36183, Organization: "Akamai Technologies, Inc."},
			"172.224.19.11":        {ASNumber: 36183, Organization: "Akamai Technologies, Inc."},
			"2a01:b740:a41:212::4": {ASNumber: 714, Organization: "Apple Inc."},
			"172.224.19.7":         {ASNumber: 36183, Organization: "Akamai Technologies, Inc."},
			"17.248.248.117":       {ASNumber: 714, Organization: "Apple Inc."},
			"17.248.248.121":       {ASNumber: 714, Organization: "Apple Inc."},
			"2a01:b740:a41:212::5": {ASNumber: 714, Organization: "Apple Inc."},
			"17.248.248.104":       {ASNumber: 714, Organization: "Apple Inc."},
			"2a01:b740:a41:213::9": {ASNumber: 714, Organization: "Apple Inc."},
			"172.224.19.14":        {ASNumber: 36183, Organization: "Akamai Technologies, Inc."},
			"172.224.19.15":        {ASNumber: 36183, Organization: "Akamai Technologies, Inc."},
			"2a01:b740:a41:212::6": {ASNumber: 714, Organization: "Apple Inc."},
			"172.224.19.17":        {ASNumber: 36183, Organization: "Akamai Technologies, Inc."},
			"172.224.19.13":        {ASNumber: 36183, Organization: "Akamai Technologies, Inc."},
			"17.248.248.105":       {ASNumber: 714, Organization: "Apple Inc."},
			"17.248.248.100":       {ASNumber: 714, Organization: "Apple Inc."},
		},
		expecteBlockingFlags: AnalysisBlockingFlagDNSBlocking,
	}}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			data := runtimex.Try1(os.ReadFile(tc.tkFile))
			data = runtimex.Try1(hujson.Standardize(data))
			var tk TestKeys
			runtimex.Try0(json.Unmarshal(data, &tk))
			log.SetLevel(log.DebugLevel)
			tk.analysisClassic(mocks.NewGeoIPASNLookupper(tc.geoInfo), log.Log)
			if tc.expecteBlockingFlags != tk.BlockingFlags {
				t.Fatal("expected", tc.expecteBlockingFlags, "got", tk.BlockingFlags)
			}
		})
	}
}

func TestAnalysisClassicContainsOnlyLoopbackAddrs(t *testing.T) {
	type testcase struct {
		name   string
		input  minipipeline.Set[string]
		expect bool
	}

	cases := []testcase{{
		name:   "with empty set",
		input:  minipipeline.NewSet[string](),
		expect: false,
	}, {
		name:   "with only loopback addrs",
		input:  minipipeline.NewSet("127.0.0.1", "::1"),
		expect: true,
	}, {
		name:   "with mixed addrs",
		input:  minipipeline.NewSet("127.0.0.1", "130.192.91.211", "::1"),
		expect: false,
	}, {
		name:   "make sure we skip non-addresses",
		input:  minipipeline.NewSet("antani"),
		expect: false,
	}, {
		name:   "make sure we say not loopback with non-addresses",
		input:  minipipeline.NewSet("::1", "130.192.91.211", "antani", "127.0.0.1"),
		expect: false,
	}, {
		name:   "make sure we say loopback with non-addresses",
		input:  minipipeline.NewSet("::1", "antani", "127.0.0.1"),
		expect: true,
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := analysisContainsOnlyLoopbackAddrs(tc.input)
			if got != tc.expect {
				t.Fatal("expected", tc.expect, "got", got)
			}
		})
	}
}
