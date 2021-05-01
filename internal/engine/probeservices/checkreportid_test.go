package probeservices_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/atomicx"
	"github.com/ooni/probe-cli/v3/internal/engine/httpx"
	"github.com/ooni/probe-cli/v3/internal/engine/kvstore"
	"github.com/ooni/probe-cli/v3/internal/engine/probeservices"
)

func TestCheckReportIDWorkingAsIntended(t *testing.T) {
	client := probeservices.Client{
		Client: httpx.Client{
			BaseURL:    "https://ams-pg.ooni.org/",
			HTTPClient: http.DefaultClient,
			Logger:     log.Log,
			UserAgent:  "miniooni/0.1.0-dev",
		},
		LoginCalls:    &atomicx.Int64{},
		RegisterCalls: &atomicx.Int64{},
		StateFile:     probeservices.NewStateFile(kvstore.NewMemoryKeyValueStore()),
	}
	reportID := `20201209T052225Z_urlgetter_IT_30722_n1_E1VUhMz08SEkgYFU`
	ctx := context.Background()
	found, err := client.CheckReportID(ctx, reportID)
	if err != nil {
		t.Fatal(err)
	}
	if found != true {
		t.Fatal("unexpected found value")
	}
}

func TestCheckReportIDWorkingWithCancelledContext(t *testing.T) {
	client := probeservices.Client{
		Client: httpx.Client{
			BaseURL:    "https://ams-pg.ooni.org/",
			HTTPClient: http.DefaultClient,
			Logger:     log.Log,
			UserAgent:  "miniooni/0.1.0-dev",
		},
		LoginCalls:    &atomicx.Int64{},
		RegisterCalls: &atomicx.Int64{},
		StateFile:     probeservices.NewStateFile(kvstore.NewMemoryKeyValueStore()),
	}
	reportID := `20201209T052225Z_urlgetter_IT_30722_n1_E1VUhMz08SEkgYFU`
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // fail immediately
	found, err := client.CheckReportID(ctx, reportID)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if found != false {
		t.Fatal("unexpected found value")
	}
}
