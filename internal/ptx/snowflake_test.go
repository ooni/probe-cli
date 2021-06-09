package ptx

import (
	"context"
	"testing"
)

func TestSnowflakeDialerWorks(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	sfd := &SnowflakeDialer{}
	conn, err := sfd.DialContext(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if conn == nil {
		t.Fatal("expected non-nil conn here")
	}
}
