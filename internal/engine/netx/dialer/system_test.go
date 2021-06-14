package dialer

import (
	"strings"
	"testing"
	"time"

	"github.com/ooni/psiphon/oopsi/golang.org/x/net/context"
)

func TestSystemDialerWorks(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // fail immediately
	conn, err := systemDialer.DialContext(ctx, "tcp", "8.8.8.8:853")
	if err == nil || !strings.HasSuffix(err.Error(), "operation was canceled") {
		t.Fatal("not the error we expected", err)
	}
	if conn != nil {
		t.Fatal("expected nil conn here")
	}
}

func TestSystemDialerHasTimeout(t *testing.T) {
	expected := 15 * time.Second
	if systemDialer.Timeout != expected {
		t.Fatal("unexpected timeout value")
	}
}
