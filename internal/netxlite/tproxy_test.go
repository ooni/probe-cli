package netxlite

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestDefaultTProxy(t *testing.T) {
	t.Run("DialContext honours the timeout", func(t *testing.T) {
		tp := &DefaultTProxy{}
		ctx := context.Background()
		timeout := time.Nanosecond
		conn, err := tp.DialContext(ctx, timeout, "tcp", "1.1.1.1:443")
		if err == nil || !strings.HasSuffix(err.Error(), "i/o timeout") {
			t.Fatal(err)
		}
		if conn != nil {
			t.Fatal("expected nil conn")
		}
	})
}
