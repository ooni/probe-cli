package resolver

import (
	"context"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/trace"
)

// stopAndWait waits for all backgroung goroutines to join. We only
// use this method in the tests to be sure we don't leak any goroutine.
func (svc *Service) stopAndWait() {
	svc.Stop()
	svc.wg.Wait()
}

func TestSuccessfulResolution(t *testing.T) {
	r := New()
	r.StartN(10)
	defer r.stopAndWait()
	saver := &trace.Saver{}
	ctx := context.Background()
	addrs, err := r.LookupHost(ctx, &LookupHostRequest{
		Domain: "dns.google",
		Logger: log.Log,
		Saver:  saver,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(addrs) <= 0 {
		t.Fatal("no returned addrs?!")
	}
	events := saver.Read()
	if len(events) <= 0 {
		t.Fatal("no returned events?!")
	}
}

func TestFailingResolution(t *testing.T) {
	r := New()
	r.StartN(10)
	defer r.stopAndWait()
	saver := &trace.Saver{}
	ctx := context.Background()
	addrs, err := r.LookupHost(ctx, &LookupHostRequest{
		Domain: "dns.antani",
		Logger: log.Log,
		Saver:  saver,
	})
	if err == nil || err.Error() != "dns_nxdomain_error" {
		t.Fatal("not the error we expected", err)
	}
	if len(addrs) > 0 {
		t.Fatal("returned any addrs?!")
	}
	events := saver.Read()
	if len(events) <= 0 {
		t.Fatal("no returned events?!")
	}
}
