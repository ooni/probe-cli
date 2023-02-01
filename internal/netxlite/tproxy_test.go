package netxlite

import (
	"context"
	"crypto/x509"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func TestDefaultTProxy(t *testing.T) {
	t.Run("DialContext honours the timeout", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			// This test is here to give us confidence we're doing the right thing
			// in terms of the underlying Go API. It's not here to make sure the
			// github CI behaves exactly equally on Windows, Linux, macOS on edge
			// cases. So, it seems fine to just skip this test on Windows.
			//
			// TODO(https://github.com/ooni/probe/issues/2368).
			t.Skip("skip test on windows")
		}
		tp := &DefaultTProxy{}
		ctx := context.Background()
		conn, err := tp.DialContext(ctx, time.Nanosecond, "tcp", "1.1.1.1:443")
		if err == nil || !strings.HasSuffix(err.Error(), "i/o timeout") {
			t.Fatal("unexpected err", err)
		}
		if conn != nil {
			t.Fatal("expected nil conn")
		}
	})
}

func TestWithCustomTProxy(t *testing.T) {
	expected := x509.NewCertPool()
	tproxy := &mocks.UnderlyingNetwork{
		MockMaybeModifyPool: func(pool *x509.CertPool) *x509.CertPool {
			runtimex.Assert(expected != pool, "got unexpected pool")
			return expected
		},
	}
	WithCustomTProxy(tproxy, func() {
		if NewDefaultCertPool() != expected {
			t.Fatal("unexpected pool")
		}
	})
}
