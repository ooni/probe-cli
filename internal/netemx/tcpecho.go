package netemx

import (
	"io"
	"net"
	"sync"

	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// TCPEchoNetStack is a [QAEnvNetStackHandler] implementing a TCP echo service.
func TCPEchoNetStack(logger model.Logger, ports ...uint16) QAEnvNetStackHandler {
	return &tcpEchoNetStack{
		closers: []io.Closer{},
		logger:  logger,
		mu:      sync.Mutex{},
		ports:   ports,
	}
}

type tcpEchoNetStack struct {
	closers []io.Closer
	logger  model.Logger
	mu      sync.Mutex
	ports   []uint16
}

// Close implements QAEnvNetStackHandler.
func (echo *tcpEchoNetStack) Close() error {
	// "this method MUST be CONCURRENCY SAFE"
	defer echo.mu.Unlock()
	echo.mu.Lock()

	// make sure we close all the child listeners
	for _, closer := range echo.closers {
		_ = closer.Close()
	}

	// "this method MUST be IDEMPOTENT"
	echo.closers = []io.Closer{}

	return nil
}

// Listen implements QAEnvNetStackHandler.
func (echo *tcpEchoNetStack) Listen(stack *netem.UNetStack) error {
	// "this method MUST be CONCURRENCY SAFE"
	defer echo.mu.Unlock()
	echo.mu.Lock()

	// for each port of interest - note that here we panic liberally because we are
	// allowed to do so by the [QAEnvNetStackHandler] documentation.
	for _, port := range echo.ports {
		// create the endpoint address
		ipAddr := net.ParseIP(stack.IPAddress())
		runtimex.Assert(ipAddr != nil, "invalid IP address")
		epnt := &net.TCPAddr{IP: ipAddr, Port: int(port)}

		// attempt to listen
		listener := runtimex.Try1(stack.ListenTCP("tcp", epnt))

		// spawn goroutine for accepting
		go echo.acceptLoop(listener)

		// track this listener as something to close later
		echo.closers = append(echo.closers, listener)
	}
	return nil
}

func (echo *tcpEchoNetStack) acceptLoop(listener net.Listener) {
	// Implementation note: because this function is only used for writing QA tests, it is
	// fine that we are using runtimex.Try1 and ignoring any panic.
	defer runtimex.CatchLogAndIgnorePanic(echo.logger, "qaEnvNetStackTCPEcho.acceptLoop")
	for {
		conn := runtimex.Try1(listener.Accept())
		go echo.serve(conn)
	}
}

func (echo *tcpEchoNetStack) serve(conn net.Conn) {
	// Implementation note: because this function is only used for writing QA tests, it is
	// fine that we are using runtimex.Try1 and ignoring any panic.
	defer runtimex.CatchLogAndIgnorePanic(echo.logger, "qaEnvTCPListenerEcho.serve")

	// make sure we close the conn
	defer conn.Close()

	// loop until there is an I/O error
	for {
		buffer := make([]byte, 4096)
		count := runtimex.Try1(conn.Read(buffer))
		_, _ = conn.Write(buffer[:count])
	}
}
