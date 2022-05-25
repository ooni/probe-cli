package netxlite

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"testing"

	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/atomicx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
)

// errorWithTimeout is an error that golang will always consider
// to be a timeout because it has a Timeout() bool method
type errorWithTimeout struct {
	error
}

// Timeout returns whether this error is a timeout.
func (err *errorWithTimeout) Timeout() bool {
	return true
}

// Unwrap allows to unwrap the error.
func (err *errorWithTimeout) Unwrap() error {
	return err.error
}

func TestSerialResolver(t *testing.T) {
	t.Run("transport okay", func(t *testing.T) {
		txp := NewDNSOverTLS((&tls.Dialer{}).DialContext, "8.8.8.8:853")
		r := NewSerialResolver(txp)
		rtx := r.Transport()
		if rtx.Network() != "dot" || rtx.Address() != "8.8.8.8:853" {
			t.Fatal("not the transport we expected")
		}
		if r.Network() != rtx.Network() {
			t.Fatal("invalid network seen from the resolver")
		}
		if r.Address() != rtx.Address() {
			t.Fatal("invalid address seen from the resolver")
		}
	})

	t.Run("LookupHost", func(t *testing.T) {
		t.Run("RoundTrip error", func(t *testing.T) {
			mocked := errors.New("mocked error")
			txp := &mocks.DNSTransport{
				MockRoundTrip: func(ctx context.Context, query model.DNSQuery) (model.DNSResponse, error) {
					return nil, mocked
				},
				MockRequiresPadding: func() bool {
					return true
				},
			}
			r := NewSerialResolver(txp)
			addrs, err := r.LookupHost(context.Background(), "www.gogle.com")
			if !errors.Is(err, mocked) {
				t.Fatal("not the error we expected")
			}
			if addrs != nil {
				t.Fatal("expected nil address here")
			}
		})

		t.Run("empty reply", func(t *testing.T) {
			txp := &mocks.DNSTransport{
				MockRoundTrip: func(ctx context.Context, query model.DNSQuery) (model.DNSResponse, error) {
					response := &mocks.DNSResponse{
						MockDecodeLookupHost: func() ([]string, error) {
							return nil, nil
						},
					}
					return response, nil
				},
				MockRequiresPadding: func() bool {
					return true
				},
			}
			r := NewSerialResolver(txp)
			addrs, err := r.LookupHost(context.Background(), "www.gogle.com")
			if !errors.Is(err, ErrOODNSNoAnswer) {
				t.Fatal("not the error we expected", err)
			}
			if addrs != nil {
				t.Fatal("expected nil address here")
			}
		})

		t.Run("with A reply", func(t *testing.T) {
			txp := &mocks.DNSTransport{
				MockRoundTrip: func(ctx context.Context, query model.DNSQuery) (model.DNSResponse, error) {
					response := &mocks.DNSResponse{
						MockDecodeLookupHost: func() ([]string, error) {
							if query.Type() != dns.TypeA {
								return nil, nil
							}
							return []string{"8.8.8.8"}, nil
						},
					}
					return response, nil
				},
				MockRequiresPadding: func() bool {
					return true
				},
			}
			r := NewSerialResolver(txp)
			addrs, err := r.LookupHost(context.Background(), "www.gogle.com")
			if err != nil {
				t.Fatal(err)
			}
			if len(addrs) != 1 || addrs[0] != "8.8.8.8" {
				t.Fatal("not the result we expected")
			}
		})

		t.Run("with AAAA reply", func(t *testing.T) {
			txp := &mocks.DNSTransport{
				MockRoundTrip: func(ctx context.Context, query model.DNSQuery) (model.DNSResponse, error) {
					response := &mocks.DNSResponse{
						MockDecodeLookupHost: func() ([]string, error) {
							if query.Type() != dns.TypeAAAA {
								return nil, nil
							}
							return []string{"::1"}, nil
						},
					}
					return response, nil
				},
				MockRequiresPadding: func() bool {
					return true
				},
			}
			r := NewSerialResolver(txp)
			addrs, err := r.LookupHost(context.Background(), "www.gogle.com")
			if err != nil {
				t.Fatal(err)
			}
			if len(addrs) != 1 || addrs[0] != "::1" {
				t.Fatal("not the result we expected")
			}
		})

		t.Run("with timeout", func(t *testing.T) {
			txp := &mocks.DNSTransport{
				MockRoundTrip: func(ctx context.Context, query model.DNSQuery) (model.DNSResponse, error) {
					err := &net.OpError{
						Err: &errorWithTimeout{ETIMEDOUT},
						Op:  "dial",
					}
					return nil, err
				},
				MockRequiresPadding: func() bool {
					return true
				},
			}
			r := NewSerialResolver(txp)
			addrs, err := r.LookupHost(context.Background(), "www.gogle.com")
			if !errors.Is(err, ETIMEDOUT) {
				t.Fatal("not the error we expected")
			}
			if addrs != nil {
				t.Fatal("expected nil address here")
			}
			if r.NumTimeouts.Load() <= 0 {
				t.Fatal("we didn't actually take the timeouts")
			}
		})
	})

	t.Run("CloseIdleConnections", func(t *testing.T) {
		var called bool
		r := &SerialResolver{
			Txp: &mocks.DNSTransport{
				MockCloseIdleConnections: func() {
					called = true
				},
			},
		}
		r.CloseIdleConnections()
		if !called {
			t.Fatal("not called")
		}
	})

	t.Run("LookupHTTPS", func(t *testing.T) {
		t.Run("for round-trip error", func(t *testing.T) {
			expected := errors.New("mocked error")
			r := &SerialResolver{
				NumTimeouts: &atomicx.Int64{},
				Txp: &mocks.DNSTransport{
					MockRoundTrip: func(ctx context.Context, query model.DNSQuery) (model.DNSResponse, error) {
						return nil, expected
					},
					MockRequiresPadding: func() bool {
						return false
					},
				},
			}
			ctx := context.Background()
			https, err := r.LookupHTTPS(ctx, "example.com")
			if !errors.Is(err, expected) {
				t.Fatal("unexpected err", err)
			}
			if https != nil {
				t.Fatal("unexpected result")
			}
		})

		t.Run("for decode error", func(t *testing.T) {
			expected := errors.New("mocked error")
			r := &SerialResolver{
				NumTimeouts: &atomicx.Int64{},
				Txp: &mocks.DNSTransport{
					MockRoundTrip: func(ctx context.Context, query model.DNSQuery) (model.DNSResponse, error) {
						response := &mocks.DNSResponse{
							MockDecodeHTTPS: func() (*model.HTTPSSvc, error) {
								return nil, expected
							},
						}
						return response, nil
					},
					MockRequiresPadding: func() bool {
						return false
					},
				},
			}
			ctx := context.Background()
			https, err := r.LookupHTTPS(ctx, "example.com")
			if !errors.Is(err, expected) {
				t.Fatal("unexpected err", err)
			}
			if https != nil {
				t.Fatal("unexpected result")
			}
		})
	})

	t.Run("LookupNS", func(t *testing.T) {
		t.Run("for round-trip error", func(t *testing.T) {
			expected := errors.New("mocked error")
			r := &SerialResolver{
				NumTimeouts: &atomicx.Int64{},
				Txp: &mocks.DNSTransport{
					MockRoundTrip: func(ctx context.Context, query model.DNSQuery) (model.DNSResponse, error) {
						return nil, expected
					},
					MockRequiresPadding: func() bool {
						return false
					},
				},
			}
			ctx := context.Background()
			ns, err := r.LookupNS(ctx, "example.com")
			if !errors.Is(err, expected) {
				t.Fatal("unexpected err", err)
			}
			if ns != nil {
				t.Fatal("unexpected result")
			}
		})

		t.Run("for decode error", func(t *testing.T) {
			expected := errors.New("mocked error")
			r := &SerialResolver{
				NumTimeouts: &atomicx.Int64{},
				Txp: &mocks.DNSTransport{
					MockRoundTrip: func(ctx context.Context, query model.DNSQuery) (model.DNSResponse, error) {
						response := &mocks.DNSResponse{
							MockDecodeNS: func() ([]*net.NS, error) {
								return nil, expected
							},
						}
						return response, nil
					},
					MockRequiresPadding: func() bool {
						return false
					},
				},
			}
			ctx := context.Background()
			https, err := r.LookupNS(ctx, "example.com")
			if !errors.Is(err, expected) {
				t.Fatal("unexpected err", err)
			}
			if https != nil {
				t.Fatal("unexpected result")
			}
		})
	})
}
