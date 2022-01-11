package archival

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/fakefill"
)

func TestSaverNewSaver(t *testing.T) {
	saver := NewSaver()
	if saver.trace == nil {
		t.Fatal("expected non-nil trace here")
	}
}

func TestSaverMoveOutTrace(t *testing.T) {
	saver := NewSaver()
	var ev DNSRoundTripEvent
	ff := &fakefill.Filler{}
	ff.Fill(&ev)
	if len(ev.Query) < 1 {
		t.Fatal("did not fill") // be sure
	}
	saver.appendDNSRoundTripEvent(&ev)
	trace := saver.MoveOutTrace()
	if len(saver.trace.DNSRoundTrip) != 0 {
		t.Fatal("expected zero length")
	}
	if len(trace.DNSRoundTrip) != 1 {
		t.Fatal("expected one entry")
	}
	entry := trace.DNSRoundTrip[0]
	if diff := cmp.Diff(&ev, entry); diff != "" {
		t.Fatal(diff)
	}
}
