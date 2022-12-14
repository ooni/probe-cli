package probeservices

import (
	"context"
	"net/http"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/atomicx"
	"github.com/ooni/probe-cli/v3/internal/httpx"
	"github.com/ooni/probe-cli/v3/internal/kvstore"
)

func TestCheckReportIDWorkingAsIntended(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	client := Client{
		APIClientTemplate: httpx.APIClientTemplate{
			BaseURL:    "https://api.ooni.io/",
			HTTPClient: http.DefaultClient,
			Logger:     log.Log,
			UserAgent:  "miniooni/0.1.0-dev",
		},
		LoginCalls:    &atomicx.Int64{},
		RegisterCalls: &atomicx.Int64{},
		StateFile:     NewStateFile(&kvstore.Memory{}),
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
