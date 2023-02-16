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
	// apiEndpoint is the endpoint where we serve ooniprobe requests
	apiEndpoint = flag.String("endpoint", "127.0.0.1:8080", "API endpoint")

	// cpuprofile controls whether to write a cpuprofile on a file
	cpuprofile = flag.String("cpuprofile", "", "write cpu profile to file")

	// debug controls whether to enable verbose logging
	debug = flag.Bool("debug", false, "Toggle debug mode")

	// prometheusEpnt is the endpoint where we serve prometheus metrics
	prometheusEpnt = flag.String("prometheus", "127.0.0.1:9091", "Prometheus endpoint")

	// sigs is the channel where we collect signals
	sigs = make(chan os.Signal, 1)

	// srvAdd is used to pass the server address to tests
	srvAddr = make(chan string, 1)

	// srvWg is used by tests to know when the server has shut down
	srvWg = new(sync.WaitGroup)
)

// newResolver creates a new [model.Resolver] suitable for serving
// requests coming from ooniprobe clients.
func newResolver(logger model.Logger) model.Resolver {
	// Implementation note: pin to a specific resolver so we don't depend upon the
	// default resolver configured by the box. Also, use an encrypted transport thus
	// we're less vulnerable to any policy implemented by the box's provider.
	resolver := netxlite.NewParallelDNSOverHTTPSResolver(logger, "https://dns.google/dns-query")
	return resolver
}

// shutdown calls srv.Shutdown with a reasonably long timeout. The srv.Shutdown
// function will immediately close any open listener and then will wait until
// all pending connections are closed or the context has expired. By giving pending
// connections a long timeout to complete, we make sure we can serve many of them
// while still eventually shutting down the server. This function will decrement
// the given wait group counter when it is done running.
func shutdown(srv *http.Server, wg *sync.WaitGroup) {
	defer wg.Done()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
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

	// create a listening server for serving ooniprobe requests
	srv := &http.Server{Addr: *apiEndpoint, Handler: mux}
	listener, err := net.Listen("tcp", *apiEndpoint)
	runtimex.PanicOnError(err, "net.Listen failed")

	// await for the server's address to become available
	srvAddr <- listener.Addr().String()
	srvWg.Add(1)

	log.Infof("serving ooniprobe requests at http://%s/", listener.Addr().String())

	// start listening in the background
	go srv.Serve(listener)

	// create another server for serving prometheus metrics
	promMux := http.NewServeMux()
	promMux.Handle("/metrics", promhttp.Handler())
	promSrv := &http.Server{Addr: *prometheusEpnt, Handler: promMux}
	go promSrv.ListenAndServe()

	log.Infof("serving prometheus metrics at http://%s/", *prometheusEpnt)

	// await for the main context to be canceled or for a signal
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	sig := <-sigs
	log.Infof("interrupted by signal: %v", sig)

	// shutdown the servers awaiting for connections being
	// served to terminate before exiting gracefully.
	log.Infof("waiting for pending requests to complete")
	shutdownWg := &sync.WaitGroup{}
	shutdownWg.Add(1)
	go shutdown(srv, shutdownWg)
	shutdownWg.Add(1)
	go shutdown(promSrv, shutdownWg)
	shutdownWg.Wait()

	// notify tests that we are now done
	srvWg.Done()
}
