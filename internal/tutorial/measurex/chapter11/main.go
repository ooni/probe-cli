package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"time"

	"github.com/ooni/probe-cli/v3/internal/measurex"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

type measurement struct {
	DNS       []*measurex.Measurement
	TH        []*measurex.Measurement
	Endpoints []*measurex.Measurement
}

func main() {
	URL := flag.String("url", "https://blog.cloudflare.com/", "URL to fetch")
	timeout := flag.Duration("timeout", 60*time.Second, "timeout to use")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	parsed, err := url.Parse(*URL)
	runtimex.PanicOnError(err, "url.Parse failed")
	mx := measurex.NewMeasurerWithDefaultSettings()
	m := &measurement{}
	mx.RegisterUDPResolvers("8.8.8.8:53", "8.8.4.4:53", "1.1.1.1:53", "1.0.0.1:53")
	for dns := range mx.LookupURLHostParallel(ctx, parsed) {
		m.DNS = append(m.DNS, dns)
	}
	mx.RegisterWCTH("https://wcth.ooni.io/")
	for th := range mx.QueryTestHelperParallel(ctx, parsed) {
		m.TH = append(m.TH, th)
	}
	httpEndpoints, err := mx.DB.SelectAllHTTPEndpointsForURL(parsed)
	runtimex.PanicOnError(err, "cannot get all the HTTP endpoints")
	cookies := measurex.NewCookieJar()
	for epnt := range mx.HTTPEndpointGetParallel(ctx, cookies, httpEndpoints...) {
		m.Endpoints = append(m.Endpoints, epnt)
	}
	data, err := json.Marshal(m)
	runtimex.PanicOnError(err, "json.Marshal failed")
	fmt.Printf("%s\n", string(data))
}
