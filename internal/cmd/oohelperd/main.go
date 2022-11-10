// Command oohelperd implements the Web Connectivity test helper.
package main

import (
	"context"
	"flag"
	"net"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/atomicx"
	"github.com/ooni/probe-cli/v3/internal/fscache"
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
	datadir := flag.String("datadir", "/var/lib/oohelperd", "Directory where to store data")
	prometheus := flag.String("prometheus", "127.0.0.1:9091", "Prometheus endpoint")
	debug := flag.Bool("debug", false, "Toggle debug mode")

	flag.Parse()
	log.SetLevel(logmap[*debug])
	defer srvCancel()

	dnscache := fscache.New(filepath.Join(*datadir, "cache", "dns"))
	endpointcache := fscache.New(filepath.Join(*datadir, "cache", "endpoint"))
	httpcache := fscache.New(filepath.Join(*datadir, "cache", "http"))
	go func() {
		ticker := time.NewTimer(15 * time.Minute)
		for {
			select {
			case <-srvCtx.Done():
				return
			case <-ticker.C:
				dnscache.Trim()
				endpointcache.Trim()
				httpcache.Trim()
			}
		}
	}()

	mux := http.NewServeMux()
	mux.Handle("/", &handler{
		BaseLogger:        log.Log,
		DNSCache:          dnscache,
		HTTPCache:         httpcache,
		Indexer:           &atomicx.Int64{},
		MaxAcceptableBody: maxAcceptableBody,
		NewClient: func(logger model.Logger) model.HTTPClient {
			// If the DoH resolver we're using insists that a given domain maps to
			// bogons, make sure we're going to fail the HTTP measurement.
			//
			// The TCP measurements scheduler in ipinfo.go will also refuse to
			// schedule TCP measurements for bogons.
			//
			// While this seems theoretical, as of 2022-08-28, I see:
			//
			//     % host polito.it
			//     polito.it has address 192.168.59.6
			//     polito.it has address 192.168.40.1
			//     polito.it mail is handled by 10 mx.polito.it.
			//
			// So, it's better to consider this as a possible corner case.
			reso := netxlite.MaybeWrapWithBogonResolver(
				true, // enabled
				newResolver(logger),
			)
			return netxlite.NewHTTPClientWithResolver(logger, reso)
		},
		NewDialer: func(logger model.Logger) model.Dialer {
			return netxlite.NewDialerWithResolver(logger, newResolver(logger))
		},
		NewResolver: newResolver,
		NewTLSHandshaker: func(logger model.Logger) model.TLSHandshaker {
			return netxlite.NewTLSHandshakerStdlib(logger)
		},
		TCPCache: endpointcache,
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
