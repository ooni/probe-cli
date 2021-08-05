// Command oohelperd contains the Web Connectivity test helper.
package main

import (
	"context"
	"flag"
	"net/http"
	"sync"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/cmd/oohelperd/internal"
	"github.com/ooni/probe-cli/v3/internal/cmd/oohelperd/internal/nwcth"
	"github.com/ooni/probe-cli/v3/internal/engine/netx"
)

const maxAcceptableBody = 1 << 24

var (
	dialer     netx.Dialer
	endpoint   = flag.String("endpoint", ":8080", "Endpoint where to listen")
	httpx      *http.Client
	httpnx     *http.Client
	quicdialer netx.QUICDialer
	resolver   netx.Resolver
	srvcancel  context.CancelFunc
	srvctx     context.Context
	srvwg      = new(sync.WaitGroup)
)

func init() {
	srvctx, srvcancel = context.WithCancel(context.Background())
	dialer = netx.NewDialer(netx.Config{Logger: log.Log})
	txp := netx.NewHTTPTransport(netx.Config{Logger: log.Log})
	httpx = &http.Client{Transport: txp}
	quicdialer = netx.NewQUICDialer(netx.Config{Logger: log.Log})
	resolver = netx.NewResolver(netx.Config{Logger: log.Log})
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
	mux.Handle("/api/unstable/nwcth", nwcth.NWCTHHandler{
		Client:            &http.Client{},
		Dialer:            dialer,
		MaxAcceptableBody: maxAcceptableBody,
		QuicDialer:        quicdialer,
		Resolver:          resolver,
	})
	mux.Handle("/", internal.Handler{
		Client:            httpx,
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
