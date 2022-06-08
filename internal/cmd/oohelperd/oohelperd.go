// Command oohelperd contains the Web Connectivity test helper.
package main

import (
	"context"
	"flag"
	"net/http"
	"sync"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/cmd/oohelperd/internal/webconnectivity"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

const maxAcceptableBody = 1 << 24

var (
	dialer     model.Dialer
	endpoint   = flag.String("endpoint", ":8080", "Endpoint where to listen")
	httpClient model.HTTPClient
	resolver   model.Resolver
	srvcancel  context.CancelFunc
	srvctx     context.Context
	srvwg      = new(sync.WaitGroup)
)

func init() {
	srvctx, srvcancel = context.WithCancel(context.Background())
	// Implementation note: pin to a specific resolver so we don't depend upon the
	// default resolver configured by the box. Also, use an encrypted transport thus
	// we're less vulnerable to any policy implemented by the box's provider.
	resolver = netxlite.NewParallelDNSOverHTTPSResolver(log.Log, "https://8.8.8.8/dns-query")
	httpClient = netxlite.NewHTTPClientWithResolver(log.Log, resolver)
}

func shutdown(srv *http.Server) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	srv.Shutdown(ctx)
}

func main() {
	logmap := map[bool]log.Level{
		true:  log.DebugLevel,
		false: log.InfoLevel,
	}
	debug := flag.Bool("debug", false, "Toggle debug mode")
	flag.Parse()
	log.SetLevel(logmap[*debug])
	testableMain()
}

func testableMain() {
	mux := http.NewServeMux()
	mux.Handle("/", webconnectivity.Handler{
		Client:            httpClient,
		Dialer:            dialer,
		MaxAcceptableBody: maxAcceptableBody,
		Resolver:          resolver,
	})
	srv := &http.Server{Addr: *endpoint, Handler: mux}
	srvwg.Add(1)
	go srv.ListenAndServe()
	<-srvctx.Done()
	shutdown(srv)
	srvwg.Done()
}
