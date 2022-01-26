package netxlite

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model/mocks"
)

func TestDNSOverTCP(t *testing.T) {
	t.Run("RoundTrip", func(t *testing.T) {
		t.Run("query too large", func(t *testing.T) {
			const address = "9.9.9.9:53"
			txp := NewDNSOverTCP(new(net.Dialer).DialContext, address)
			reply, err := txp.RoundTrip(context.Background(), make([]byte, 1<<18))
			if err == nil {
				t.Fatal("expected an error here")
			}
			if reply != nil {
				t.Fatal("expected nil reply here")
			}
		})

		t.Run("dial failure", func(t *testing.T) {
			const address = "9.9.9.9:53"
			mocked := errors.New("mocked error")
			fakedialer := &mocks.Dialer{
				MockDialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
					return nil, mocked
				},
			}
			txp := NewDNSOverTCP(fakedialer.DialContext, address)
			reply, err := txp.RoundTrip(context.Background(), make([]byte, 1<<11))
			if !errors.Is(err, mocked) {
				t.Fatal("not the error we expected")
			}
			if reply != nil {
				t.Fatal("expected nil reply here")
			}
		})

		t.Run("SetDeadline failure", func(t *testing.T) {
			const address = "9.9.9.9:53"
			mocked := errors.New("mocked error")
			fakedialer := &mocks.Dialer{
				MockDialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
					return &mocks.Conn{
						MockSetDeadline: func(t time.Time) error {
							return mocked
						},
						MockClose: func() error {
							return nil
						},
					}, nil
				},
			}
			txp := NewDNSOverTCP(fakedialer.DialContext, address)
			reply, err := txp.RoundTrip(context.Background(), make([]byte, 1<<11))
			if !errors.Is(err, mocked) {
				t.Fatal("not the error we expected")
			}
			if reply != nil {
				t.Fatal("expected nil reply here")
			}
		})

		t.Run("write failure", func(t *testing.T) {
			const address = "9.9.9.9:53"
			mocked := errors.New("mocked error")
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
			txp := NewDNSOverTCP(fakedialer.DialContext, address)
			reply, err := txp.RoundTrip(context.Background(), make([]byte, 1<<11))
			if !errors.Is(err, mocked) {
				t.Fatal("not the error we expected")
			}
			if reply != nil {
				t.Fatal("expected nil reply here")
			}
		})

		t.Run("first read fails", func(t *testing.T) {
			const address = "9.9.9.9:53"
			mocked := errors.New("mocked error")
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
			txp := NewDNSOverTCP(fakedialer.DialContext, address)
			reply, err := txp.RoundTrip(context.Background(), make([]byte, 1<<11))
			if !errors.Is(err, mocked) {
				t.Fatal("not the error we expected")
			}
			if reply != nil {
				t.Fatal("expected nil reply here")
			}
		})

		t.Run("second read fails", func(t *testing.T) {
			const address = "9.9.9.9:53"
			mocked := errors.New("mocked error")
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
			txp := NewDNSOverTCP(fakedialer.DialContext, address)
			reply, err := txp.RoundTrip(context.Background(), make([]byte, 1<<11))
			if !errors.Is(err, mocked) {
				t.Fatal("not the error we expected")
			}
			if reply != nil {
				t.Fatal("expected nil reply here")
			}
		})

		t.Run("successful case", func(t *testing.T) {
			const address = "9.9.9.9:53"
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
			txp := NewDNSOverTCP(fakedialer.DialContext, address)
			reply, err := txp.RoundTrip(context.Background(), make([]byte, 1<<11))
			if err != nil {
				t.Fatal(err)
			}
			if len(reply) != 1 || reply[0] != 1 {
				t.Fatal("not the response we expected")
			}
		})
	})

	t.Run("other functions okay with TCP", func(t *testing.T) {
		const address = "9.9.9.9:53"
		txp := NewDNSOverTCP(new(net.Dialer).DialContext, address)
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
		txp := NewDNSOverTLS((&tls.Dialer{}).DialContext, address)
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
