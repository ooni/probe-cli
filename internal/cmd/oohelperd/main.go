// Command oohelperd implements the Web Connectivity test helper.
package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/oohelperd"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/version"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	// apiEndpoint is the endpoint where we serve ooniprobe requests
	apiEndpoint = flag.String("api-endpoint", "127.0.0.1:8080", "API endpoint")

	// debug controls whether to enable verbose logging
	debug = flag.Bool("debug", false, "Toggle debug mode")

	// pprofEndpoint is the endpoint where we serve pprof info.
	pprofEndpoint = flag.String("pprof-endpoint", "127.0.0.1:6061", "Pprof endpoint")

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

	prometheusMetricsPassword = os.Getenv("PROMETHEUS_METRICS_PASSWORD")
)

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
	mux.Handle("/", oohelperd.NewHandler(log.Log, &netxlite.Netx{}))
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, req *http.Request) {
		user, pass, ok := req.BasicAuth()
		if ok && user == "prom" && pass == prometheusMetricsPassword {
			promhttp.Handler().ServeHTTP(w, req)
		} else {
			w.Header().Set("WWW-Authenticate", "Basic realm=metrics")
			w.WriteHeader(401)
			w.Write([]byte("401 Unauthorized\n"))
		}
	})

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
	go shutdown(pprofSrv, shutdownWg)
	shutdownWg.Wait()

	// notify tests that we are now done
	srvWg.Done()
}
