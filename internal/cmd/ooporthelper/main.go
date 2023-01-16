// command ooporthelper implements the Port Filtering test helper
package main

import (
	"context"
	"flag"
	"net"
	"sync"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/experiment/portfiltering"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

var (
	srvCtx      context.Context
	srvCancel   context.CancelFunc
	srvWg       = new(sync.WaitGroup)
	srvTestChan = make(chan string, len(TestPorts)) // buffered channel for testing
	srvTest     bool
)

func init() {
	srvCtx, srvCancel = context.WithCancel(context.Background())
}

func shutdown(ctx context.Context, l net.Listener) {
	<-ctx.Done()
	l.Close()
}

// TODO(DecFox): Add the ability of an echo service to generate some traffic
func handleConnection(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	<-ctx.Done()
}

func listenTCP(ctx context.Context, port string) {
	defer srvWg.Done()
	address := net.JoinHostPort("127.0.0.1", port)
	listener, err := net.Listen("tcp", address)
	runtimex.PanicOnError(err, "net.Listen failed")
	go shutdown(ctx, listener)
	srvTestChan <- port // send to channel to imply server will start listening on port
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Infof("listener unable to accept connections on port: %s", port)
			return
		}
		go handleConnection(ctx, conn)
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
	ports := portfiltering.Ports
	if srvTest {
		ports = TestPorts
	}
	for _, port := range ports {
		srvWg.Add(1)
		ctx, cancel := context.WithCancel(srvCtx)
		defer cancel()
		go listenTCP(ctx, port)
	}
	<-srvCtx.Done()
	srvWg.Wait() // wait for listeners on all ports to close
}
