package dslx_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/dslx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netemx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// qaStringLessFunc is an utility function to force cmp.Diff to sort string
// slices before performing comparison so that the order doesn't matter
func qaStringLessFunc(a, b string) bool {
	return a < b
}

func TestDNSLookupQA(t *testing.T) {
	// testcase is a test case implemented by this function
	type testcase struct {
		// name is the test case name
		name string

		// newRuntime is the function that creates a new runtime
		newRuntime func(netx model.MeasuringNetwork) dslx.Runtime

		// configureDPI configures DPI
		configureDPI func(dpi *netem.DPIEngine)

		// domain is the domain to resolve
		domain dslx.DomainName

		// expectErr is the expected DNS error or nil
		expectErr error

		// expectAddrs contains the expected DNS addresses
		expectAddrs []string
	}

	cases := []testcase{{
		name: "successful case with minimal runtime",
		newRuntime: func(netx model.MeasuringNetwork) dslx.Runtime {
			return dslx.NewMinimalRuntime(log.Log, time.Now(), dslx.MinimalRuntimeOptionMeasuringNetwork(netx))
		},
		configureDPI: func(dpi *netem.DPIEngine) {
			// nothing
		},
		domain:      "dns.google",
		expectErr:   nil,
		expectAddrs: []string{"8.8.8.8", "8.8.4.4"},
	}, {
		name: "with injected nxdomain error and minimal runtime",
		newRuntime: func(netx model.MeasuringNetwork) dslx.Runtime {
			return dslx.NewMinimalRuntime(log.Log, time.Now(), dslx.MinimalRuntimeOptionMeasuringNetwork(netx))
		},
		configureDPI: func(dpi *netem.DPIEngine) {
			dpi.AddRule(&netem.DPISpoofDNSResponse{
				Addresses: []string{}, // empty to cause NXDOMAIN
				Logger:    log.Log,
				Domain:    "dns.google",
			})
		},
		domain:      "dns.google",
		expectErr:   dslx.ErrDNSLookupParallel,
		expectAddrs: []string{},
	}}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// create an internet testing scenario
			env := netemx.MustNewScenario(netemx.InternetScenario)
			defer env.Close()

			// create a dslx.Runtime using the client stack
			rt := tc.newRuntime(&netxlite.Netx{
				Underlying: &netxlite.NetemUnderlyingNetworkAdapter{UNet: env.ClientStack},
			})
			defer rt.Close()

			// configure the DPI engine
			tc.configureDPI(env.DPIEngine())

			// create DNS lookup function
			function := dslx.DNSLookupParallel(
				dslx.DNSLookupGetaddrinfo(rt),
				dslx.DNSLookupUDP(rt, net.JoinHostPort(netemx.AddressDNSQuad9Net, "53")),
			)

			// create context
			ctx := context.Background()

			// perform DNS lookup
			results := function.Apply(ctx, dslx.NewMaybeWithValue(dslx.NewDomainToResolve(tc.domain)))

			// unpack the results
			resolvedAddrs, err := results.State, results.Error

			// make sure the error matches expectations
			switch {
			case err == nil && tc.expectErr == nil:
				// nothing

			case err != nil && tc.expectErr == nil:
				t.Fatal("expected", tc.expectErr, "got", err)

			case err == nil && tc.expectErr != nil:
				t.Fatal("expected", tc.expectErr, "got", err)

			case err != nil && tc.expectErr != nil:
				if err.Error() != tc.expectErr.Error() {
					t.Fatal("expected", tc.expectErr, "got", err)
				}
				return // no reason to continue
			}

			// make sure that the domain has been correctly copied
			if resolvedAddrs.Domain != string(tc.domain) {
				t.Fatal("expected", tc.domain, "got", resolvedAddrs.Domain)
			}

			// make sure we resolved the expected IP addresses
			if diff := cmp.Diff(tc.expectAddrs, resolvedAddrs.Addresses, cmpopts.SortSlices(qaStringLessFunc)); diff != "" {
				t.Fatal(diff)
			}

			// TODO(bassosimone): make sure the observations are OK
		})
	}
}
