package enginenetx

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

func TestFilterOutNilTactics(t *testing.T) {
	inputs := []*httpsDialerTactic{
		nil,
		nil,
		{
			Address:        "130.192.91.211",
			InitialDelay:   0,
			Port:           "443",
			SNI:            "x.org",
			VerifyHostname: "api.ooni.io",
		},
		nil,
		{
			Address:        "130.192.91.211",
			InitialDelay:   0,
			Port:           "443",
			SNI:            "www.polito.it",
			VerifyHostname: "api.ooni.io",
		},
		nil,
		nil,
	}

	expect := []*httpsDialerTactic{
		inputs[2], inputs[4],
	}

	var output []*httpsDialerTactic
	for tx := range filterOutNilTactics(streamTacticsFromSlice(inputs)) {
		output = append(output, tx)
	}

	if diff := cmp.Diff(expect, output); diff != "" {
		t.Fatal(diff)
	}
}

func TestFilterOnlyKeepUniqueTactics(t *testing.T) {
	templates := []*httpsDialerTactic{{
		Address:        "130.192.91.211",
		InitialDelay:   0,
		Port:           "443",
		SNI:            "www.example.com",
		VerifyHostname: "api.ooni.io",
	}, {
		Address:        "130.192.91.211",
		InitialDelay:   0,
		Port:           "443",
		SNI:            "www.kernel.org",
		VerifyHostname: "api.ooni.io",
	}, {
		Address:        "130.192.91.211",
		InitialDelay:   0,
		Port:           "443",
		SNI:            "x.org",
		VerifyHostname: "api.ooni.io",
	}, {
		Address:        "130.192.91.211",
		InitialDelay:   0,
		Port:           "443",
		SNI:            "www.polito.it",
		VerifyHostname: "api.ooni.io",
	}}

	inputs := []*httpsDialerTactic{
		templates[2], templates[1], templates[1],
		templates[2], templates[2], templates[1],
		templates[0], templates[1], templates[0],
		templates[2], templates[1], templates[2],
		templates[1], templates[0], templates[1],
		templates[3], // only once at the end
	}

	expect := []*httpsDialerTactic{
		templates[2], templates[1], templates[0], templates[3],
	}

	var output []*httpsDialerTactic
	for tx := range filterOnlyKeepUniqueTactics(streamTacticsFromSlice(inputs)) {
		output = append(output, tx)
	}

	if diff := cmp.Diff(expect, output); diff != "" {
		t.Fatal(diff)
	}
}

func TestFilterAssignInitalDelays(t *testing.T) {
	inputs := []*httpsDialerTactic{}
	ff := &testingx.FakeFiller{}
	ff.Fill(&inputs)
	idx := 0
	for tx := range filterAssignInitialDelays(streamTacticsFromSlice(inputs)) {
		if tx.InitialDelay != happyEyeballsDelay(idx) {
			t.Fatal("unexpected .InitialDelay", tx.InitialDelay, "for", idx)
		}
		idx++
	}
	if idx < 1 {
		t.Fatal("expected to see at least one entry")
	}
}
