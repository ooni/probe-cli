package netxlite

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"io"
	"math"
	"net"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
)

func TestDNSOverTCPTransport(t *testing.T) {
	t.Run("RoundTrip", func(t *testing.T) {
		t.Run("cannot encode query", func(t *testing.T) {
			expected := errors.New("mocked error")
			const address = "9.9.9.9:53"
			txp := NewUnwrappedDNSOverTCPTransport(new(net.Dialer).DialContext, address)
			query := &mocks.DNSQuery{
				MockBytes: func() ([]byte, error) {
					return nil, expected
				},
			}
			resp, err := txp.RoundTrip(context.Background(), query)
			if !errors.Is(err, expected) {
				t.Fatal("unexpected err", err)
			}
			if resp != nil {
				t.Fatal("expected nil response here")
			}
		})

		t.Run("query too large", func(t *testing.T) {
			const address = "9.9.9.9:53"
			txp := NewUnwrappedDNSOverTCPTransport(new(net.Dialer).DialContext, address)
			query := &mocks.DNSQuery{
				MockBytes: func() ([]byte, error) {
					return make([]byte, math.MaxUint16+1), nil
				},
			}
			resp, err := txp.RoundTrip(context.Background(), query)
			if !errors.Is(err, ErrSimpleFrameSize) {
				t.Fatal("unexpected err", err)
			}
			if resp != nil {
				t.Fatal("expected nil response here")
			}
		})

		t.Run("dial failure", func(t *testing.T) {
			const address = "9.9.9.9:53"
			mocked := errors.New("mocked error")
			query := &mocks.DNSQuery{
				MockBytes: func() ([]byte, error) {
					return make([]byte, 128), nil
				},
			}
			fakedialer := &mocks.Dialer{
				MockDialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
					return nil, mocked
				},
			}
			txp := NewUnwrappedDNSOverTCPTransport(fakedialer.DialContext, address)
			resp, err := txp.RoundTrip(context.Background(), query)
			if !errors.Is(err, mocked) {
				t.Fatal("not the error we expected")
			}
			if resp != nil {
				t.Fatal("expected nil resp here")
			}
		})

		t.Run("write failure", func(t *testing.T) {
			const address = "9.9.9.9:53"
			mocked := errors.New("mocked error")
			query := &mocks.DNSQuery{
				MockBytes: func() ([]byte, error) {
					return make([]byte, 128), nil
				},
			}
			fakedialer := &mocks.Dialer{
				MockDialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
					return &mocks.Conn{
						MockSetDeadline: func(t time.Time) error {
							return nil
						},
						MockWrite: func(b []byte) (int, error) {
							return 0, mocked
						},
						MockClose: func() error {
							return nil
						},
					}, nil
				},
			}
			txp := NewUnwrappedDNSOverTCPTransport(fakedialer.DialContext, address)
			resp, err := txp.RoundTrip(context.Background(), query)
			if !errors.Is(err, mocked) {
				t.Fatal("not the error we expected")
			}
			if resp != nil {
				t.Fatal("expected nil resp here")
			}
		})

		t.Run("first read fails", func(t *testing.T) {
			const address = "9.9.9.9:53"
			mocked := errors.New("mocked error")
			query := &mocks.DNSQuery{
				MockBytes: func() ([]byte, error) {
					return make([]byte, 128), nil
				},
			}
			fakedialer := &mocks.Dialer{
				MockDialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
					return &mocks.Conn{
						MockSetDeadline: func(t time.Time) error {
							return nil
						},
						MockWrite: func(b []byte) (int, error) {
							return len(b), nil
						},
						MockRead: func(b []byte) (int, error) {
							return 0, mocked
						},
						MockClose: func() error {
							return nil
						},
					}, nil
				},
			}
			txp := NewUnwrappedDNSOverTCPTransport(fakedialer.DialContext, address)
			resp, err := txp.RoundTrip(context.Background(), query)
			if !errors.Is(err, mocked) {
				t.Fatal("not the error we expected")
			}
			if resp != nil {
				t.Fatal("expected nil resp here")
			}
		})

		t.Run("second read fails", func(t *testing.T) {
			const address = "9.9.9.9:53"
			mocked := errors.New("mocked error")
			query := &mocks.DNSQuery{
				MockBytes: func() ([]byte, error) {
					return make([]byte, 128), nil
				},
			}
			input := io.MultiReader(
				bytes.NewReader([]byte{byte(0), byte(2)}),
				&mocks.Reader{
					MockRead: func(b []byte) (int, error) {
						return 0, mocked
					},
				},
			)
			fakedialer := &mocks.Dialer{
				MockDialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
					return &mocks.Conn{
						MockSetDeadline: func(t time.Time) error {
							return nil
						},
						MockWrite: func(b []byte) (int, error) {
							return len(b), nil
						},
						MockRead: input.Read,
						MockClose: func() error {
							return nil
						},
					}, nil
				},
			}
			txp := NewUnwrappedDNSOverTCPTransport(fakedialer.DialContext, address)
			resp, err := txp.RoundTrip(context.Background(), query)
			if !errors.Is(err, mocked) {
				t.Fatal("not the error we expected")
			}
			if resp != nil {
				t.Fatal("expected nil resp here")
			}
		})

		t.Run("decode failure", func(t *testing.T) {
			const address = "9.9.9.9:53"
			mocked := errors.New("mocked error")
			query := &mocks.DNSQuery{
				MockBytes: func() ([]byte, error) {
					return make([]byte, 128), nil
				},
			}
			input := bytes.NewReader([]byte{byte(0), byte(1), byte(1)})
			fakedialer := &mocks.Dialer{
				MockDialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
					return &mocks.Conn{
						MockSetDeadline: func(t time.Time) error {
							return nil
						},
						MockWrite: func(b []byte) (int, error) {
							return len(b), nil
						},
						MockRead: input.Read,
						MockClose: func() error {
							return nil
						},
					}, nil
				},
			}
			txp := NewUnwrappedDNSOverTCPTransport(fakedialer.DialContext, address)
			txp.decoder = &mocks.DNSDecoder{
				MockDecodeResponse: func(data []byte, query model.DNSQuery) (model.DNSResponse, error) {
					return nil, mocked
				},
			}
			resp, err := txp.RoundTrip(context.Background(), query)
			if !errors.Is(err, mocked) {
				t.Fatal("unexpected err", err)
			}
			if resp != nil {
				t.Fatal("expected nil resp here")
			}
		})

		t.Run("successful case", func(t *testing.T) {
			const address = "9.9.9.9:53"
			query := &mocks.DNSQuery{
				MockBytes: func() ([]byte, error) {
					return make([]byte, 128), nil
				},
			}
			input := bytes.NewReader([]byte{byte(0), byte(1), byte(1)})
			fakedialer := &mocks.Dialer{
				MockDialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
					return &mocks.Conn{
						MockSetDeadline: func(t time.Time) error {
							return nil
						},
						MockWrite: func(b []byte) (int, error) {
							return len(b), nil
						},
						MockRead: input.Read,
						MockClose: func() error {
							return nil
						},
					}, nil
				},
			}
			txp := NewUnwrappedDNSOverTCPTransport(fakedialer.DialContext, address)
			expectedResp := &mocks.DNSResponse{}
			txp.decoder = &mocks.DNSDecoder{
				MockDecodeResponse: func(data []byte, query model.DNSQuery) (model.DNSResponse, error) {
					return expectedResp, nil
				},
			}
			resp, err := txp.RoundTrip(context.Background(), query)
			if err != nil {
				t.Fatal(err)
			}
			if resp != expectedResp {
				t.Fatal("not the response we expected")
			}
		})
	})

	t.Run("other functions okay with TCP", func(t *testing.T) {
		const address = "9.9.9.9:53"
		txp := NewUnwrappedDNSOverTCPTransport(new(net.Dialer).DialContext, address)
		if txp.RequiresPadding() != false {
			t.Fatal("invalid RequiresPadding")
		}
		if txp.Network() != "tcp" {
			t.Fatal("invalid Network")
		}
		if txp.Address() != address {
			t.Fatal("invalid Address")
		}
		txp.CloseIdleConnections()
	})

	t.Run("other functions okay with TLS", func(t *testing.T) {
		const address = "9.9.9.9:853"
		txp := NewUnwrappedDNSOverTLSTransport((&tls.Dialer{}).DialContext, address)
		if txp.RequiresPadding() != true {
			t.Fatal("invalid RequiresPadding")
		}
		if txp.Network() != "dot" {
			t.Fatal("invalid Network")
		}
		if txp.Address() != address {
			t.Fatal("invalid Address")
		}
		txp.CloseIdleConnections()
	})
}
