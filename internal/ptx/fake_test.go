package ptx

import (
	"context"
	"testing"
)

func TestFakeDialerWorks(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}

	fd := &FakeDialer{Address: "8.8.8.8:53"}
	conn, err := fd.DialContext(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if fd.Name() != "fake" {
		t.Fatal("invalid value returned by fd.Name")
	}
	if fd.AsBridgeArgument() != "fake 8.8.8.8:53" {
		t.Fatal("invalid value returned by fd.AsBridgeArgument")
	}
	conn.Close()
}
