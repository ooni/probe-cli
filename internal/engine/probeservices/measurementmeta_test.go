package probeservices_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/atomicx"
	"github.com/ooni/probe-cli/v3/internal/engine/httpx"
	"github.com/ooni/probe-cli/v3/internal/engine/kvstore"
	"github.com/ooni/probe-cli/v3/internal/engine/probeservices"
)

func TestGetMeasurementMetaWorkingAsIntended(t *testing.T) {
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
	config := probeservices.MeasurementMetaConfig{
		ReportID: `20201209T052225Z_urlgetter_IT_30722_n1_E1VUhMz08SEkgYFU`,
		Full:     true,
		Input:    `https://www.example.org`,
	}
	ctx := context.Background()
	mmeta, err := client.GetMeasurementMeta(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	if mmeta.Anomaly != false {
		t.Fatal("unexpected anomaly value")
	}
	if mmeta.CategoryCode != "" {
		t.Fatal("unexpected category code value")
	}
	if mmeta.Confirmed != false {
		t.Fatal("unexpected confirmed value")
	}
	if mmeta.Failure != true {
		// TODO(bassosimone): this field seems wrong
		t.Fatal("unexpected failure value")
	}
	if mmeta.Input == nil || *mmeta.Input != config.Input {
		t.Fatal("unexpected input value")
	}
	if mmeta.MeasurementStartTime.String() != "2020-12-09 05:22:25 +0000 UTC" {
		t.Fatal("unexpected measurement start time value")
	}
	if mmeta.ProbeASN != 30722 {
		t.Fatal("unexpected probe asn value")
	}
	if mmeta.ProbeCC != "IT" {
		t.Fatal("unexpected probe cc value")
	}
	if mmeta.ReportID != config.ReportID {
		t.Fatal("unexpected report id value")
	}
	// TODO(bassosimone): we could better this check
	var scores interface{}
	if err := json.Unmarshal([]byte(mmeta.Scores), &scores); err != nil {
		t.Fatalf("cannot parse scores value: %+v", err)
	}
	if mmeta.TestName != "urlgetter" {
		t.Fatal("unexpected test name value")
	}
	if mmeta.TestStartTime.String() != "2020-12-09 05:22:25 +0000 UTC" {
		t.Fatal("unexpected test start time value")
	}
	// TODO(bassosimone): we could better this check
	var rawmeas interface{}
	if err := json.Unmarshal([]byte(mmeta.RawMeasurement), &rawmeas); err != nil {
		t.Fatalf("cannot parse raw measurement: %+v", err)
	}
}

func TestGetMeasurementMetaWorkingWithCancelledContext(t *testing.T) {
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
	config := probeservices.MeasurementMetaConfig{
		ReportID: `20201209T052225Z_urlgetter_IT_30722_n1_E1VUhMz08SEkgYFU`,
		Full:     true,
		Input:    `https://www.example.org`,
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // fail immediately
	mmeta, err := client.GetMeasurementMeta(ctx, config)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if mmeta != nil {
		t.Fatal("we expected a nil mmeta here")
	}
}
