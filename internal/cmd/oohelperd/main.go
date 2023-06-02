// Command oohelperd implements the Web Connectivity test helper.
package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/http/cookiejar"
	"net/http/pprof"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/version"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/net/publicsuffix"
)

// maxAcceptableBodySize is the maximum acceptable body size for incoming
// API requests as well as when we're measuring webpages.
const maxAcceptableBodySize = 1 << 24

var (
	// apiEndpoint is the endpoint where we serve ooniprobe requests
	apiEndpoint = flag.String("api-endpoint", "127.0.0.1:8080", "API endpoint")

	// debug controls whether to enable verbose logging
	debug = flag.Bool("debug", false, "Toggle debug mode")

	// pprofEndpoint is the endpoint where we serve pprof info.
	pprofEndpoint = flag.String("pprof-endpoint", "127.0.0.1:6061", "Pprof endpoint")

	// prometheusEndpoint is the endpoint where we serve prometheus metrics
	prometheusEndpoint = flag.String("prometheus-endpoint", "127.0.0.1:9091", "Prometheus endpoint")

	// replace runs the commands to replace a running oohelperd.
	replace = flag.Bool("replace", false, "Replaces a running oohelperd instance")

	// sigs is the channel where we collect signals
	sigs = make(chan os.Signal, 1)

	// srvAdd is used to pass the server address to tests
	srvAddr = make(chan string, 1)

	// srvWg is used by tests to know when the server has shut down
	srvWg = new(sync.WaitGroup)

	// versionFlag indicates we must print the version on stdout
	versionFlag = flag.Bool("version", false, "Prints version information on the stdout")
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

// newCookieJar is the factory for constructing a new cookier jar.
func newCookieJar() *cookiejar.Jar {
	// Implementation note: the [cookiejar.New] function always returns a
	// nil error; hence, it's safe here to use [runtimex.Try1].
	return runtimex.Try1(cookiejar.New(&cookiejar.Options{
		PublicSuffixList: publicsuffix.List,
	}))
}

// newHTTPClientWithTransportFactory creates a new HTTP client.
func newHTTPClientWithTransportFactory(
	logger model.Logger,
	txpFactory func(model.DebugLogger, model.Resolver) model.HTTPTransport,
) model.HTTPClient {
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

	// fix: We MUST set a cookie jar for measuring HTTP. See
	// https://github.com/ooni/probe/issues/2488 for additional
	// context and pointers to the relevant measurements.
	client := &http.Client{
		Transport:     txpFactory(logger, reso),
		CheckRedirect: nil,
		Jar:           newCookieJar(),
		Timeout:       0,
	}

	return netxlite.WrapHTTPClient(client)
}

// newHandler constructs the [handler] used by [main].
func newHandler() *handler {
	return &handler{
		BaseLogger:        log.Log,
		Indexer:           &atomic.Int64{},
		MaxAcceptableBody: maxAcceptableBodySize,
		Measure:           measure,

		NewHTTPClient: func(logger model.Logger) model.HTTPClient {
			return newHTTPClientWithTransportFactory(
				logger,
				netxlite.NewHTTPTransportWithResolver,
			)
		},

		NewHTTP3Client: func(logger model.Logger) model.HTTPClient {
			return newHTTPClientWithTransportFactory(
				logger,
				netxlite.NewHTTP3TransportWithResolver,
			)
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

	// set log level
	logmap := map[bool]log.Level{
		true:  log.DebugLevel,
		false: log.InfoLevel,
	}
	log.SetLevel(logmap[*debug])

	if *replace {
		replaceRunningInstance(newReplaceDeps())
		return
	}
	if *versionFlag {
		fmt.Printf("oohelperd/%s %s dirty=%v commit=%s\n",
			version.Version,
			runtimex.BuildInfo.GoVersion,
			runtimex.BuildInfo.VcsModified,
			runtimex.BuildInfo.VcsRevision,
		)
		return
	}

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

	// start listening in the background
	go srv.Serve(listener)
	log.Infof("serving ooniprobe requests at http://%s/", listener.Addr().String())

	// create another server for serving prometheus metrics
	promMux := http.NewServeMux()
	promMux.Handle("/metrics", promhttp.Handler())
	promSrv := &http.Server{Addr: *prometheusEndpoint, Handler: promMux}
	go promSrv.ListenAndServe()
	log.Infof("serving prometheus metrics at http://%s/", *prometheusEndpoint)

	// create another server for serving pprof metrics
	pprofMux := http.NewServeMux()
	pprofMux.Handle("/debug/pprof/profile", http.HandlerFunc(pprof.Profile))
	pprofMux.Handle("/debug/pprof/trace", http.HandlerFunc(pprof.Trace))
	pprofSrv := &http.Server{Addr: *pprofEndpoint, Handler: pprofMux}
	go pprofSrv.ListenAndServe()
	log.Infof("serving CPU profile at http://%s/debug/pprof/profile", *pprofEndpoint)
	log.Infof("serving execution traces at http://%s/debug/pprof/trace", *pprofEndpoint)

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
	shutdownWg.Add(1)
	go shutdown(pprofSrv, shutdownWg)
	shutdownWg.Wait()

	// notify tests that we are now done
	srvWg.Done()
}
