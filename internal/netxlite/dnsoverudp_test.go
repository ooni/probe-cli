package netxlite

import (
	"bytes"
	"context"
	"errors"
	"net"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/google/go-cmp/cmp"
	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/netxlite/filtering"
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
							MockLocalAddr: func() net.Addr {
								return &mocks.Addr{
									MockNetwork: func() string {
										return "udp"
									},
									MockString: func() string {
										return "127.0.0.1:1345"
									},
								}
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
							MockLocalAddr: func() net.Addr {
								return &mocks.Addr{
									MockNetwork: func() string {
										return "udp"
									},
									MockString: func() string {
										return "127.0.0.1:1345"
									},
								}
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
							MockLocalAddr: func() net.Addr {
								return &mocks.Addr{
									MockNetwork: func() string {
										return "udp"
									},
									MockString: func() string {
										return "127.0.0.1:1345"
									},
								}
							},
						}, nil
					},
				}, "9.9.9.9:53",
			)
			expectedErr := errors.New("mocked error")
			txp.Decoder = &mocks.DNSDecoder{
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

		t.Run("decode success", func(t *testing.T) {
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
							MockLocalAddr: func() net.Addr {
								return &mocks.Addr{
									MockNetwork: func() string {
										return "udp"
									},
									MockString: func() string {
										return "127.0.0.1:1345"
									},
								}
							},
						}, nil
					},
				}, "9.9.9.9:53",
			)
			expectedResp := &mocks.DNSResponse{}
			txp.Decoder = &mocks.DNSDecoder{
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

		t.Run("using a real server", func(t *testing.T) {
			srvr := &filtering.DNSServer{
				OnQuery: func(domain string) filtering.DNSAction {
					return filtering.DNSActionCache
				},
				Cache: map[string][]string{
					"dns.google.": {"8.8.8.8"},
				},
			}
			listener, err := srvr.Start("127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			dialer := NewDialerWithoutResolver(model.DiscardLogger)
			txp := NewDNSOverUDPTransport(dialer, listener.LocalAddr().String())
			encoder := &DNSEncoderMiekg{}
			query := encoder.Encode("dns.google.", dns.TypeA, false)
			resp, err := txp.RoundTrip(context.Background(), query)
			if err != nil {
				t.Fatal(err)
			}
			addrs, err := resp.DecodeLookupHost()
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(addrs, []string{"8.8.8.8"}); diff != "" {
				t.Fatal(diff)
			}
		})
	})

	t.Run("AsyncRoundTrip", func(t *testing.T) {
		t.Run("calling Next with cancelled context", func(t *testing.T) {
			srvr := &filtering.DNSServer{
				OnQuery: func(domain string) filtering.DNSAction {
					return filtering.DNSActionCache
				},
				Cache: map[string][]string{
					"dns.google.": {"8.8.8.8"},
				},
			}
			listener, err := srvr.Start("127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			dialer := NewDialerWithoutResolver(model.DiscardLogger)
			txp := NewDNSOverUDPTransport(dialer, listener.LocalAddr().String())
			encoder := &DNSEncoderMiekg{}
			query := encoder.Encode("dns.google.", dns.TypeA, false)
			ctx := context.Background()
			rch, err := txp.AsyncRoundTrip(ctx, query, 1)
			if err != nil {
				t.Fatal(err)
			}
			defer rch.Close()
			ctx, cancel := context.WithCancel(ctx)
			cancel() // fail immediately
			resp, err := rch.Next(ctx)
			if !errors.Is(err, context.Canceled) {
				t.Fatal("unexpected err", err)
			}
			if resp != nil {
				t.Fatal("unexpected resp")
			}
		})

		t.Run("no-one is reading the channel", func(t *testing.T) {
			srvr := &filtering.DNSServer{
				OnQuery: func(domain string) filtering.DNSAction {
					return filtering.DNSActionLocalHostPlusCache // i.e., two responses
				},
				Cache: map[string][]string{
					"dns.google.": {"8.8.8.8"},
				},
			}
			listener, err := srvr.Start("127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			dialer := NewDialerWithoutResolver(model.DiscardLogger)
			txp := NewDNSOverUDPTransport(dialer, listener.LocalAddr().String())
			encoder := &DNSEncoderMiekg{}
			query := encoder.Encode("dns.google.", dns.TypeA, false)
			ctx := context.Background()
			rch, err := txp.AsyncRoundTrip(ctx, query, 1)
			if err != nil {
				t.Fatal(err)
			}
			defer rch.Close()
			<-rch.Joined // should see no-one is reading and stop
		})

		t.Run("typical usage to obtain late responses", func(t *testing.T) {
			srvr := &filtering.DNSServer{
				OnQuery: func(domain string) filtering.DNSAction {
					return filtering.DNSActionLocalHostPlusCache
				},
				Cache: map[string][]string{
					"dns.google.": {"8.8.8.8"},
				},
			}
			listener, err := srvr.Start("127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			dialer := NewDialerWithoutResolver(model.DiscardLogger)
			txp := NewDNSOverUDPTransport(dialer, listener.LocalAddr().String())
			encoder := &DNSEncoderMiekg{}
			query := encoder.Encode("dns.google.", dns.TypeA, false)
			rch, err := txp.AsyncRoundTrip(context.Background(), query, 1)
			if err != nil {
				t.Fatal(err)
			}
			defer rch.Close()
			resp, err := rch.Next(context.Background())
			if err != nil {
				t.Fatal(err)
			}
			addrs, err := resp.DecodeLookupHost()
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(addrs, []string{"127.0.0.1"}); diff != "" {
				t.Fatal(diff)
			}
			// One would not normally busy loop but it's fine to do that in the context
			// of this test because we know we're going to receive a second reply. In
			// a real network experiment here we'll do other activities, e.g., contacting
			// the test helper or fetching a webpage.
			var additional []model.DNSResponse
			for {
				additional = rch.TryNextResponses()
				if len(additional) > 0 {
					if len(additional) != 1 {
						t.Fatal("expected exactly one additional response")
					}
					break
				}
			}
			addrs, err = additional[0].DecodeLookupHost()
			if err != nil {
				t.Fatal(err)
			}
			if diff := cmp.Diff(addrs, []string{"8.8.8.8"}); diff != "" {
				t.Fatal(diff)
			}
		})

		t.Run("correct behavior when read times out", func(t *testing.T) {
			srvr := &filtering.DNSServer{
				OnQuery: func(domain string) filtering.DNSAction {
					return filtering.DNSActionTimeout
				},
			}
			listener, err := srvr.Start("127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			dialer := NewDialerWithoutResolver(model.DiscardLogger)
			txp := NewDNSOverUDPTransport(dialer, listener.LocalAddr().String())
			txp.IOTimeout = 30 * time.Millisecond // short timeout to have a fast test
			encoder := &DNSEncoderMiekg{}
			query := encoder.Encode("dns.google.", dns.TypeA, false)
			rch, err := txp.AsyncRoundTrip(context.Background(), query, 1)
			if err != nil {
				t.Fatal(err)
			}
			defer rch.Close()
			result := <-rch.Response
			if result.Err == nil || result.Err.Error() != "generic_timeout_error" {
				t.Fatal("unexpected error", result.Err)
			}
			if result.Operation != ReadOperation {
				t.Fatal("unexpected failed operation", result.Operation)
			}
		})
	})

	t.Run("CloseIdleConnections", func(t *testing.T) {
		var called bool
		dialer := &mocks.Dialer{
			MockCloseIdleConnections: func() {
				called = true
			},
		}
		const address = "9.9.9.9:53"
		txp := NewDNSOverUDPTransport(dialer, address)
		txp.CloseIdleConnections()
		if !called {
			t.Fatal("not called")
		}
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
