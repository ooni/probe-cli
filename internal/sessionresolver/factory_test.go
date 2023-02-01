package sessionresolver

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"syscall"
	"testing"

	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/bytecounter"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/netxlite/filtering"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// testDNSOverHTTPSHandler is an [http.Handler] serving DNS over HTTPS.
type testDNSOverHTTPSHandler struct {
	// A contains the addresses to return.
	A []net.IP
}

var _ http.Handler = &testDNSOverHTTPSHandler{}

// ServeHTTP implements http.Handler
func (h *testDNSOverHTTPSHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	rawQuery, err := netxlite.ReadAllContext(r.Context(), r.Body)
	if err != nil {
		panic(err)
	}
	query := &dns.Msg{}
	if err := query.Unpack(rawQuery); err != nil {
		panic(err)
	}
	runtimex.Assert(len(query.Question) == 1, "expected a single question")
	resp := &dns.Msg{}
	resp.SetReply(query)
	resp.Compress = true
	resp.RecursionAvailable = true
	question0 := query.Question[0]
	switch question0.Qtype {
	case dns.TypeA:
		for _, entry := range h.A {
			resp.Answer = append(resp.Answer, &dns.A{
				Hdr: dns.RR_Header{
					Name:   question0.Name,
					Rrtype: dns.TypeA,
					Class:  dns.ClassINET,
					Ttl:    0,
				},
				A: entry,
			})
		}
	default:
		// nothing
	}
	rawResp, err := resp.Pack()
	if err != nil {
		panic(err)
	}
	w.Header().Add("content-type", "application/dns-message")
	w.Write(rawResp)

}

func Test_newChildResolver(t *testing.T) {
	t.Run("we cannot create an HTTP3 enabled resolver with a proxy URL", func(t *testing.T) {
		reso, err := newChildResolver(
			model.DiscardLogger,
			"https://www.google.com",
			true,
			bytecounter.New(),
			&url.URL{}, // even an empty URL is enough
		)
		if !errors.Is(err, errCannotUseHTTP3WithAProxyURL) {
			t.Fatal("unexpected error", err)
		}
		if reso != nil {
			t.Fatal("expected nil resolver here")
		}
	})

	t.Run("we return an error when we cannot parse the resolver URL", func(t *testing.T) {
		reso, err := newChildResolver(
			model.DiscardLogger,
			"\t",
			true,
			bytecounter.New(),
			nil,
		)
		if err == nil || !strings.HasSuffix(err.Error(), "invalid control character in URL") {
			t.Fatal("unexpected error", err)
		}
		if reso != nil {
			t.Fatal("expected nil resolver here")
		}
	})

	t.Run("we return an error when we don't support the URL scheme", func(t *testing.T) {
		reso, err := newChildResolver(
			model.DiscardLogger,
			"dot://8.8.8.8:853/",
			true,
			bytecounter.New(),
			nil,
		)
		if !errors.Is(err, errUnsupportedResolverScheme) {
			t.Fatal("unexpected error", err)
		}
		if reso != nil {
			t.Fatal("expected nil resolver here")
		}
	})

	t.Run("for HTTPS resolvers", func(t *testing.T) {

		t.Run("the returned resolver wraps errors", func(t *testing.T) {
			srvr := filtering.NewHTTPServerCleartext(filtering.HTTPActionReset)
			defer srvr.Close()
			reso, err := newChildResolver(
				model.DiscardLogger,
				srvr.URL().String(),
				false,
				bytecounter.New(),
				nil,
			)
			if err != nil {
				t.Fatal(err)
			}
			addrs, err := reso.LookupHost(context.Background(), "dns.google")
			if err == nil || err.Error() != netxlite.FailureConnectionReset {
				t.Fatal("unexpected error", err)
			}
			if len(addrs) != 0 {
				t.Fatal("expected zero length addrs here")
			}
		})

		t.Run("what we get is a DNS-over-HTTPS resolver", func(t *testing.T) {
			handler := &testDNSOverHTTPSHandler{
				A: []net.IP{net.IPv4(8, 8, 8, 8)},
			}
			srvr := httptest.NewServer(handler)
			defer srvr.Close()

			reso, err := newChildResolver(
				model.DiscardLogger,
				srvr.URL,
				false,
				bytecounter.New(),
				nil,
			)
			if err != nil {
				t.Fatal(err)
			}
			addrs, err := reso.LookupHost(context.Background(), "dns.google")
			if err != nil {
				t.Fatal("unexpected error", err)
			}

			if len(addrs) != 1 {
				t.Fatal("expected a single addr here")
			}
			if addrs[0] != "8.8.8.8" {
				t.Fatal("unexpected addr")
			}
		})

		t.Run("we count the bytes received and sent", func(t *testing.T) {
			counter := bytecounter.New()

			handler := &testDNSOverHTTPSHandler{
				A: []net.IP{net.IPv4(8, 8, 8, 8)},
			}
			srvr := httptest.NewServer(handler)
			defer srvr.Close()

			reso, err := newChildResolver(
				model.DiscardLogger,
				srvr.URL,
				false,
				counter,
				nil,
			)
			if err != nil {
				t.Fatal(err)
			}
			addrs, err := reso.LookupHost(context.Background(), "dns.google")
			if err != nil {
				t.Fatal("unexpected error", err)
			}

			if len(addrs) != 1 {
				t.Fatal("expected a single addr here")
			}
			if addrs[0] != "8.8.8.8" {
				t.Fatal("unexpected addr")
			}

			if counter.BytesReceived() <= 0 {
				t.Fatal("expected to see received bytes")
			}

			if counter.BytesSent() <= 0 {
				t.Fatal("expected to see sent bytes")
			}
		})

		t.Run("the returned resolver is such that we reject bogons", func(t *testing.T) {

			handler := &testDNSOverHTTPSHandler{
				A: []net.IP{net.IPv4(10, 10, 34, 34)},
			}
			srvr := httptest.NewServer(handler)
			defer srvr.Close()

			reso, err := newChildResolver(
				model.DiscardLogger,
				srvr.URL,
				false,
				bytecounter.New(),
				nil,
			)
			if err != nil {
				t.Fatal(err)
			}
			addrs, err := reso.LookupHost(context.Background(), "dns.google")
			if err == nil || err.Error() != netxlite.FailureDNSBogonError {
				t.Fatal("unexpected error", err)
			}

			if len(addrs) != 0 {
				t.Fatal("expected no addrs here")
			}
		})
	})

	t.Run("for the system resolver", func(t *testing.T) {

		t.Run("the returned resolver wraps errors", func(t *testing.T) {
			tproxy := &mocks.UnderlyingNetwork{
				MockGetaddrinfoLookupANY: func(ctx context.Context, domain string) ([]string, string, error) {
					return nil, "", syscall.ENETDOWN
				},
				MockGetaddrinfoResolverNetwork: func() string {
					return netxlite.StdlibResolverGetaddrinfo
				},
			}
			netxlite.WithCustomTProxy(tproxy, func() {
				reso, err := newChildResolver(
					model.DiscardLogger,
					"system:///",
					false,
					bytecounter.New(),
					nil,
				)
				if err != nil {
					t.Fatal(err)
				}

				addrs, err := reso.LookupHost(context.Background(), "dns.google")
				if err == nil || err.Error() != netxlite.FailureNetworkDown {
					t.Fatal("unexpected error", err)
				}
				if len(addrs) != 0 {
					t.Fatal("expected zero length addrs here")
				}
			})
		})

		t.Run("the returned resolver is such that we reject bogons", func(t *testing.T) {
			tproxy := &mocks.UnderlyingNetwork{
				MockGetaddrinfoLookupANY: func(ctx context.Context, domain string) ([]string, string, error) {
					addrs := []string{"10.10.34.34"}
					return addrs, "", nil
				},
				MockGetaddrinfoResolverNetwork: func() string {
					return netxlite.StdlibResolverGetaddrinfo
				},
			}
			netxlite.WithCustomTProxy(tproxy, func() {
				reso, err := newChildResolver(
					model.DiscardLogger,
					"system:///",
					false,
					bytecounter.New(),
					nil,
				)
				if err != nil {
					t.Fatal(err)
				}

				addrs, err := reso.LookupHost(context.Background(), "dns.google")
				if err == nil || err.Error() != netxlite.FailureDNSBogonError {
					t.Fatal("unexpected error", err)
				}
				if len(addrs) != 0 {
					t.Fatal("expected zero length addrs here")
				}
			})
		})

		t.Run("we count the bytes sent and received", func(t *testing.T) {
			counter := bytecounter.New()

			tproxy := &mocks.UnderlyingNetwork{
				MockGetaddrinfoLookupANY: func(ctx context.Context, domain string) ([]string, string, error) {
					addrs := []string{"8.8.8.8"}
					return addrs, "", nil
				},
				MockGetaddrinfoResolverNetwork: func() string {
					return netxlite.StdlibResolverGetaddrinfo
				},
			}
			netxlite.WithCustomTProxy(tproxy, func() {
				reso, err := newChildResolver(
					model.DiscardLogger,
					"system:///",
					false,
					counter,
					nil,
				)
				if err != nil {
					t.Fatal(err)
				}

				addrs, err := reso.LookupHost(context.Background(), "dns.google")
				if err != nil {
					t.Fatal("unexpected error", err)
				}
				if len(addrs) != 1 {
					t.Fatal("expected a single addr here")
				}
				if addrs[0] != "8.8.8.8" {
					t.Fatal("unexpected addr")
				}

				if counter.BytesReceived() <= 0 {
					t.Fatal("expected to see received bytes")
				}

				if counter.BytesSent() <= 0 {
					t.Fatal("expected to see sent bytes")
				}
			})
		})
	})
}
