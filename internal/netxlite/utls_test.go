package netxlite

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	utls "gitlab.com/yawning/utls.git"
)

func TestNewTLSHandshakerUTLS(t *testing.T) {
	th := NewTLSHandshakerUTLS(log.Log, &utls.HelloChrome_83)
	logger := th.(*tlsHandshakerLogger)
	if logger.DebugLogger != log.Log {
		t.Fatal("invalid logger")
	}
	configurable := logger.TLSHandshaker.(*tlsHandshakerConfigurable)
	if configurable.NewConn == nil {
		t.Fatal("expected non-nil NewConn")
	}
}

func TestUTLSConn(t *testing.T) {
	t.Run("Handshake", func(t *testing.T) {
		t.Run("not interrupted with success", func(t *testing.T) {
			ctx := context.Background()
			conn := &utlsConn{
				testableHandshake: func() error {
					return nil
				},
			}
			err := conn.HandshakeContext(ctx)
			if err != nil {
				t.Fatal(err)
			}
		})

		t.Run("not interrupted with failure", func(t *testing.T) {
			expected := errors.New("mocked error")
			ctx := context.Background()
			conn := &utlsConn{
				testableHandshake: func() error {
					return expected
				},
			}
			err := conn.HandshakeContext(ctx)
			if !errors.Is(err, expected) {
				t.Fatal("not the error we expected", err)
			}
		})

		t.Run("interrupted", func(t *testing.T) {
			wg := sync.WaitGroup{}
			wg.Add(1)
			sigch := make(chan interface{})
			ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
			defer cancel()
			conn := &utlsConn{
				testableHandshake: func() error {
					defer wg.Done()
					<-sigch
					return nil
				},
			}
			err := conn.HandshakeContext(ctx)
			if !errors.Is(err, context.DeadlineExceeded) {
				t.Fatal("not the error we expected", err)
			}
			close(sigch)
			wg.Wait()
		})

		t.Run("with panic", func(t *testing.T) {
			wg := sync.WaitGroup{}
			wg.Add(1)
			ctx := context.Background()
			conn := &utlsConn{
				testableHandshake: func() error {
					defer wg.Done()
					panic("mascetti")
				},
			}
			err := conn.HandshakeContext(ctx)
			if !errors.Is(err, ErrUTLSHandshakePanic) {
				t.Fatal("not the error we expected", err)
			}
			wg.Wait()
		})
	})

	t.Run("NetConn", func(t *testing.T) {
		factory := newConnUTLS(&utls.HelloChrome_70)
		conn := &mocks.Conn{}
		tconn, err := factory(conn, &tls.Config{})
		if err != nil {
			t.Fatal(err)
		}
		if tconn.NetConn() != conn {
			t.Fatal("NetConn is not WAI")
		}
	})
}

func Test_newConnUTLSWithHelloID(t *testing.T) {
	tests := []struct {
		name        string
		config      *tls.Config
		cid         *utls.ClientHelloID
		wantNilConn bool
		wantErr     error
	}{{
		name: "with only supported fields",
		config: &tls.Config{
			DynamicRecordSizingDisabled: true,
			InsecureSkipVerify:          true,
			NextProtos:                  []string{"h3"},
			RootCAs:                     NewDefaultCertPool(),
			ServerName:                  "ooni.org",
		},
		cid:         &utls.HelloFirefox_55,
		wantNilConn: false,
		wantErr:     nil,
	}, {
		name: "with unsupported fields",
		config: &tls.Config{
			Time: func() time.Time {
				return time.Now()
			},
		},
		cid:         &utls.HelloChrome_58,
		wantNilConn: true,
		wantErr:     errUTLSIncompatibleStdlibConfig,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conn, err := net.Dial("udp", "8.8.8.8:443") // we just need a conn
			if err != nil {
				t.Fatal(err)
			}
			defer conn.Close()
			got, err := newConnUTLSWithHelloID(conn, tt.config, tt.cid)
			if !errors.Is(err, tt.wantErr) {
				t.Fatal("unexpected err", err)
			}
			if got != nil && tt.wantNilConn {
				t.Fatal("expected nil conn here")
			}
		})
	}
}
