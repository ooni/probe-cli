package netxlite

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
)

func TestAddressFamilyResolver(t *testing.T) {
	t.Run("Address", func(t *testing.T) {
		expected := "1.1.1.1:53"
		child := &mocks.Resolver{
			MockAddress: func() string {
				return expected
			},
		}
		reso := NewAddressFamilyResolver(child, AddressFamilyINET)
		if got := reso.Address(); got != expected {
			t.Fatal("unexpected address", got)
		}
	})

	t.Run("CloseIdleConnections", func(t *testing.T) {
		var called bool
		child := &mocks.Resolver{
			MockCloseIdleConnections: func() {
				called = true
			},
		}
		reso := NewAddressFamilyResolver(child, AddressFamilyINET)
		reso.CloseIdleConnections()
		if !called {
			t.Fatal("not called")
		}
	})

	t.Run("LookupHTTPS", func(t *testing.T) {
		expected := errors.New("mocked error")
		child := &mocks.Resolver{
			MockLookupHTTPS: func(ctx context.Context, domain string) (*model.HTTPSSvc, error) {
				return nil, expected
			},
		}
		reso := NewAddressFamilyResolver(child, AddressFamilyINET)
		svc, err := reso.LookupHTTPS(context.Background(), "ooni.torproject.org")
		if !errors.Is(err, expected) {
			t.Fatal("unexpected error", err)
		}
		if svc != nil {
			t.Fatal("expected nil")
		}
	})

	t.Run("LookupHost", func(t *testing.T) {
		// testcase is a test case for this function
		type testcase struct {
			// name is the test case name
			name string

			// child is the child [model.Resolver] to use
			child model.Resolver

			// family is the family to filter for
			family AddressFamily

			// expectedErr is the expected error
			expectedErr error

			// expectedAddrs contains the expected addresses
			expectedAddrs []string

			// disableWrappedErrorCheck disables the check ensuring
			// that the returned error has been wrapped.
			disableWrappedErrorCheck bool
		}

		// testcases contains all the test cases
		testcases := []testcase{{
			name: "the DNS lookup fails",
			child: &mocks.Resolver{
				MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
					return nil, io.EOF
				},
			},
			family:                   AddressFamilyINET,
			expectedErr:              io.EOF,
			expectedAddrs:            nil,
			disableWrappedErrorCheck: true,
		}, {
			name: "we want AF_INET but don't have any IPv4 addresses",
			child: &mocks.Resolver{
				MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
					addrs := []string{
						"2a00:1450:4002:410::2004",
					}
					return addrs, nil
				},
			},
			family:                   AddressFamilyINET,
			expectedErr:              ErrOODNSNoAnswer,
			expectedAddrs:            nil,
			disableWrappedErrorCheck: false,
		}, {
			name: "we want AF_INET6 but don't have any IPv6 addresses",
			child: &mocks.Resolver{
				MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
					addrs := []string{
						"142.251.209.4",
					}
					return addrs, nil
				},
			},
			family:                   AddressFamilyINET6,
			expectedErr:              ErrOODNSNoAnswer,
			expectedAddrs:            nil,
			disableWrappedErrorCheck: false,
		}, {
			name: "we want AF_INET and have some IPv4 addresses",
			child: &mocks.Resolver{
				MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
					addrs := []string{
						"2a00:1450:4002:410::2004",
						"142.251.209.4",
					}
					return addrs, nil
				},
			},
			family:      AddressFamilyINET,
			expectedErr: nil,
			expectedAddrs: []string{
				"142.251.209.4",
			},
			disableWrappedErrorCheck: false,
		}, {
			name: "we want AF_INET6 and have some IPv6 addresses",
			child: &mocks.Resolver{
				MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
					addrs := []string{
						"2a00:1450:4002:410::2004",
						"142.251.209.4",
					}
					return addrs, nil
				},
			},
			family:      AddressFamilyINET6,
			expectedErr: nil,
			expectedAddrs: []string{
				"2a00:1450:4002:410::2004",
			},
			disableWrappedErrorCheck: false,
		}, {
			name: "the underlying resolver returns a non-IP-address string",
			child: &mocks.Resolver{
				MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
					return []string{"antani"}, nil
				},
			},
			family:                   AddressFamilyINET,
			expectedErr:              ErrOODNSNoAnswer,
			expectedAddrs:            nil,
			disableWrappedErrorCheck: false,
		}}

		// run all the test cases
		for _, tc := range testcases {
			t.Run(tc.name, func(t *testing.T) {
				// create the [AddressFamilyResolver]
				reso := NewAddressFamilyResolver(tc.child, tc.family)

				// perform the lookup
				addrs, err := reso.LookupHost(context.Background(), "www.google.com")

				// make sure we've got the expected error
				if !errors.Is(err, tc.expectedErr) {
					t.Fatal("unexpected error", err)
				}

				// make sure we've got the expected addresses
				if diff := cmp.Diff(tc.expectedAddrs, addrs); diff != "" {
					t.Fatal(diff)
				}

				// check whether the returned error is wrapped
				if tc.disableWrappedErrorCheck || err == nil {
					return
				}
				var wrapper *ErrWrapper
				if !errors.As(err, &wrapper) {
					t.Fatal("error has not been wrapped")
				}
			})
		}
	})

	t.Run("LookupNS", func(t *testing.T) {
		expected := errors.New("mocked error")
		child := &mocks.Resolver{
			MockLookupNS: func(ctx context.Context, domain string) ([]*net.NS, error) {
				return nil, expected
			},
		}
		reso := NewAddressFamilyResolver(child, AddressFamilyINET)
		nss, err := reso.LookupNS(context.Background(), "ooni.torproject.org")
		if !errors.Is(err, expected) {
			t.Fatal("unexpected error", err)
		}
		if len(nss) != 0 {
			t.Fatal("expected zero-length slice")
		}
	})

	t.Run("Network", func(t *testing.T) {
		expected := "tcp"
		child := &mocks.Resolver{
			MockNetwork: func() string {
				return expected
			},
		}
		reso := NewAddressFamilyResolver(child, AddressFamilyINET)
		if got := reso.Network(); got != expected {
			t.Fatal("unexpected network", got)
		}
	})
}
