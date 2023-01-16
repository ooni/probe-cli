// Command apitool is a simple tool to fetch individual OONI measurements.
//
// This tool IS NOT intended for batch downloading.
//
// Please, see https://ooni.org/data for information pertaining how to
// access OONI data in bulk. Please see https://explorer.ooni.org if your
// intent is to navigate and explore OONI data
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"sync/atomic"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine/probeservices"
	"github.com/ooni/probe-cli/v3/internal/httpx"
	"github.com/ooni/probe-cli/v3/internal/kvstore"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/version"
)

func newclient() probeservices.Client {
	txp := netxlite.NewHTTPTransportStdlib(log.Log)
	ua := fmt.Sprintf("apitool/%s ooniprobe-engine/%s", version.Version, version.Version)
	return probeservices.Client{
		APIClientTemplate: httpx.APIClientTemplate{
			BaseURL:    *backend,
			HTTPClient: &http.Client{Transport: txp},
			Logger:     log.Log,
			UserAgent:  ua,
		},
		LoginCalls:    &atomic.Int64{},
		RegisterCalls: &atomic.Int64{},
		StateFile:     probeservices.NewStateFile(&kvstore.Memory{}),
	}
}

var osExit = os.Exit

func fatalOnError(err error, message string) {
	if err != nil {
		log.WithError(err).Error(message)
		osExit(1) // overridable from tests
	}
}

var (
	backend  = flag.String("backend", "https://api.ooni.io/", "Backend to use")
	debug    = flag.Bool("v", false, "Enable verbose mode")
	input    = flag.String("input", "", "Input of the measurement")
	mode     = flag.String("mode", "", "One of: check, meta, raw")
	reportid = flag.String("report-id", "", "Report ID of the measurement")
)

var logmap = map[bool]log.Level{
	true:  log.DebugLevel,
	false: log.InfoLevel,
}

func main() {
	flag.Parse()
	log.SetLevel(logmap[*debug])
	client := newclient()
	switch *mode {
	case "meta":
		meta(client)
	case "raw":
		raw(client)
	default:
		fatalOnError(fmt.Errorf("invalid -mode flag value: %s", *mode), "usage error")
	}
}

func meta(c probeservices.Client) {
	pprint(mmeta(c, false))
}

func raw(c probeservices.Client) {
	m := mmeta(c, true)
	rm := []byte(m.RawMeasurement)
	var opaque interface{}
	err := json.Unmarshal(rm, &opaque)
	fatalOnError(err, "json.Unmarshal failed")
	pprint(opaque)
}

func pprint(opaque interface{}) {
	data, err := json.MarshalIndent(opaque, "", "  ")
	fatalOnError(err, "json.MarshalIndent failed")
	fmt.Printf("%s\n", data)
}

func mmeta(c probeservices.Client, full bool) *model.OOAPIMeasurementMeta {
	config := model.OOAPIMeasurementMetaConfig{
		ReportID: *reportid,
		Full:     full,
		Input:    *input,
	}
	ctx := context.Background()
	m, err := c.GetMeasurementMeta(ctx, config)
	fatalOnError(err, "client.GetMeasurementMeta failed")
	return m
}
