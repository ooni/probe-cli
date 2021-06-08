package dialer

import (
	"strings"
	"testing"

	"github.com/ooni/psiphon/oopsi/golang.org/x/net/context"
)

func TestSystemDialer(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // fail immediately
	conn, err := Default.DialContext(ctx, "tcp", "8.8.8.8:853")
	if err == nil || !strings.HasSuffix(err.Error(), "operation was canceled") {
		t.Fatal("not the error we expected", err)
	}
	if conn != nil {
		t.Fatal("expected nil conn here")
	}
}
