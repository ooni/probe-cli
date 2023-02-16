// Command oohelperd implements the Web Connectivity test helper.
package main

import (
	"context"
	"flag"
	"net"
	"net/http"
	"os"
	"os/signal"
	"runtime/pprof"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/apex/log"
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

// newHandler constructs the [handler] used by [main].
func newHandler() *handler {
	return &handler{
		BaseLogger:        log.Log,
		Indexer:           &atomic.Int64{},
		MaxAcceptableBody: maxAcceptableBody,
		Measure:           measure,
		NewHTTPClient: func(logger model.Logger) model.HTTPClient {
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
		NewHTTP3Client: func(logger model.Logger) model.HTTPClient {
			reso := netxlite.MaybeWrapWithBogonResolver(
				true, // enabled
				newResolver(logger),
			)
			return netxlite.NewHTTP3ClientWithResolver(logger, reso)
		},
		NewDialer: func(logger model.Logger) model.Dialer {
			return netxlite.NewDialerWithoutResolver(logger)
		},
		NewQUICDialer: func(logger model.Logger) model.QUICDialer {
			return netxlite.NewQUICDialerWithoutResolver(
				netxlite.NewQUICListener(),
				logger,
			)
		},
		NewResolver: newResolver,
		NewTLSHandshaker: func(logger model.Logger) model.TLSHandshaker {
			return netxlite.NewTLSHandshakerStdlib(logger)
		},
	}
}

func main() {
	// initialize variables for command line options
	prometheus := flag.String("prometheus", "127.0.0.1:9091", "Prometheus endpoint")
	debug := flag.Bool("debug", false, "Toggle debug mode")
	cpuprofile := flag.String("cpuprofile", "", "write cpu profile to file")

	// parse command line options
	flag.Parse()

	// optionally collect a CPU profile
	if *cpuprofile != "" {
		fp, err := os.Create(*cpuprofile)
		runtimex.PanicOnError(err, "os.Create failed")
		pprof.StartCPUProfile(fp)
		defer func() {
			pprof.StopCPUProfile()
			log.Infof("written cpuprofile at: %s", *cpuprofile)
			log.Infof("to analyze the profile run: go tool pprof oohelperd %s", *cpuprofile)
			log.Infof("use the web command to get an interactive web profile")
		}()
	}

	// set log level
	logmap := map[bool]log.Level{
		true:  log.DebugLevel,
		false: log.InfoLevel,
	}
	log.SetLevel(logmap[*debug])

	// create the HTTP server mux
	mux := http.NewServeMux()

	// add the main oohelperd handler to the mux
	mux.Handle("/", newHandler())

	// create a listening server for oohelperd
	srv := &http.Server{Addr: *endpoint, Handler: mux}
	listener, err := net.Listen("tcp", *endpoint)
	runtimex.PanicOnError(err, "net.Listen failed")

	// await for the server's address to become available
	srvAddr <- listener.Addr().String()
	srvWg.Add(1)

	// start listening in the background
	defer srvCancel()
	go srv.Serve(listener)

	// create another server for serving prometheus metrics
	promMux := http.NewServeMux()
	promMux.Handle("/metrics", promhttp.Handler())
	promSrv := &http.Server{Addr: *prometheus, Handler: promMux}
	go promSrv.ListenAndServe()

	// await for the main context to be canceled or for a signal
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-srvCtx.Done():
	case sig := <-sigs:
		log.Infof("interrupted by signal: %v", sig)
	}

	// shutdown the servers
	shutdown(srv)
	shutdown(promSrv)

	// close the listener
	listener.Close()

	// notify tests that we are now done
	srvWg.Done()
}
