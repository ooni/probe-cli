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
	Endpoints []*measurex.Measurement
}

func main() {
	URL := flag.String("url", "https://blog.cloudflare.com/", "URL to fetch")
	address := flag.String("address", "8.8.4.4:53", "DNS-over-UDP server address")
	timeout := flag.Duration("timeout", 60*time.Second, "timeout to use")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	parsed, err := url.Parse(*URL)
	runtimex.PanicOnError(err, "url.Parse failed")
	mx := measurex.NewMeasurerWithDefaultSettings()
	m := &measurement{}
	m.DNS = append(m.DNS, mx.LookupHostUDP(ctx, parsed.Hostname(), *address))
	m.DNS = append(m.DNS, mx.LookupHTTPSSvcUDP(ctx, parsed.Hostname(), *address))
	httpEndpoints, err := mx.DB.SelectAllHTTPEndpointsForURL(parsed)
	runtimex.PanicOnError(err, "cannot get all the HTTP endpoints")
	cookies := measurex.NewCookieJar()
	for _, epnt := range httpEndpoints {
		m.Endpoints = append(m.Endpoints, mx.HTTPEndpointGet(ctx, epnt, cookies))
	}
	data, err := json.Marshal(m)
	runtimex.PanicOnError(err, "json.Marshal failed")
	fmt.Printf("%s\n", string(data))
}
