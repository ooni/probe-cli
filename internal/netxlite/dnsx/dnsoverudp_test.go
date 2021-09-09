package dnsx

import (
	"bytes"
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/netxlite/mocks"
)

func TestDNSOverUDPDialFailure(t *testing.T) {
	mocked := errors.New("mocked error")
	const address = "9.9.9.9:53"
	txp := NewDNSOverUDP(&mocks.Dialer{
		MockDialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			return nil, mocked
		},
	}, address)
	data, err := txp.RoundTrip(context.Background(), nil)
	if !errors.Is(err, mocked) {
		t.Fatal("not the error we expected")
	}
	if data != nil {
		t.Fatal("expected no response here")
	}
}

func TestDNSOverUDPSetDeadlineError(t *testing.T) {
	mocked := errors.New("mocked error")
	txp := NewDNSOverUDP(
		&mocks.Dialer{
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
		}, "9.9.9.9:53",
	)
	data, err := txp.RoundTrip(context.Background(), nil)
	if !errors.Is(err, mocked) {
		t.Fatal("not the error we expected")
	}
	if data != nil {
		t.Fatal("expected no response here")
	}
}

func TestDNSOverUDPWriteFailure(t *testing.T) {
	mocked := errors.New("mocked error")
	txp := NewDNSOverUDP(
		&mocks.Dialer{
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
		}, "9.9.9.9:53",
	)
	data, err := txp.RoundTrip(context.Background(), nil)
	if !errors.Is(err, mocked) {
		t.Fatal("not the error we expected")
	}
	if data != nil {
		t.Fatal("expected no response here")
	}
}

func TestDNSOverUDPReadFailure(t *testing.T) {
	mocked := errors.New("mocked error")
	txp := NewDNSOverUDP(
		&mocks.Dialer{
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
		}, "9.9.9.9:53",
	)
	data, err := txp.RoundTrip(context.Background(), nil)
	if !errors.Is(err, mocked) {
		t.Fatal("not the error we expected")
	}
	if data != nil {
		t.Fatal("expected no response here")
	}
}

func TestDNSOverUDPReadSuccess(t *testing.T) {
	const expected = 17
	input := bytes.NewReader(make([]byte, expected))
	txp := NewDNSOverUDP(
		&mocks.Dialer{
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
		}, "9.9.9.9:53",
	)
	data, err := txp.RoundTrip(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != expected {
		t.Fatal("expected non nil data")
	}
}

func TestDNSOverUDPTransportOK(t *testing.T) {
	const address = "9.9.9.9:53"
	txp := NewDNSOverUDP(&net.Dialer{}, address)
	if txp.RequiresPadding() != false {
		t.Fatal("invalid RequiresPadding")
	}
	if txp.Network() != "udp" {
		t.Fatal("invalid Network")
	}
	if txp.Address() != address {
		t.Fatal("invalid Address")
	}
}
