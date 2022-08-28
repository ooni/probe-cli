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
	"github.com/ooni/probe-cli/v3/internal/atomicx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const maxAcceptableBody = 1 << 24

var (
	endpoint  = flag.String("endpoint", "127.0.0.1:8080", "API endpoint")
	srvAddr   = make(chan string, 1) // with buffer
	srvCancel context.CancelFunc
	srvCtx    context.Context
	srvWg     = new(sync.WaitGroup)
)

func init() {
	srvCtx, srvCancel = context.WithCancel(context.Background())
}

func newResolver(logger model.Logger) model.Resolver {
	// Implementation note: pin to a specific resolver so we don't depend upon the
	// default resolver configured by the box. Also, use an encrypted transport thus
	// we're less vulnerable to any policy implemented by the box's provider.
	resolver := netxlite.NewParallelDNSOverHTTPSResolver(logger, "https://dns.google/dns-query")
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
	prometheus := flag.String("prometheus", "127.0.0.1:9091", "Prometheus endpoint")
	debug := flag.Bool("debug", false, "Toggle debug mode")
	flag.Parse()
	log.SetLevel(logmap[*debug])
	defer srvCancel()
	mux := http.NewServeMux()
	mux.Handle("/", &handler{
		BaseLogger:        log.Log,
		Indexer:           &atomicx.Int64{},
		MaxAcceptableBody: maxAcceptableBody,
		NewClient: func(logger model.Logger) model.HTTPClient {
			return netxlite.NewHTTPClientWithResolver(logger, newResolver(logger))
		},
		NewDialer: func(logger model.Logger) model.Dialer {
			return netxlite.NewDialerWithResolver(logger, newResolver(logger))
		},
		NewResolver: newResolver,
		NewTLSHandshaker: func(logger model.Logger) model.TLSHandshaker {
			return netxlite.NewTLSHandshakerStdlib(logger)
		},
	})
	srv := &http.Server{Addr: *endpoint, Handler: mux}
	listener, err := net.Listen("tcp", *endpoint)
	runtimex.PanicOnError(err, "net.Listen failed")
	srvAddr <- listener.Addr().String()
	srvWg.Add(1)
	go srv.Serve(listener)
	promMux := http.NewServeMux()
	promMux.Handle("/metrics", promhttp.Handler())
	promSrv := &http.Server{Addr: *prometheus, Handler: promMux}
	go promSrv.ListenAndServe()
	<-srvCtx.Done()
	shutdown(srv)
	shutdown(promSrv)
	listener.Close()
	srvWg.Done()
}
