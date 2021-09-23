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
	DNS       []*measurex.DNSMeasurement
	Endpoints []*measurex.HTTPEndpointMeasurement
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
	mx.Resolvers = []*measurex.ResolverInfo{{
		Network: measurex.ResolverUDP,
		Address: "8.8.8.8:53",
	}, {
		Network: measurex.ResolverUDP,
		Address: "8.8.4.4:53",
	}, {
		Network: measurex.ResolverUDP,
		Address: "1.1.1.1:53",
	}, {
		Network: measurex.ResolverUDP,
		Address: "1.0.0.1:53",
	}}
	m := &measurement{}
	for dns := range mx.LookupURLHostParallel(ctx, parsed) {
		m.DNS = append(m.DNS, dns)
	}
	headers := measurex.NewHTTPRequestHeaderForMeasuring()
	httpEndpoints, err := measurex.AllHTTPEndpointsForURL(parsed, headers, m.DNS...)
	runtimex.PanicOnError(err, "cannot get all the HTTP endpoints")
	cookies := measurex.NewCookieJar()
	for epnt := range mx.HTTPEndpointGetParallel(ctx, cookies, httpEndpoints...) {
		m.Endpoints = append(m.Endpoints, epnt)
	}
	data, err := json.Marshal(m)
	runtimex.PanicOnError(err, "json.Marshal failed")
	fmt.Printf("%s\n", string(data))
}
