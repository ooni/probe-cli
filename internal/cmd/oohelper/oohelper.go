// Command oohelper contains a simple command line
// client for the Web Connectivity test helper.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/cmd/oohelper/internal"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

var (
	ctx, cancel = context.WithCancel(context.Background())
	debug       = flag.Bool("debug", false, "Toggle debug mode")
	httpClient  model.HTTPClient
	resolver    model.Resolver
	server      = flag.String("server", "", "URL of the test helper")
	target      = flag.String("target", "", "Target URL for the test helper")
)

func init() {
	// Use a nonstandard resolver, which is enough to work around the
	// puzzling https://github.com/ooni/probe/issues/1409 issue.
	const resolverURL = "https://8.8.8.8/dns-query"
	resolver = netxlite.NewParallelDNSOverHTTPSResolver(log.Log, resolverURL)
	txp := netxlite.NewHTTPTransportWithResolver(log.Log, resolver)
	httpClient = netxlite.NewHTTPClient(txp)
}

func main() {
	defer cancel()
	logmap := map[bool]log.Level{
		true:  log.DebugLevel,
		false: log.InfoLevel,
	}
	flag.Parse()
	log.SetLevel(logmap[*debug])
	cresp := wcth()
	data, err := json.MarshalIndent(cresp, "", "    ")
	runtimex.PanicOnError(err, "json.MarshalIndent failed")
	fmt.Printf("%s\n", string(data))
}

func wcth() interface{} {
	serverURL := *server
	if serverURL == "" {
		serverURL = "https://wcth.ooni.io/"
	}
	clnt := internal.OOClient{HTTPClient: httpClient, Resolver: resolver}
	config := internal.OOConfig{TargetURL: *target, ServerURL: serverURL}
	cresp, err := clnt.Do(ctx, config)
	runtimex.PanicOnError(err, "client.Do failed")
	return cresp
}
