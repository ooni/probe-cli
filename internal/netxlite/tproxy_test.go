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
		conn, err := tp.DialContext(ctx, 100*time.Microsecond, "tcp", "1.1.1.1:443")
		if err == nil || !strings.HasSuffix(err.Error(), "i/o timeout") {
			t.Fatal(err)
		}
		if conn != nil {
			t.Fatal("expected nil conn")
		}
	})
}
