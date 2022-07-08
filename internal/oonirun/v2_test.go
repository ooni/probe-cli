package oonirun

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/kvstore"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// TODO(bassosimone): it would be cool to write unit tests. However, to do that
// we need to ~redesign the engine package for unit-testability.

func TestOONIRunV2Link(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		descriptor := &v2Descriptor{
			Name:        "",
			Description: "",
			Author:      "",
			Nettests: []v2Nettest{{
				Inputs: []string{},
				Options: map[string]any{
					"SleepTime": int64(10 * time.Millisecond),
				},
				TestName: "example",
			}},
		}
		data, err := json.Marshal(descriptor)
		runtimex.PanicOnError(err, "json.Marshal failed")
		w.Write(data)
	}))
	defer server.Close()
	ctx := context.Background()
	config := &LinkConfig{
		AcceptChanges: true, // avoid "oonirun: need to accept changes" error
		Annotations: map[string]string{
			"platform": "linux",
		},
		KVStore:     &kvstore.Memory{},
		MaxRuntime:  0,
		NoCollector: true,
		NoJSON:      true,
		Random:      false,
		ReportFile:  "",
		Session:     newSession(ctx, t),
	}
	r := NewLinkRunner(config, server.URL)
	if err := r.Run(ctx); err != nil {
		t.Fatal(err)
	}
}
