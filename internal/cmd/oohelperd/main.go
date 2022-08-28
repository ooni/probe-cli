// Command oohelperd implements the Web Connectivity test helper.
package main

import (
	"context"
	"flag"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

const maxAcceptableBody = 1 << 24

var (
	endpoint  = flag.String("endpoint", "127.0.0.1:8080", "Endpoint where to listen")
	srvAddr   = make(chan string, 1) // with buffer
	srvCancel context.CancelFunc
	srvCtx    context.Context
	srvWg     = new(sync.WaitGroup)
)

func init() {
	srvCtx, srvCancel = context.WithCancel(context.Background())
}

func newResolver() model.Resolver {
	// Implementation note: pin to a specific resolver so we don't depend upon the
	// default resolver configured by the box. Also, use an encrypted transport thus
	// we're less vulnerable to any policy implemented by the box's provider.
	resolver := netxlite.NewParallelDNSOverHTTPSResolver(log.Log, "https://8.8.8.8/dns-query")
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
	defer srvCancel()
	mux := http.NewServeMux()
	mux.Handle("/", &handler{
		MaxAcceptableBody: maxAcceptableBody,
		NewClient: func() model.HTTPClient {
			return netxlite.NewHTTPClientWithResolver(log.Log, newResolver())
		},
		NewDialer: func() model.Dialer {
			return netxlite.NewDialerWithResolver(log.Log, newResolver())
		},
		NewResolver: newResolver,
		NewTLSHandshaker: func() model.TLSHandshaker {
			return netxlite.NewTLSHandshakerStdlib(log.Log)
		},
	})
	srv := &http.Server{Addr: *endpoint, Handler: mux}
	listener, err := net.Listen("tcp", *endpoint)
	runtimex.PanicOnError(err, "net.Listen failed")
	srvAddr <- listener.Addr().String()
	srvWg.Add(1)
	go srv.Serve(listener)
	<-srvCtx.Done()
	shutdown(srv)
	listener.Close()
	srvWg.Done()
}
