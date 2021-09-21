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

type measurement struct {
	URLs []*measurex.URLMeasurement
}

func main() {
	URL := flag.String("url", "https://blog.cloudflare.com/", "URL to fetch")
	timeout := flag.Duration("timeout", 60*time.Second, "timeout to use")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	mx := measurex.NewMeasurerWithDefaultSettings()
	mx.RegisterWCTH("https://wcth.ooni.io/")
	mx.RegisterUDPResolvers("8.8.8.8:53", "8.8.4.4:53", "1.1.1.1:53", "1.0.0.1:53")
	cookies := measurex.NewCookieJar()
	all := &measurement{}
	for m := range mx.MeasureHTTPURLAndFollowRedirections(ctx, *URL, cookies) {
		all.URLs = append(all.URLs, m)
	}
	data, err := json.Marshal(all)
	runtimex.PanicOnError(err, "json.Marshal failed")
	fmt.Printf("%s\n", string(data))
}
