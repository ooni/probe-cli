package main

//
// TCP proxy
//
// Reading this file gives you an understanding of how TCP is treated
// by the internal proxy services running on the userspace stack.
//

import (
	"context"
	"io"
	"math"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/apex/log"
)

// tcpProxyLoop is the main goroutine running a TCP proxy using userspace TCP.
//
// Arguments:
//
// - ctx is the context binding the lifetime of this goroutine;
//
// - wg is the wait group used by the parent;
//
// - tcpState is the TCP state for implementing DNAT;
//
// - listener is the TCP listener to use;
//
// - localPort is the local port as a string (for convenience).
//
// This goroutine runs until [ctx] is done or [listener] is closed.
func tcpProxyLoop(
	ctx context.Context,
	wg *sync.WaitGroup,
	tcpState *tcpState,
	listener net.Listener,
	localPort string,
) {
	// notify termination
	defer wg.Done()

	// arrange for propagating termination signal
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for ctx.Err() == nil {

		// accept new connection from the forward path
		uconn, err := listener.Accept()
		if err != nil {
			log.Warnf("tcpProxyLoop: Accept: %s", err.Error())
			return
		}

		// serve connection until [ctx] is done or [conn] starts failing
		go tcpProxyServe(ctx, tcpState, uconn, localPort)
	}
}

// tcpProxyServe implements tcpProxyLoop for a single [conn].
func tcpProxyServe(
	ctx context.Context,
	tcpState *tcpState,
	uconn net.Conn,
	localPort string,
) {
	// create scoped context to react to cancellation
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// shut down early when parent ctx is done
	defer func() {
		<-ctx.Done()
		uconn.Close()
	}()

	// obtain the port used by the client which is the same port used by
	// the userspace TCP client running inside, e.g., miniooni
	clientAddr := uconn.RemoteAddr().String()
	_, clientPortStr, err := net.SplitHostPort(clientAddr)
	if err != nil {
		log.Warnf("tcpProxyServe: net.SplitHostPort: %s", err.Error())
		return
	}
	clientPort, err := strconv.Atoi(clientPortStr)
	if err != nil {
		log.Warnf("tcpProxyServe: strconv.Atoi: %s", err.Error())
		return
	}
	if clientPort < 0 || clientPort >= math.MaxUint16 {
		log.Warn("tcpProxyServe: invalid port number")
		return
	}

	// obtain the real destination address using DNAT
	var destAddr net.IP
	tcpState.mu.Lock()
	destAddr = tcpState.dnat[uint16(clientPort)]
	tcpState.mu.Unlock()
	if destAddr == nil {
		log.Warnf("tcpProxyServe: missing DNAT entry for %d", clientPort)
		return
	}

	// compute the real remote endpoint
	endpoint := net.JoinHostPort(destAddr.String(), localPort)

	// dial the real connection
	dialer := &net.Dialer{
		Timeout: 15 * time.Second,
	}
	realConn, err := dialer.DialContext(ctx, "tcp", endpoint)
	if err != nil {
		log.Warnf("tcpProxyServer: dialer.DialContext: %s", err.Error())
		return
	}
	defer realConn.Close()

	// pipe the two connections
	wg := &sync.WaitGroup{}
	wg.Add(2)
	go tcpProxyReadWrite(wg, uconn, realConn)
	go tcpProxyReadWrite(wg, realConn, uconn)

	// wait for termination
	wg.Wait()
}

// tcpProxyReadWrite reads from left and writes to right
func tcpProxyReadWrite(wg *sync.WaitGroup, left, right net.Conn) {
	defer wg.Done()
	io.Copy(left, right)
}
