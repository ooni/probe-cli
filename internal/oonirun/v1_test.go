package oonirun

import (
	"context"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/kvstore"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// TODO(bassosimone): it would be cool to write unit tests. However, to do that
// we need to ~redesign the engine package for unit-testability.

func TestOONIRunV1Link(t *testing.T) {
	ctx := context.Background()
	config := &LinkConfig{
		AcceptChanges: false,
		Annotations: map[string]string{
			"platform": "linux",
		},
		Callbacks:   model.NewPrinterCallbacks(model.DiscardLogger),
		KVStore:     &kvstore.Memory{},
		MaxRuntime:  0,
		NoCollector: true,
		NoJSON:      true,
		Random:      false,
		ReportFile:  "",
		Session:     newSession(ctx, t),
	}
	r := NewLinkRunner(config, "https://run.ooni.io/nettest?tn=example&mv=1.2.0")
	if err := r.Run(ctx); err != nil {
		t.Fatal(err)
	}
	r = NewLinkRunner(config, "ooni://nettest?tn=example&mv=1.2.0")
	if err := r.Run(ctx); err != nil {
		t.Fatal(err)
	}
}
