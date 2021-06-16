package connector

import (
	"context"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/trace"
)

// stopAndWait waits for all backgroung goroutines to join. We only
// use this method in the tests to be sure we don't leak any goroutine.
func (svc *Service) stopAndWait() {
	svc.Stop()
	svc.wg.Wait()
}

func TestSuccessfulDial(t *testing.T) {
	d := New()
	d.StartN(10)
	defer d.stopAndWait()
	saver := &trace.Saver{}
	ctx := context.Background()
	conn, err := d.DialContext(ctx, &DialRequest{
		Network: "tcp",
		Address: "8.8.8.8:53",
		Logger:  log.Log,
		Saver:   saver,
	})
	if err != nil {
		t.Fatal(err)
	}
	if conn == nil {
		t.Fatal("nil conn?!")
	}
	defer conn.Close()
	events := saver.Read()
	if len(events) <= 0 {
		t.Fatal("no returned events?!")
	}
}

func TestFailingDial(t *testing.T) {
	d := New()
	d.StartN(10)
	defer d.stopAndWait()
	saver := &trace.Saver{}
	ctx, cancel := context.WithTimeout(context.Background(), 250*time.Microsecond)
	defer cancel()
	conn, err := d.DialContext(ctx, &DialRequest{
		Network: "tcp",
		Address: "8.8.8.8:53",
		Logger:  log.Log,
		Saver:   saver,
	})
	if err == nil || err.Error() != "generic_timeout_error" {
		t.Fatal("not the error we expected", err)
	}
	if conn != nil {
		t.Fatal("returned a valid conn?!")
	}
	events := saver.Read()
	if len(events) <= 0 {
		t.Fatal("no returned events?!")
	}
}
