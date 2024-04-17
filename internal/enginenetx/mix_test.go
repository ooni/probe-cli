package enginenetx

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

func TestMixSequentially(t *testing.T) {
	primary := []*httpsDialerTactic{}
	fallback := []*httpsDialerTactic{}

	ff := &testingx.FakeFiller{}
	ff.Fill(&primary)
	ff.Fill(&fallback)

	expect := append([]*httpsDialerTactic{}, primary...)
	expect = append(expect, fallback...)

	var output []*httpsDialerTactic
	for tx := range mixSequentially(streamTacticsFromSlice(primary), streamTacticsFromSlice(fallback)) {
		output = append(output, tx)
	}

	if diff := cmp.Diff(expect, output); diff != "" {
		t.Fatal(diff)
	}
}

func TestMixDeterministicThenRandom(t *testing.T) {
	// define primary data source
	primary := []*httpsDialerTactic{{
		Address:        "130.192.91.211",
		InitialDelay:   0,
		Port:           "443",
		SNI:            "a1.com",
		VerifyHostname: "api.ooni.io",
	}, {
		Address:        "130.192.91.211",
		InitialDelay:   0,
		Port:           "443",
		SNI:            "a2.com",
		VerifyHostname: "api.ooni.io",
	}, {
		Address:        "130.192.91.211",
		InitialDelay:   0,
		Port:           "443",
		SNI:            "a3.com",
		VerifyHostname: "api.ooni.io",
	}, {
		Address:        "130.192.91.211",
		InitialDelay:   0,
		Port:           "443",
		SNI:            "a4.com",
		VerifyHostname: "api.ooni.io",
	}, {
		Address:        "130.192.91.211",
		InitialDelay:   0,
		Port:           "443",
		SNI:            "a5.com",
		VerifyHostname: "api.ooni.io",
	}, {
		Address:        "130.192.91.211",
		InitialDelay:   0,
		Port:           "443",
		SNI:            "a6.com",
		VerifyHostname: "api.ooni.io",
	}, {
		Address:        "130.192.91.211",
		InitialDelay:   0,
		Port:           "443",
		SNI:            "a7.com",
		VerifyHostname: "api.ooni.io",
	}}

	// define fallback data source
	fallback := []*httpsDialerTactic{{
		Address:        "130.192.91.211",
		InitialDelay:   0,
		Port:           "443",
		SNI:            "b1.com",
		VerifyHostname: "api.ooni.io",
	}, {
		Address:        "130.192.91.211",
		InitialDelay:   0,
		Port:           "443",
		SNI:            "b2.com",
		VerifyHostname: "api.ooni.io",
	}, {
		Address:        "130.192.91.211",
		InitialDelay:   0,
		Port:           "443",
		SNI:            "b3.com",
		VerifyHostname: "api.ooni.io",
	}, {
		Address:        "130.192.91.211",
		InitialDelay:   0,
		Port:           "443",
		SNI:            "b4.com",
		VerifyHostname: "api.ooni.io",
	}, {
		Address:        "130.192.91.211",
		InitialDelay:   0,
		Port:           "443",
		SNI:            "b5.com",
		VerifyHostname: "api.ooni.io",
	}, {
		Address:        "130.192.91.211",
		InitialDelay:   0,
		Port:           "443",
		SNI:            "b6.com",
		VerifyHostname: "api.ooni.io",
	}, {
		Address:        "130.192.91.211",
		InitialDelay:   0,
		Port:           "443",
		SNI:            "b7.com",
		VerifyHostname: "api.ooni.io",
	}}

	// define the expectations for the beginning of the result
	expectBeinning := []*httpsDialerTactic{{
		Address:        "130.192.91.211",
		InitialDelay:   0,
		Port:           "443",
		SNI:            "a1.com",
		VerifyHostname: "api.ooni.io",
	}, {
		Address:        "130.192.91.211",
		InitialDelay:   0,
		Port:           "443",
		SNI:            "a2.com",
		VerifyHostname: "api.ooni.io",
	}, {
		Address:        "130.192.91.211",
		InitialDelay:   0,
		Port:           "443",
		SNI:            "b1.com",
		VerifyHostname: "api.ooni.io",
	}, {
		Address:        "130.192.91.211",
		InitialDelay:   0,
		Port:           "443",
		SNI:            "b2.com",
		VerifyHostname: "api.ooni.io",
	}, {
		Address:        "130.192.91.211",
		InitialDelay:   0,
		Port:           "443",
		SNI:            "b3.com",
		VerifyHostname: "api.ooni.io",
	}}

	// remix
	outch := mixDeterministicThenRandom(
		&mixDeterministicThenRandomConfig{
			C: streamTacticsFromSlice(primary),
			N: 2,
		},
		&mixDeterministicThenRandomConfig{
			C: streamTacticsFromSlice(fallback),
			N: 3,
		},
	)
	var output []*httpsDialerTactic
	for tx := range outch {
		output = append(output, tx)
	}

	// make sure we have the expected number of entries
	if len(output) != 14 {
		t.Fatal("we need 14 entries")
	}
	if diff := cmp.Diff(expectBeinning, output[:5]); diff != "" {
		t.Fatal(diff)
	}

	// make sure each entry is represented
	const (
		inprimary = 1 << 0
		infallback
		inoutput
	)
	mapping := make(map[string]int)
	for _, entry := range primary {
		mapping[entry.tacticSummaryKey()] |= inprimary
	}
	for _, entry := range fallback {
		mapping[entry.tacticSummaryKey()] |= infallback
	}
	for _, entry := range output {
		mapping[entry.tacticSummaryKey()] |= inoutput
	}
	for entry, flags := range mapping {
		if flags != (inprimary|inoutput) && flags != (infallback|inoutput) {
			t.Fatal("unexpected flags", flags, "for entry", entry)
		}
	}
}
