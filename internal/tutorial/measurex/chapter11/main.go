package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"time"

	"github.com/ooni/probe-cli/v3/internal/measurex"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func main() {
	URL := flag.String("url", "https://blog.cloudflare.com/", "URL to fetch")
	timeout := flag.Duration("timeout", 60*time.Second, "timeout to use")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	mx := measurex.NewMeasurerWithDefaultSettings()
	cookies := measurex.NewCookieJar()
	headers := measurex.NewHTTPRequestHeaderForMeasuring()
	m, err := mx.MeasureURL(ctx, *URL, headers, cookies)
	runtimex.PanicOnError(err, "mx.MeasureURL failed")
	data, err := json.Marshal(m)
	runtimex.PanicOnError(err, "json.Marshal failed")
	fmt.Printf("%s\n", string(data))
}
