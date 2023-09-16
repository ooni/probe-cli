package testingsocks5

import (
	"bytes"
	"context"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestServerHandleConnect(t *testing.T) {
	t.Run("sendReply failure", func(t *testing.T) {
		// create a connection that fails as soon as we try to send
		expectedErr := errors.New("mocked error")
		cconn := &mocks.Conn{
			MockWrite: func(b []byte) (int, error) {
				return 0, expectedErr
			},
		}

		// create a netx where we fake dialing
		netx := &netxlite.Netx{
			Underlying: &mocks.UnderlyingNetwork{
				MockDialTimeout: func() time.Duration {
					return 15 * time.Second
				},
				MockDialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
					sconn := &mocks.Conn{
						MockClose: func() error {
							return nil
						},
						MockLocalAddr: func() net.Addr {
							return &net.TCPAddr{
								IP:   net.ParseIP("::17"),
								Port: 54321,
							}
						},
					}
					return sconn, nil
				},
			},
		}

		// create fake server and request
		server := &Server{
			closeOnce: sync.Once{},
			listener:  &mocks.Listener{}, // not used
			logger:    model.DiscardLogger,
			netx:      netx,
		}
		req := &request{
			Version: socks5Version,
			Command: connectCommand,
			DestAddr: &addrSpec{
				Address: "::55",
				Port:    80,
			},
		}

		err := server.handleConnect(context.Background(), cconn, req)
		if !errors.Is(err, expectedErr) {
			t.Fatal("unexpected error", err)
		}
	})
}

func TestSendReply(t *testing.T) {
	t.Run("we can serialize an IPv6 address", func(t *testing.T) {
		buffer := &bytes.Buffer{}
		err := sendReply(buffer, successReply, &net.TCPAddr{IP: net.ParseIP("::1"), Port: 80})
		if err != nil {
			t.Fatal(err)
		}
		expected := []byte{
			0x05,                   // version
			0x00,                   // successful response
			0x00,                   // reserved
			0x04,                   // IPv6
			0x00, 0x00, 0x00, 0x00, // ::1 (1/4)
			0x00, 0x00, 0x00, 0x00, // ::1 (2/4)
			0x00, 0x00, 0x00, 0x00, // ::1 (3/4)
			0x00, 0x00, 0x00, 0x01, // ::1 (4/4)
			0x00, 0x50, // port 80
		}
		if diff := cmp.Diff(expected, buffer.Bytes()); diff != "" {
			t.Fatal(diff)
		}
	})
}
