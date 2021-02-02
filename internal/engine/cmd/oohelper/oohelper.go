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
	"github.com/ooni/probe-cli/v3/internal/engine/cmd/oohelper/internal"
	"github.com/ooni/probe-cli/v3/internal/engine/runtimex"
	"github.com/ooni/probe-cli/v3/internal/engine/netx"
)

var (
	ctx, cancel = context.WithCancel(context.Background())
	debug       = flag.Bool("debug", false, "Toggle debug mode")
	httpClient  *http.Client
	resolver    netx.Resolver
	server      = flag.String("server", "https://wcth.ooni.io/", "URL of the test helper")
	target      = flag.String("target", "", "Target URL for the test helper")
)

func init() {
	txp := netx.NewHTTPTransport(netx.Config{Logger: log.Log})
	httpClient = &http.Client{Transport: txp}
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
