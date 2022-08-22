package netxlite

import (
	"bytes"
	"context"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/google/go-cmp/cmp"
	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/netxlite/filtering"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

func TestDNSOverUDPTransport(t *testing.T) {
	t.Run("RoundTrip", func(t *testing.T) {
		t.Run("cannot encode query", func(t *testing.T) {
			expected := errors.New("mocked error")
			const address = "9.9.9.9:53"
			txp := NewUnwrappedDNSOverUDPTransport(nil, address)
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
			txp := NewUnwrappedDNSOverUDPTransport(&mocks.Dialer{
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
			txp := NewUnwrappedDNSOverUDPTransport(
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
			txp := NewUnwrappedDNSOverUDPTransport(
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
			txp := NewUnwrappedDNSOverUDPTransport(
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
			txp := NewUnwrappedDNSOverUDPTransport(
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
			txp := NewUnwrappedDNSOverUDPTransport(dialer, listener.LocalAddr().String())
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

	t.Run("recording delayed DNS responses", func(t *testing.T) {
		t.Run("uses a context-injected custom trace (success case)", func(t *testing.T) {
			var (
				delayedDNSResponseCalled bool
				goodQueryType            bool
				goodTransportNetwork     bool
				goodTransportAddress     bool
				goodLookupAddrs          bool
				goodError                bool
			)
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
			expectedAddress := listener.LocalAddr().String()
			txp := NewUnwrappedDNSOverUDPTransport(dialer, expectedAddress)
			encoder := &DNSEncoderMiekg{}
			query := encoder.Encode("dns.google.", dns.TypeA, false)
			zeroTime := time.Now()
			deterministicTime := testingx.NewTimeDeterministic(zeroTime)
			expectedAddrs := []string{"8.8.8.8"}
			respChannel := make(chan *model.DNSResponse, 8)
			mu := new(sync.Mutex)
			tx := &mocks.Trace{
				MockTimeNow: deterministicTime.Now,
				MockOnDelayedDNSResponse: func(started time.Time, txp model.DNSTransport,
					query model.DNSQuery, response model.DNSResponse, addrs []string, err error,
					finished time.Time) error {
					mu.Lock()
					delayedDNSResponseCalled = true
					goodQueryType = (query.Type() == dns.TypeA)
					goodTransportNetwork = (txp.Network() == "udp")
					goodTransportAddress = (txp.Address() == expectedAddress)
					goodLookupAddrs = (cmp.Diff(expectedAddrs, addrs) == "")
					goodError = (err == nil)
					mu.Unlock()
					select {
					case respChannel <- &response:
						return nil
					default:
						return errors.New("full buffer")
					}
				},
				MockOnConnectDone: func(started time.Time, network, domain, remoteAddr string, err error,
					finished time.Time) {
					// do nothing
				},
				MockMaybeWrapNetConn: func(conn net.Conn) net.Conn {
					return conn
				},
			}
			ctx := ContextWithTrace(context.Background(), tx)
			rch, err := txp.RoundTrip(ctx, query)
			<-respChannel // wait for the delayed response
			if err != nil {
				t.Fatal(err)
			}
			addrs, err := rch.DecodeLookupHost()
			if err != nil {
				t.Fatal(err)
			}
			mu.Lock()
			if diff := cmp.Diff(addrs, []string{"127.0.0.1"}); diff != "" {
				t.Fatal(diff)
			}
			if !delayedDNSResponseCalled {
				t.Fatal("delayedDNSResponse not called")
			}
			if !goodQueryType {
				t.Fatal("unexpected query type")
			}
			if !goodTransportNetwork {
				t.Fatal("unexpected DNS transport network")
			}
			if !goodTransportAddress {
				t.Fatal("unexpected DNS Transport address")
			}
			if !goodLookupAddrs {
				t.Fatal("unexpected delayed DNSLookup address")
			}
			if !goodError {
				t.Fatal("unexpected error encountered")
			}
			mu.Unlock()
		})

		t.Run("uses a context-injected custom trace (failure case)", func(t *testing.T) {
			var (
				delayedDNSResponseCalled bool
				goodQueryType            bool
				goodTransportNetwork     bool
				goodTransportAddress     bool
				goodLookupAddrs          bool
				goodError                bool
			)
			srvr := &filtering.DNSServer{
				OnQuery: func(domain string) filtering.DNSAction {
					return filtering.DNSActionLocalHostPlusCache
				},
				Cache: map[string][]string{
					// Note: the cache here is nonexistent so we should
					// get a "no such host" error from the server.
				},
			}
			listener, err := srvr.Start("127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			dialer := NewDialerWithoutResolver(model.DiscardLogger)
			expectedAddress := listener.LocalAddr().String()
			txp := NewUnwrappedDNSOverUDPTransport(dialer, expectedAddress)
			encoder := &DNSEncoderMiekg{}
			query := encoder.Encode("dns.google.", dns.TypeA, false)
			zeroTime := time.Now()
			deterministicTime := testingx.NewTimeDeterministic(zeroTime)
			respChannel := make(chan *model.DNSResponse, 8)
			mu := new(sync.Mutex)
			tx := &mocks.Trace{
				MockTimeNow: deterministicTime.Now,
				MockOnDelayedDNSResponse: func(started time.Time, txp model.DNSTransport,
					query model.DNSQuery, response model.DNSResponse, addrs []string, err error,
					finished time.Time) error {
					mu.Lock()
					delayedDNSResponseCalled = true
					goodQueryType = (query.Type() == dns.TypeA)
					goodTransportNetwork = (txp.Network() == "udp")
					goodTransportAddress = (txp.Address() == expectedAddress)
					goodLookupAddrs = (len(addrs) == 0)
					goodError = errors.Is(err, ErrOODNSNoSuchHost)
					mu.Unlock()
					respChannel <- &response
					return errors.New("mocked") // return error to stop background routine to record responses
				},
				MockOnConnectDone: func(started time.Time, network, domain, remoteAddr string, err error,
					finished time.Time) {
					// do nothing
				},
				MockMaybeWrapNetConn: func(conn net.Conn) net.Conn {
					return conn
				},
			}
			ctx := ContextWithTrace(context.Background(), tx)
			rch, err := txp.RoundTrip(ctx, query)
			<-respChannel // wait for the delayed response
			if err != nil {
				t.Fatal(err)
			}
			addrs, err := rch.DecodeLookupHost()
			if err != nil {
				t.Fatal(err)
			}
			mu.Lock()
			if diff := cmp.Diff(addrs, []string{"127.0.0.1"}); diff != "" {
				t.Fatal(diff)
			}
			if !delayedDNSResponseCalled {
				t.Fatal("delayedDNSResponse not called")
			}
			if !goodQueryType {
				t.Fatal("unexpected query type")
			}
			if !goodTransportNetwork {
				t.Fatal("unexpected DNS transport network")
			}
			if !goodTransportAddress {
				t.Fatal("unexpected DNS Transport address")
			}
			if !goodLookupAddrs {
				t.Fatal("unexpected delayed DNSLookup address")
			}
			if !goodError {
				t.Fatal("unexpected error encountered")
			}
			mu.Unlock()
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
		txp := NewUnwrappedDNSOverUDPTransport(dialer, address)
		txp.CloseIdleConnections()
		if !called {
			t.Fatal("not called")
		}
	})

	t.Run("other functions okay", func(t *testing.T) {
		const address = "9.9.9.9:53"
		txp := NewUnwrappedDNSOverUDPTransport(NewDialerWithoutResolver(log.Log), address)
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
