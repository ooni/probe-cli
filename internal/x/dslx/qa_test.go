package dslx_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/gopacket/layers"
	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netemx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/x/dslx"
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

		// newRuntime is the function creating a new runtime
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
		name: "success with minimal runtime",
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

			// TODO(https://github.com/ooni/probe/issues/2620): make sure the observations are OK
		})
	}
}

func TestMeasureResolvedAddressesQA(t *testing.T) {
	// testcase is a test case implemented by this function
	type testcase struct {
		// name is the test case name
		name string

		// newRuntime is the function creating a new runtime
		newRuntime func(netx model.MeasuringNetwork) dslx.Runtime

		// configureDPI configures DPI
		configureDPI func(dpi *netem.DPIEngine)

		// expectTCP contains the expected TCP connect stats
		expectTCP map[string]int64

		// expectTLS contains the expected TLS handshake stats
		expectTLS map[string]int64

		// expectQUIC contains the expected QUIC handshake stats
		expectQUIC map[string]int64
	}

	cases := []testcase{{
		name: "success with minimal runtime",
		newRuntime: func(netx model.MeasuringNetwork) dslx.Runtime {
			return dslx.NewMinimalRuntime(log.Log, time.Now(), dslx.MinimalRuntimeOptionMeasuringNetwork(netx))
		},
		configureDPI: func(dpi *netem.DPIEngine) {
			// nothing
		},
		expectTCP:  map[string]int64{"": 2},
		expectTLS:  map[string]int64{"": 2},
		expectQUIC: map[string]int64{"": 2},
	}, {
		name: "TCP connection refused with minimal runtime",
		newRuntime: func(netx model.MeasuringNetwork) dslx.Runtime {
			return dslx.NewMinimalRuntime(log.Log, time.Now(), dslx.MinimalRuntimeOptionMeasuringNetwork(netx))
		},
		configureDPI: func(dpi *netem.DPIEngine) {
			dpi.AddRule(&netem.DPICloseConnectionForServerEndpoint{
				Logger:          log.Log,
				ServerIPAddress: "8.8.8.8",
				ServerPort:      443,
			})
			dpi.AddRule(&netem.DPICloseConnectionForServerEndpoint{
				Logger:          log.Log,
				ServerIPAddress: "8.8.4.4",
				ServerPort:      443,
			})
		},
		expectTCP: map[string]int64{
			"connection_refused": 2,
		},
		expectTLS:  map[string]int64{},
		expectQUIC: map[string]int64{"": 2},
	}, {
		name: "TLS handshake reset with minimal runtime",
		newRuntime: func(netx model.MeasuringNetwork) dslx.Runtime {
			return dslx.NewMinimalRuntime(log.Log, time.Now(), dslx.MinimalRuntimeOptionMeasuringNetwork(netx))
		},
		configureDPI: func(dpi *netem.DPIEngine) {
			dpi.AddRule(&netem.DPIResetTrafficForTLSSNI{
				Logger: log.Log,
				SNI:    "dns.google",
			})
		},
		expectTCP: map[string]int64{"": 2},
		expectTLS: map[string]int64{
			"connection_reset": 2,
		},
		expectQUIC: map[string]int64{"": 2},
	}, {
		name: "QUIC handshake timeout with minimal runtime",
		newRuntime: func(netx model.MeasuringNetwork) dslx.Runtime {
			return dslx.NewMinimalRuntime(log.Log, time.Now(), dslx.MinimalRuntimeOptionMeasuringNetwork(netx))
		},
		configureDPI: func(dpi *netem.DPIEngine) {
			dpi.AddRule(&netem.DPIDropTrafficForServerEndpoint{
				Logger:          log.Log,
				ServerIPAddress: "8.8.8.8",
				ServerPort:      443,
				ServerProtocol:  layers.IPProtocolUDP,
			})
			dpi.AddRule(&netem.DPIDropTrafficForServerEndpoint{
				Logger:          log.Log,
				ServerIPAddress: "8.8.4.4",
				ServerPort:      443,
				ServerProtocol:  layers.IPProtocolUDP,
			})
		},
		expectTCP: map[string]int64{"": 2},
		expectTLS: map[string]int64{"": 2},
		expectQUIC: map[string]int64{
			"generic_timeout_error": 2,
		},
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

			// create stats
			var (
				tcpConnectStats    = dslx.NewStats[*dslx.TCPConnection]()
				tlsHandshakeStats  = dslx.NewStats[*dslx.TLSConnection]()
				quicHandshakeStats = dslx.NewStats[*dslx.QUICConnection]()
			)

			// create endpoint measurement function
			function := dslx.MeasureResolvedAddresses(
				// measure 443/tcp
				dslx.Compose7(
					dslx.MakeEndpoint("tcp", 443),
					dslx.TCPConnect(rt),
					tcpConnectStats.Observer(),
					dslx.TLSHandshake(rt),
					tlsHandshakeStats.Observer(),
					dslx.HTTPRequestOverTLS(rt),
					dslx.Discard[*dslx.HTTPResponse](),
				),

				// measure 443/udp
				dslx.Compose5(
					dslx.MakeEndpoint("udp", 443),
					dslx.QUICHandshake(rt),
					quicHandshakeStats.Observer(),
					dslx.HTTPRequestOverQUIC(rt),
					dslx.Discard[*dslx.HTTPResponse](),
				),
			)

			// create context
			ctx := context.Background()

			// fake out the resolved addresses
			resolvedAddrs := &dslx.ResolvedAddresses{
				Addresses: []string{"8.8.8.8", "8.8.4.4"},
				Domain:    "dns.google",
			}

			// measure the endpoints
			_ = function.Apply(ctx, dslx.NewMaybeWithValue(resolvedAddrs))

			// make sure the TCP connect results are consistent
			if diff := cmp.Diff(tc.expectTCP, tcpConnectStats.Export()); diff != "" {
				t.Fatal(diff)
			}

			// make sure the TLS handshake results are consistent
			if diff := cmp.Diff(tc.expectTLS, tlsHandshakeStats.Export()); diff != "" {
				t.Fatal(diff)
			}

			// make sure the QUIC handshake results are consistent
			if diff := cmp.Diff(tc.expectQUIC, quicHandshakeStats.Export()); diff != "" {
				t.Fatal(diff)
			}

			// TODO(https://github.com/ooni/probe/issues/2620): make sure the observations are OK
		})
	}
}
