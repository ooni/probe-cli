// command ooporthelper implements the Port Filtering test helper
package main

import (
	"context"
	"flag"
	"net"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

var (
	srvCtx    context.Context
	srvCancel context.CancelFunc
)

func init() {
	srvCtx, srvCancel = context.WithCancel(context.Background())
}

func tcpHandler(conn net.Conn) {
	defer conn.Close()
}

func listenTCP(port string) {
	listener, err := net.Listen("tcp", port)
	if err != nil {
		runtimex.PanicOnError(err, "net.Listen failed")
	}
	defer listener.Close()
	for {
		conn, err := listener.Accept()
		if err != nil {
			runtimex.PanicOnError(err, "listener.Accept failed")
		}
		go tcpHandler(conn)
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
		go listenTCP(port)
	}
	<-srvCtx.Done()
}
