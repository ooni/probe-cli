package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"time"

	"github.com/ooni/probe-cli/v3/internal/measurex"
	"github.com/ooni/probe-cli/v3/internal/netxlite/iox"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func main() {
	sni := flag.String("sni", "dns.google", "value for SNI extension")
	address := flag.String("address", "8.8.4.4:443", "remote endpoint address")
	timeout := flag.Duration("timeout", 60*time.Second, "timeout to use")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	mx := measurex.NewMeasurerWithDefaultSettings()
	epnt := &measurex.HTTPEndpoint{
		Domain:  *sni,
		Network: "tcp",
		Address: *address,
		SNI:     *sni,
		ALPN:    []string{"h2", "http/1.1"},
		URL: &url.URL{
			Scheme: "https",
			Host:   *sni,
			Path:   "/",
		},
		Header: measurex.NewHTTPRequestHeaderForMeasuring(),
	}
	cookies := measurex.NewCookieJar()
	prep := mx.HTTPEndpointPrepareGet(ctx, epnt, cookies)
	m := prep.Measurement()
	resp, err := prep.Resume()
	if err == nil {
		data, err := iox.ReadAllContext(ctx, resp.Body)
		if err == nil {
			fmt.Printf("{\"full body size\": %d}\n", len(data))
		}
		resp.Body.Close()
	}
	data, err := json.Marshal(m)
	runtimex.PanicOnError(err, "json.Marshal failed")
	fmt.Printf("%s\n", string(data))
}
