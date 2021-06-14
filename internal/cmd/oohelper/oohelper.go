// Command oohelper contains a simple command line
// client for the Web Connectivity test helper.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/cmd/oohelper/internal"
	"github.com/ooni/probe-cli/v3/internal/engine/netx"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

var (
	ctx, cancel = context.WithCancel(context.Background())
	debug       = flag.Bool("debug", false, "Toggle debug mode")
	httpClient  *http.Client
	resolver    netx.Resolver
	server      = flag.String("server", "https://wcth.ooni.io/", "URL of the test helper")
	target      = flag.String("target", "", "Target URL for the test helper")
)

func newhttpclient() *http.Client {
	// Use a nonstandard resolver, which is enough to work around the
	// puzzling https://github.com/ooni/probe/issues/1409 issue.
	childResolver, err := netx.NewDNSClient(
		netx.Config{Logger: log.Log}, "dot://8.8.8.8:853")
	runtimex.PanicOnError(err, "netx.NewDNSClient should not fail here")
	txp := netx.NewHTTPTransport(netx.Config{
		BaseResolver: childResolver,
		Logger:       log.Log,
	})
	return &http.Client{Transport: txp}
}

func init() {
	httpClient = newhttpclient()
	resolver = netx.NewResolver(netx.Config{Logger: log.Log})
}

func main() {
	logmap := map[bool]log.Level{
		true:  log.DebugLevel,
		false: log.InfoLevel,
	}
	flag.Parse()
	log.SetLevel(logmap[*debug])
	clnt := internal.OOClient{HTTPClient: httpClient, Resolver: resolver}
	config := internal.OOConfig{TargetURL: *target, ServerURL: *server}
	defer cancel()
	cresp, err := clnt.Do(ctx, config)
	runtimex.PanicOnError(err, "client.Do failed")
	data, err := json.MarshalIndent(cresp, "", "    ")
	runtimex.PanicOnError(err, "json.MarshalIndent failed")
	fmt.Printf("%s\n", string(data))
}
