package tracex

import (
	"bytes"
	"context"
	"errors"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func TestWrapResolver(t *testing.T) {
	var saver *Saver
	reso := &mocks.Resolver{}
	if saver.WrapResolver(reso) != reso {
		t.Fatal("unexpected result")
	}
}

func TestResolverSaver(t *testing.T) {
	t.Run("LookupHost", func(t *testing.T) {
		t.Run("on failure", func(t *testing.T) {
			expected := netxlite.ErrOODNSNoSuchHost
			saver := &Saver{}
			reso := saver.WrapResolver(newFakeResolverWithExplicitError(expected))
			addrs, err := reso.LookupHost(context.Background(), "www.google.com")
			if !errors.Is(err, expected) {
				t.Fatal("not the error we expected")
			}
			if addrs != nil {
				t.Fatal("expected nil address here")
			}
			ev := saver.Read()
			if len(ev) != 2 {
				t.Fatal("expected number of events")
			}
			if ev[0].Value().Hostname != "www.google.com" {
				t.Fatal("unexpected Hostname")
			}
			if ev[0].Name() != "resolve_start" {
				t.Fatal("unexpected name")
			}
			if !ev[0].Value().Time.Before(time.Now()) {
				t.Fatal("the saved time is wrong")
			}
			if ev[1].Value().Addresses != nil {
				t.Fatal("unexpected Addresses")
			}
			if ev[1].Value().Duration <= 0 {
				t.Fatal("unexpected Duration")
			}
			if ev[1].Value().Err != netxlite.FailureDNSNXDOMAINError {
				t.Fatal("unexpected Err")
			}
			if ev[1].Value().Hostname != "www.google.com" {
				t.Fatal("unexpected Hostname")
			}
			if ev[1].Name() != "resolve_done" {
				t.Fatal("unexpected name")
			}
			if !ev[1].Value().Time.After(ev[0].Value().Time) {
				t.Fatal("the saved time is wrong")
			}
		})

		t.Run("on success", func(t *testing.T) {
			expected := []string{"8.8.8.8", "8.8.4.4"}
			saver := &Saver{}
			reso := saver.WrapResolver(newFakeResolverWithResult(expected))
			addrs, err := reso.LookupHost(context.Background(), "www.google.com")
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(addrs, expected) {
				t.Fatal("not the result we expected")
			}
			ev := saver.Read()
			if len(ev) != 2 {
				t.Fatal("expected number of events")
			}
			if ev[0].Value().Hostname != "www.google.com" {
				t.Fatal("unexpected Hostname")
			}
			if ev[0].Name() != "resolve_start" {
				t.Fatal("unexpected name")
			}
			if !ev[0].Value().Time.Before(time.Now()) {
				t.Fatal("the saved time is wrong")
			}
			if !reflect.DeepEqual(ev[1].Value().Addresses, expected) {
				t.Fatal("unexpected Addresses")
			}
			if ev[1].Value().Duration <= 0 {
				t.Fatal("unexpected Duration")
			}
			if ev[1].Value().Err.IsNotNil() {
				t.Fatal("unexpected Err")
			}
			if ev[1].Value().Hostname != "www.google.com" {
				t.Fatal("unexpected Hostname")
			}
			if ev[1].Name() != "resolve_done" {
				t.Fatal("unexpected name")
			}
			if !ev[1].Value().Time.After(ev[0].Value().Time) {
				t.Fatal("the saved time is wrong")
			}
		})

		t.Run("with stdlib resolver there's correct .Network remapping", func(t *testing.T) {
			saver := &Saver{}
			reso := saver.WrapResolver(netxlite.NewStdlibResolver(model.DiscardLogger))
			ctx, cancel := context.WithCancel(context.Background())
			cancel() // immediately fail the operation
			_, _ = reso.LookupHost(ctx, "www.google.com")
			// basically, we just want to ensure that the engine name is converted
			ev := saver.Read()
			if len(ev) != 2 {
				t.Fatal("expected number of events")
			}
			if ev[0].Value().Proto != netxlite.StdlibResolverSystem {
				t.Fatal("unexpected Proto")
			}
			if ev[1].Value().Proto != netxlite.StdlibResolverSystem {
				t.Fatal("unexpected Proto")
			}
		})
	})

	t.Run("Network", func(t *testing.T) {
		t.Run("when using a custom resolver", func(t *testing.T) {
			saver := &Saver{}
			child := &mocks.Resolver{
				MockNetwork: func() string {
					return "x"
				},
			}
			reso := saver.WrapResolver(child)
			if reso.Network() != "x" {
				t.Fatal("unexpected result")
			}
		})

		t.Run("when using the stdlib resolver", func(t *testing.T) {
			child := netxlite.NewStdlibResolver(model.DiscardLogger)
			switch network := child.Network(); network {
			case netxlite.StdlibResolverGetaddrinfo,
				netxlite.StdlibResolverGolangNetResolver:
				// ok
			default:
				t.Fatal("unexpected child resolver network", network)
			}
			saver := &Saver{}
			reso := saver.WrapResolver(child)
			if network := reso.Network(); network != netxlite.StdlibResolverSystem {
				t.Fatal("unexpected wrapped resolver network", network)
			}
		})
	})

	t.Run("Address", func(t *testing.T) {
		saver := &Saver{}
		child := &mocks.Resolver{
			MockAddress: func() string {
				return "x"
			},
		}
		reso := saver.WrapResolver(child)
		if reso.Address() != "x" {
			t.Fatal("unexpected result")
		}
	})

	t.Run("LookupHTTPS", func(t *testing.T) {
		expected := errors.New("mocked")
		saver := &Saver{}
		child := &mocks.Resolver{
			MockLookupHTTPS: func(ctx context.Context, domain string) (*model.HTTPSSvc, error) {
				return nil, expected
			},
		}
		reso := saver.WrapResolver(child)
		https, err := reso.LookupHTTPS(context.Background(), "dns.google")
		if !errors.Is(err, expected) {
			t.Fatal("unexpected err", err)
		}
		if https != nil {
			t.Fatal("expected nil")
		}
	})

	t.Run("LookupNS", func(t *testing.T) {
		expected := errors.New("mocked")
		saver := &Saver{}
		child := &mocks.Resolver{
			MockLookupNS: func(ctx context.Context, domain string) ([]*net.NS, error) {
				return nil, expected
			},
		}
		reso := saver.WrapResolver(child)
		ns, err := reso.LookupNS(context.Background(), "dns.google")
		if !errors.Is(err, expected) {
			t.Fatal("unexpected err", err)
		}
		if len(ns) != 0 {
			t.Fatal("expected zero length array")
		}
	})

	t.Run("CloseIdleConnections", func(t *testing.T) {
		var called bool
		saver := &Saver{}
		child := &mocks.Resolver{
			MockCloseIdleConnections: func() {
				called = true
			},
		}
		reso := saver.WrapResolver(child)
		reso.CloseIdleConnections()
		if !called {
			t.Fatal("not called")
		}
	})
}

func TestWrapDNSTransport(t *testing.T) {
	var saver *Saver
	txp := &mocks.DNSTransport{}
	if saver.WrapDNSTransport(txp) != txp {
		t.Fatal("unexpected result")
	}
}

func TestDNSTransportSaver(t *testing.T) {
	t.Run("RoundTrip", func(t *testing.T) {
		t.Run("on failure", func(t *testing.T) {
			expected := netxlite.ErrOODNSNoSuchHost
			saver := &Saver{}
			txp := saver.WrapDNSTransport(&mocks.DNSTransport{
				MockRoundTrip: func(ctx context.Context, query model.DNSQuery) (model.DNSResponse, error) {
					return nil, expected
				},
				MockNetwork: func() string {
					return "fake"
				},
				MockAddress: func() string {
					return ""
				},
			})
			rawQuery := []byte{0xde, 0xad, 0xbe, 0xef}
			query := &mocks.DNSQuery{
				MockBytes: func() ([]byte, error) {
					return rawQuery, nil
				},
			}
			reply, err := txp.RoundTrip(context.Background(), query)
			if !errors.Is(err, expected) {
				t.Fatal("not the error we expected")
			}
			if reply != nil {
				t.Fatal("expected nil reply here")
			}
			ev := saver.Read()
			if len(ev) != 2 {
				t.Fatal("expected number of events")
			}
			if !bytes.Equal(ev[0].Value().DNSQuery, rawQuery) {
				t.Fatal("unexpected DNSQuery")
			}
			if ev[0].Name() != "dns_round_trip_start" {
				t.Fatal("unexpected name")
			}
			if !ev[0].Value().Time.Before(time.Now()) {
				t.Fatal("the saved time is wrong")
			}
			if !bytes.Equal(ev[1].Value().DNSQuery, rawQuery) {
				t.Fatal("unexpected DNSQuery")
			}
			if ev[1].Value().DNSResponse != nil {
				t.Fatal("unexpected DNSReply")
			}
			if ev[1].Value().Duration <= 0 {
				t.Fatal("unexpected Duration")
			}
			if ev[1].Value().Err != netxlite.FailureDNSNXDOMAINError {
				t.Fatal("unexpected Err")
			}
			if ev[1].Name() != "dns_round_trip_done" {
				t.Fatal("unexpected name")
			}
			if !ev[1].Value().Time.After(ev[0].Value().Time) {
				t.Fatal("the saved time is wrong")
			}
		})

		t.Run("on success", func(t *testing.T) {
			expected := []byte{0xef, 0xbe, 0xad, 0xde}
			saver := &Saver{}
			response := &mocks.DNSResponse{
				MockBytes: func() []byte {
					return expected
				},
			}
			txp := saver.WrapDNSTransport(&mocks.DNSTransport{
				MockRoundTrip: func(ctx context.Context, query model.DNSQuery) (model.DNSResponse, error) {
					return response, nil
				},
				MockNetwork: func() string {
					return "fake"
				},
				MockAddress: func() string {
					return ""
				},
			})
			rawQuery := []byte{0xde, 0xad, 0xbe, 0xef}
			query := &mocks.DNSQuery{
				MockBytes: func() ([]byte, error) {
					return rawQuery, nil
				},
			}
			reply, err := txp.RoundTrip(context.Background(), query)
			if err != nil {
				t.Fatal("we expected nil error here")
			}
			if !bytes.Equal(reply.Bytes(), expected) {
				t.Fatal("expected another reply here")
			}
			ev := saver.Read()
			if len(ev) != 2 {
				t.Fatal("expected number of events")
			}
			if !bytes.Equal(ev[0].Value().DNSQuery, rawQuery) {
				t.Fatal("unexpected DNSQuery")
			}
			if ev[0].Name() != "dns_round_trip_start" {
				t.Fatal("unexpected name")
			}
			if !ev[0].Value().Time.Before(time.Now()) {
				t.Fatal("the saved time is wrong")
			}
			if !bytes.Equal(ev[1].Value().DNSQuery, rawQuery) {
				t.Fatal("unexpected DNSQuery")
			}
			if !bytes.Equal(ev[1].Value().DNSResponse, expected) {
				t.Fatal("unexpected DNSReply")
			}
			if ev[1].Value().Duration <= 0 {
				t.Fatal("unexpected Duration")
			}
			if ev[1].Value().Err.IsNotNil() {
				t.Fatal("unexpected Err")
			}
			if ev[1].Name() != "dns_round_trip_done" {
				t.Fatal("unexpected name")
			}
			if !ev[1].Value().Time.After(ev[0].Value().Time) {
				t.Fatal("the saved time is wrong")
			}
		})

		t.Run("with getaddrinfo transport there's correct .Network remapping", func(t *testing.T) {
			saver := &Saver{}
			reso := saver.WrapDNSTransport(netxlite.NewDNSOverGetaddrinfoTransport())
			ctx, cancel := context.WithCancel(context.Background())
			cancel() // immediately fail the operation
			query := &mocks.DNSQuery{
				MockBytes: func() ([]byte, error) {
					return []byte{}, nil
				},
				MockType: func() uint16 {
					return dns.TypeANY
				},
				MockID: func() uint16 {
					return 1453
				},
				MockDomain: func() string {
					return "dns.google"
				},
			}
			_, _ = reso.RoundTrip(ctx, query)
			// basically, we just want to ensure that the engine name is converted
			ev := saver.Read()
			if len(ev) != 2 {
				t.Fatal("expected number of events")
			}
			if ev[0].Value().Proto != netxlite.StdlibResolverSystem {
				t.Fatal("unexpected Proto")
			}
			if ev[1].Value().Proto != netxlite.StdlibResolverSystem {
				t.Fatal("unexpected Proto")
			}
		})
	})

	t.Run("Network", func(t *testing.T) {
		t.Run("with custom child transport", func(t *testing.T) {
			saver := &Saver{}
			child := &mocks.DNSTransport{
				MockNetwork: func() string {
					return "x"
				},
			}
			txp := saver.WrapDNSTransport(child)
			if txp.Network() != "x" {
				t.Fatal("unexpected result")
			}
		})

		t.Run("when using the stdlib resolver", func(t *testing.T) {
			child := netxlite.NewDNSOverGetaddrinfoTransport()
			switch network := child.Network(); network {
			case netxlite.StdlibResolverGetaddrinfo,
				netxlite.StdlibResolverGolangNetResolver:
				// ok
			default:
				t.Fatal("unexpected child resolver network", network)
			}
			saver := &Saver{}
			reso := saver.WrapDNSTransport(child)
			if network := reso.Network(); network != netxlite.StdlibResolverSystem {
				t.Fatal("unexpected wrapped resolver network", network)
			}
		})
	})

	t.Run("Address", func(t *testing.T) {
		saver := &Saver{}
		child := &mocks.DNSTransport{
			MockAddress: func() string {
				return "x"
			},
		}
		txp := saver.WrapDNSTransport(child)
		if txp.Address() != "x" {
			t.Fatal("unexpected result")
		}
	})

	t.Run("CloseIdleConnections", func(t *testing.T) {
		var called bool
		saver := &Saver{}
		child := &mocks.DNSTransport{
			MockCloseIdleConnections: func() {
				called = true
			},
		}
		txp := saver.WrapDNSTransport(child)
		txp.CloseIdleConnections()
		if !called {
			t.Fatal("not called")
		}
	})

	t.Run("RequiresPadding", func(t *testing.T) {
		saver := &Saver{}
		child := &mocks.DNSTransport{
			MockRequiresPadding: func() bool {
				return true
			},
		}
		txp := saver.WrapDNSTransport(child)
		if !txp.RequiresPadding() {
			t.Fatal("unexpected result")
		}
	})
}

func newFakeResolverWithExplicitError(err error) model.Resolver {
	runtimex.PanicIfNil(err, "passed nil error")
	return &mocks.Resolver{
		MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
			return nil, err
		},
		MockNetwork: func() string {
			return "fake"
		},
		MockAddress: func() string {
			return ""
		},
		MockCloseIdleConnections: func() {
			// nothing
		},
		MockLookupHTTPS: func(ctx context.Context, domain string) (*model.HTTPSSvc, error) {
			return nil, errors.New("not implemented")
		},
		MockLookupNS: func(ctx context.Context, domain string) ([]*net.NS, error) {
			return nil, errors.New("not implemented")
		},
	}
}

func newFakeResolverWithResult(r []string) model.Resolver {
	return &mocks.Resolver{
		MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
			return r, nil
		},
		MockNetwork: func() string {
			return "fake"
		},
		MockAddress: func() string {
			return ""
		},
		MockCloseIdleConnections: func() {
			// nothing
		},
		MockLookupHTTPS: func(ctx context.Context, domain string) (*model.HTTPSSvc, error) {
			return nil, errors.New("not implemented")
		},
		MockLookupNS: func(ctx context.Context, domain string) ([]*net.NS, error) {
			return nil, errors.New("not implemented")
		},
	}
}

func TestResolverNetworkAdaptNames(t *testing.T) {
	type args struct {
		input string
	}
	tests := []struct {
		name string
		args args
		want string
	}{{
		name: "with StdlibResolverGetaddrinfo",
		args: args{
			input: netxlite.StdlibResolverGetaddrinfo,
		},
		want: netxlite.StdlibResolverSystem,
	}, {
		name: "with StdlibResolverGolangNetResolver",
		args: args{
			input: netxlite.StdlibResolverGolangNetResolver,
		},
		want: netxlite.StdlibResolverSystem,
	}, {
		name: "with StdlibResolverSystem",
		args: args{
			input: netxlite.StdlibResolverSystem,
		},
		want: netxlite.StdlibResolverSystem,
	}, {
		name: "with any other name",
		args: args{
			input: "doh",
		},
		want: "doh",
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ResolverNetworkAdaptNames(tt.args.input); got != tt.want {
				t.Errorf("ResolverNetworkAdaptNames() = %v, want %v", got, tt.want)
			}
		})
	}
}
