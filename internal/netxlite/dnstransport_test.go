package netxlite

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestWrapDNSTransport(t *testing.T) {
	orig := &mocks.DNSTransport{}
	txp := wrapDNSTransport(orig)
	errWrapper := txp.(*dnsTransportErrWrapper)
	underlying := errWrapper.DNSTransport
	if orig != underlying {
		t.Fatal("unexpected underlying transport")
	}
}

func TestDNSTransportErrWrapper(t *testing.T) {
	t.Run("RoundTrip", func(t *testing.T) {
		t.Run("on success", func(t *testing.T) {
			expectedResp := &mocks.DNSResponse{}
			child := &mocks.DNSTransport{
				MockRoundTrip: func(ctx context.Context, query model.DNSQuery) (model.DNSResponse, error) {
					return expectedResp, nil
				},
			}
			txp := &dnsTransportErrWrapper{
				DNSTransport: child,
			}
			query := &mocks.DNSQuery{}
			ctx := context.Background()
			resp, err := txp.RoundTrip(ctx, query)
			if err != nil {
				t.Fatal(err)
			}
			if resp != expectedResp {
				t.Fatal("unexpected resp")
			}
		})

		t.Run("on failure", func(t *testing.T) {
			expectedErr := io.EOF
			child := &mocks.DNSTransport{
				MockRoundTrip: func(ctx context.Context, query model.DNSQuery) (model.DNSResponse, error) {
					return nil, expectedErr
				},
			}
			txp := &dnsTransportErrWrapper{
				DNSTransport: child,
			}
			query := &mocks.DNSQuery{}
			ctx := context.Background()
			resp, err := txp.RoundTrip(ctx, query)
			if !errors.Is(err, expectedErr) {
				t.Fatal("unexpected err", err)
			}
			if resp != nil {
				t.Fatal("unexpected resp")
			}
			var errWrapper *ErrWrapper
			if !errors.As(err, &errWrapper) {
				t.Fatal("error has not been wrapped")
			}
		})
	})

	t.Run("RequiresPadding", func(t *testing.T) {
		child := &mocks.DNSTransport{
			MockRequiresPadding: func() bool {
				return true
			},
		}
		txp := &dnsTransportErrWrapper{
			DNSTransport: child,
		}
		if !txp.RequiresPadding() {
			t.Fatal("expected true")
		}
	})

	t.Run("Network", func(t *testing.T) {
		child := &mocks.DNSTransport{
			MockNetwork: func() string {
				return "x"
			},
		}
		txp := &dnsTransportErrWrapper{
			DNSTransport: child,
		}
		if txp.Network() != "x" {
			t.Fatal("unexpected Network")
		}
	})

	t.Run("Address", func(t *testing.T) {
		child := &mocks.DNSTransport{
			MockAddress: func() string {
				return "x"
			},
		}
		txp := &dnsTransportErrWrapper{
			DNSTransport: child,
		}
		if txp.Address() != "x" {
			t.Fatal("unexpected Address")
		}
	})

	t.Run("CloseIdleConnections", func(t *testing.T) {
		var called bool
		child := &mocks.DNSTransport{
			MockCloseIdleConnections: func() {
				called = true
			},
		}
		txp := &dnsTransportErrWrapper{
			DNSTransport: child,
		}
		txp.CloseIdleConnections()
		if !called {
			t.Fatal("not called")
		}
	})
}
