package netxlite

import (
	"bytes"
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
)

func TestDNSOverUDPTransport(t *testing.T) {
	t.Run("RoundTrip", func(t *testing.T) {
		t.Run("cannot encode query", func(t *testing.T) {
			expected := errors.New("mocked error")
			const address = "9.9.9.9:53"
			txp := NewDNSOverUDPTransport(nil, address)
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

		t.Run("dial failure", func(t *testing.T) {
			mocked := errors.New("mocked error")
			const address = "9.9.9.9:53"
			txp := NewDNSOverUDPTransport(&mocks.Dialer{
				MockDialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
					return nil, mocked
				},
			}, address)
			query := &mocks.DNSQuery{
				MockBytes: func() ([]byte, error) {
					return make([]byte, 128), nil
				},
			}
			resp, err := txp.RoundTrip(context.Background(), query)
			if !errors.Is(err, mocked) {
				t.Fatal("not the error we expected")
			}
			if resp != nil {
				t.Fatal("expected no response here")
			}
		})

		t.Run("Write failure", func(t *testing.T) {
			mocked := errors.New("mocked error")
			txp := NewDNSOverUDPTransport(
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
			query := &mocks.DNSQuery{
				MockBytes: func() ([]byte, error) {
					return make([]byte, 128), nil
				},
			}
			resp, err := txp.RoundTrip(context.Background(), query)
			if !errors.Is(err, mocked) {
				t.Fatal("not the error we expected")
			}
			if resp != nil {
				t.Fatal("expected no response here")
			}
		})

		t.Run("Read failure", func(t *testing.T) {
			mocked := errors.New("mocked error")
			txp := NewDNSOverUDPTransport(
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
			query := &mocks.DNSQuery{
				MockBytes: func() ([]byte, error) {
					return make([]byte, 128), nil
				},
			}
			resp, err := txp.RoundTrip(context.Background(), query)
			if !errors.Is(err, mocked) {
				t.Fatal("not the error we expected")
			}
			if resp != nil {
				t.Fatal("expected no response here")
			}
		})

		t.Run("decode failure", func(t *testing.T) {
			const expected = 17
			input := bytes.NewReader(make([]byte, expected))
			txp := NewDNSOverUDPTransport(
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
			expectedErr := errors.New("mocked error")
			txp.decoder = &mocks.DNSDecoder{
				MockDecodeResponse: func(data []byte, query model.DNSQuery) (model.DNSResponse, error) {
					return nil, expectedErr
				},
			}
			query := &mocks.DNSQuery{
				MockBytes: func() ([]byte, error) {
					return make([]byte, 128), nil
				},
			}
			resp, err := txp.RoundTrip(context.Background(), query)
			if !errors.Is(err, expectedErr) {
				t.Fatal("unexpected err", err)
			}
			if resp != nil {
				t.Fatal("expected nil resp")
			}
		})

		t.Run("read success", func(t *testing.T) {
			const expected = 17
			input := bytes.NewReader(make([]byte, expected))
			txp := NewDNSOverUDPTransport(
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
			expectedResp := &mocks.DNSResponse{}
			txp.decoder = &mocks.DNSDecoder{
				MockDecodeResponse: func(data []byte, query model.DNSQuery) (model.DNSResponse, error) {
					return expectedResp, nil
				},
			}
			query := &mocks.DNSQuery{
				MockBytes: func() ([]byte, error) {
					return make([]byte, 128), nil
				},
			}
			resp, err := txp.RoundTrip(context.Background(), query)
			if err != nil {
				t.Fatal(err)
			}
			if resp != expectedResp {
				t.Fatal("unexpected resp")
			}
		})
	})

	t.Run("other functions okay", func(t *testing.T) {
		const address = "9.9.9.9:53"
		txp := NewDNSOverUDPTransport(NewDialerWithoutResolver(log.Log), address)
		if txp.RequiresPadding() != false {
			t.Fatal("invalid RequiresPadding")
		}
		if txp.Network() != "udp" {
			t.Fatal("invalid Network")
		}
		if txp.Address() != address {
			t.Fatal("invalid Address")
		}
	})
}
