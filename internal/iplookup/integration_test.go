package iplookup_test

import (
	"context"
	"errors"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/iplookup"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// This test makes sure that each of the lookuppers work as intended.
func TestEachLookupperWorksAsIntended(t *testing.T) {
	// create client for testing
	c := &iplookup.Client{
		Logger:        model.DiscardLogger,
		Resolver:      netxlite.NewStdlibResolver(model.DiscardLogger),
		TestingHTTPDo: nil,
	}

	// testcase is a test case for this test
	type testcase struct {
		// name is the test case name
		name string

		// method is the method to invoke
		method iplookup.Method

		// family is the family to use
		family model.AddressFamily

		// expectErr is the error we expect
		expectErr error
	}

	// testcases contains all the test cases
	testcases := []testcase{{
		name:      "cloudflare v4",
		method:    iplookup.MethodWebClouflare,
		family:    model.AddressFamilyINET,
		expectErr: nil,
	}, {
		name:      "cloudflare v6",
		method:    iplookup.MethodWebClouflare,
		family:    model.AddressFamilyINET6,
		expectErr: nil,
	}, {
		name:      "ekiga v4",
		method:    iplookup.MethodSTUNEkiga,
		family:    model.AddressFamilyINET,
		expectErr: nil,
	}, {
		name:      "ekiga v6",
		method:    iplookup.MethodSTUNEkiga,
		family:    model.AddressFamilyINET6,
		expectErr: netxlite.ErrOODNSNoAnswer,
	}, {
		name:      "google v4",
		method:    iplookup.MethodSTUNGoogle,
		family:    model.AddressFamilyINET,
		expectErr: nil,
	}, {
		name:      "google v6",
		method:    iplookup.MethodSTUNGoogle,
		family:    model.AddressFamilyINET6,
		expectErr: nil,
	}, {
		name:      "ubuntu v4",
		method:    iplookup.MethodWebUbuntu,
		family:    model.AddressFamilyINET,
		expectErr: nil,
	}, {
		name:      "ubuntu v6",
		method:    iplookup.MethodWebUbuntu,
		family:    model.AddressFamilyINET6,
		expectErr: netxlite.ErrOODNSNoAnswer,
	}, {
		name:      "random v4",
		method:    iplookup.MethodAllRandom,
		family:    model.AddressFamilyINET,
		expectErr: nil,
	}, {
		name:      "random v6",
		method:    iplookup.MethodAllRandom,
		family:    model.AddressFamilyINET6,
		expectErr: nil,
	}}

	// test each lookupper
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			addr, err := c.LookupIPAddr(context.Background(), tc.method, tc.family)
			if !errors.Is(err, tc.expectErr) {
				t.Fatal("unexpected error", err)
			}
			if addr != "" {
				t.Log(tc.name, tc.family, addr)
			} else {
				t.Log(tc.name, tc.family, err)
			}
		})
	}
}
