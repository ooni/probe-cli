package netxlite

import (
	"context"
	"runtime"
	"strings"
	"testing"
	"time"
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
