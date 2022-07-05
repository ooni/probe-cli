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
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/webstepsx"
	"github.com/ooni/probe-cli/v3/internal/engine/netx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// TODO(bassosimone): we should refactor this package to be flat w/o the internal
// package. We're now past the many-small-packages time.

const maxAcceptableBody = 1 << 24

var (
	endpoint  = flag.String("endpoint", ":8080", "Endpoint where to listen")
	srvcancel context.CancelFunc
	srvctx    context.Context
	srvwg     = new(sync.WaitGroup)
)

func init() {
	srvctx, srvcancel = context.WithCancel(context.Background())
}

func newresolver() model.Resolver {
	// fix: use 8.8.8.8:53/udp so we pin to a specific resolver.
	resolver, err := netx.NewDNSClient(netx.Config{Logger: log.Log}, "udp://8.8.8.8:53")
	runtimex.PanicOnError(err, "NewDNSClient failed")
	return resolver
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
	mux.Handle("/api/v1/websteps", &webstepsx.THHandler{})
	mux.Handle("/", webconnectivity.Handler{
		MaxAcceptableBody: maxAcceptableBody,
		NewClient: func() model.HTTPClient {
			txp := netx.NewHTTPTransport(netx.Config{
				FullResolver: newresolver(),
				Logger:       log.Log,
			})
			return &http.Client{Transport: txp}
		},
		NewDialer: func() model.Dialer {
			return netx.NewDialer(netx.Config{
				FullResolver: newresolver(),
				Logger:       log.Log,
			})
		},
		NewResolver: newresolver,
	})
	srv := &http.Server{Addr: *endpoint, Handler: mux}
	srvwg.Add(1)
	go srv.ListenAndServe()
	<-srvctx.Done()
	shutdown(srv)
	srvwg.Done()
}
