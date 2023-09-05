package testingx

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/mocks"
)

func TestTLSHandlerTimeoutWithShortTimeout(t *testing.T) {
	thx := &tlsHandlerTimeout{
		timeout: 250 * time.Microsecond,
	}
	conn := &mocks.Conn{
		MockClose: func() error {
			return nil
		},
	}
	cert, err := thx.GetCertificate(context.Background(), conn, &tls.ClientHelloInfo{})
	if err == nil || err.Error() != "internal error" {
		t.Fatal("unexpected error", err)
	}
	if cert != nil {
		t.Fatal("expected nil cert")
	}
}

func TestTLSServerMainLoopTransientError(t *testing.T) {
	called := &atomic.Bool{}
	tlx := &TLSServer{
		cancel: func() {
			// nothing to do here
		},
		closeOnce: sync.Once{},
		endpoint:  "10.0.0.1:443",
		handler:   nil, // not used
		listener: &mocks.Listener{
			MockAccept: func() (net.Conn, error) {
				if called.Load() {
					return nil, net.ErrClosed
				}
				called.Store(true)
				return nil, errors.New("mocked error")
			},
			MockClose: func() error {
				return nil
			},
			MockAddr: func() net.Addr {
				return &mocks.Addr{
					MockString: func() string {
						return "10.0.0.1:443"
					},
					MockNetwork: func() string {
						return "tcp"
					},
				}
			},
		},
		wg: sync.WaitGroup{},
	}

	tlx.wg.Add(1)
	go tlx.mainloop(context.Background())
	tlx.wg.Wait()
}
