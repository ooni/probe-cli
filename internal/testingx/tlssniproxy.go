package testingx

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/logx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// TLSSNIProxyNetx is how [TLSSNIProxy] views [*netxlite.Netx].
type TLSSNIProxyNetx interface {
	ListenTCP(network string, addr *net.TCPAddr) (net.Listener, error)
	NewDialerWithResolver(dl model.DebugLogger, r model.Resolver, w ...model.DialerWrapper) model.Dialer
	NewStdlibResolver(logger model.DebugLogger) model.Resolver
}

// TLSSNIProxy is a proxy using the SNI to figure out where to connect to.
type TLSSNIProxy struct {
	// closeOnce provides "once" semantics for Close.
	closeOnce sync.Once

	// listener is the TCP listener we're using.
	listener net.Listener

	// logger is the logger we should use.
	logger model.Logger

	// netx is the underlying network.
	netx TLSSNIProxyNetx

	// wg is the wait group for the background listener
	wg *sync.WaitGroup
}

// MustNewTLSSNIProxyEx creates a new [*TLSSNIProxy].
func MustNewTLSSNIProxyEx(
	logger model.Logger, netx TLSSNIProxyNetx, tcpAddr *net.TCPAddr) *TLSSNIProxy {
	listener := runtimex.Try1(netx.ListenTCP("tcp", tcpAddr))
	proxy := &TLSSNIProxy{
		closeOnce: sync.Once{},
		listener:  listener,
		logger: &logx.PrefixLogger{
			Prefix: fmt.Sprintf("%-16s", "TLSPROXY"),
			Logger: logger,
		},
		netx: netx,
		wg:   &sync.WaitGroup{},
	}
	proxy.wg.Add(1)
	go proxy.mainloop()
	return proxy
}

// Close implements io.Closer
func (tp *TLSSNIProxy) Close() (err error) {
	tp.closeOnce.Do(func() {
		err = tp.listener.Close()
		tp.wg.Wait()
	})
	return
}

// Endpoint returns the listening endpoint or nil after Close has been called.
func (tp *TLSSNIProxy) Endpoint() string {
	return tp.listener.Addr().String()
}

func (tp *TLSSNIProxy) mainloop() {
	// make sure panics don't crash the process
	defer runtimex.CatchLogAndIgnorePanic(tp.logger, "TLSSNIProxy.mainloop")

	defer tp.wg.Done()
	for {
		conn, err := tp.listener.Accept()
		if errors.Is(err, net.ErrClosed) {
			return
		}

		// use panics to reduce the testing surface, which is ~okay given
		// that this code is meant to support testing
		runtimex.PanicOnError(err, "tp.listener.Accept() failed")

		// we're creating a goroutine per connection, which is ~okay because
		// this code is designed for helping with testing
		go tp.handle(conn)
	}
}

func (tp *TLSSNIProxy) handle(clientConn net.Conn) {
	// make sure panics don't crash the process
	defer runtimex.CatchLogAndIgnorePanic(tp.logger, "TLSSNIProxy.handle")

	// make sure we close the client connection
	defer clientConn.Close()

	// read initial records
	buffer := make([]byte, 1<<17)
	count := runtimex.Try1(clientConn.Read(buffer))
	rawRecords := buffer[:count]

	// inspecty the raw records to find the SNI
	sni := runtimex.Try1(netem.ExtractTLServerName(rawRecords))

	// connect to the remote host
	tcpDialer := tp.netx.NewDialerWithResolver(tp.logger, tp.netx.NewStdlibResolver(tp.logger))
	serverConn := runtimex.Try1(tcpDialer.DialContext(context.Background(), "tcp", net.JoinHostPort(sni, "443")))
	defer serverConn.Close()

	// forward the initial records to the server
	_ = runtimex.Try1(serverConn.Write(rawRecords))

	// route traffic between the conns
	wg := &sync.WaitGroup{}
	wg.Add(2)
	go tp.forward(wg, clientConn, serverConn)
	go tp.forward(wg, serverConn, clientConn)
	wg.Wait()
}

func (tp *TLSSNIProxy) forward(wg *sync.WaitGroup, left, right net.Conn) {
	defer wg.Done()
	_, _ = io.Copy(right, left)
}
