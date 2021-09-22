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
	"github.com/ooni/probe-cli/v3/internal/measurex"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

var (
	ctx, cancel = context.WithCancel(context.Background())
	debug       = flag.Bool("debug", false, "Toggle debug mode")
	httpClient  *http.Client
	resolver    netx.Resolver
	server      = flag.String("server", "", "URL of the test helper")
	target      = flag.String("target", "", "Target URL for the test helper")
	fwebsteps   = flag.Bool("websteps", false, "Use the websteps TH")
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
	defer cancel()
	logmap := map[bool]log.Level{
		true:  log.DebugLevel,
		false: log.InfoLevel,
	}
	flag.Parse()
	log.SetLevel(logmap[*debug])
	apimap := map[bool]func() interface{}{
		false: wcth,
		true:  webstepsth,
	}
	cresp := apimap[*fwebsteps]()
	data, err := json.MarshalIndent(cresp, "", "    ")
	runtimex.PanicOnError(err, "json.MarshalIndent failed")
	fmt.Printf("%s\n", string(data))
}

func webstepsth() interface{} {
	serverURL := *server
	if serverURL == "" {
		serverURL = "http://127.0.0.1:8080/api/v1/websteps"
	}
	clnt := &measurex.THClient{
		DNServers:  []string{"8.8.8.8:53", "8.8.4.4:53", "1.1.1.1:53", "1.0.0.1:53"},
		HTTPClient: httpClient,
		ServerURL:  serverURL,
	}
	cresp, err := clnt.Run(ctx, *target)
	runtimex.PanicOnError(err, "client.Run failed")
	return cresp
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
