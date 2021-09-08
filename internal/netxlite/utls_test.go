package netxlite

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/apex/log"
	utls "gitlab.com/yawning/utls.git"
)

func TestNewTLSHandshakerUTLS(t *testing.T) {
	th := NewTLSHandshakerUTLS(log.Log, &utls.HelloChrome_83)
	logger := th.(*tlsHandshakerLogger)
	if logger.Logger != log.Log {
		t.Fatal("invalid logger")
	}
	errWrapper := logger.TLSHandshaker.(*tlsHandshakerErrWrapper)
	configurable := errWrapper.TLSHandshaker.(*tlsHandshakerConfigurable)
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
}
