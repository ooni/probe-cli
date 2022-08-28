// command ooporthelper implements the Port Filtering test helper
package main

import (
	"context"
	"flag"
	"net"
	"sync"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

var (
	srvCtx    context.Context
	srvCancel context.CancelFunc
	srvWg     = new(sync.WaitGroup)
)

func init() {
	srvCtx, srvCancel = context.WithCancel(context.Background())
}

func shutdown(ctx context.Context, l net.Listener) {
	<-ctx.Done()
	l.Close()
}

// TODO(DecFox): Add the ability of an echo service to generate some traffic
func handleConnetion(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	<-ctx.Done()
}

func listenTCP(ctx context.Context, port string) {
	defer srvWg.Done()
	listener, err := net.Listen("tcp", port)
	if err != nil {
		runtimex.PanicOnError(err, "net.Listen failed")
	}
	go shutdown(ctx, listener)
	for {
		conn, err := listener.Accept()
		if err != nil {
			runtimex.PanicOnError(err, "listener.Accept failed")
		}
		go handleConnetion(ctx, conn)
	}
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
	for _, port := range Ports {
		srvWg.Add(1)
		ctx, cancel := context.WithCancel(srvCtx)
		defer cancel()
		go listenTCP(ctx, port)
	}
	<-srvCtx.Done()
	srvWg.Wait()
}
