package testingx

import "net"

// TCPListener creates TCP connections for HTTP, TLS, etc. This type should work both
// with the standard library and with netem as its backend.
type TCPListener interface {
	ListenTCP(network string, addr *net.TCPAddr) (net.Listener, error)
}

// TCPListenerStdlib implements [HTTPListener] for the stdlib.
type TCPListenerStdlib struct{}

var _ TCPListener = &TCPListenerStdlib{}

// ListenTCP implements HTTPListener.
func (*TCPListenerStdlib) ListenTCP(network string, addr *net.TCPAddr) (net.Listener, error) {
	return net.ListenTCP(network, addr)
}

// tcpMaybeResetNetConn is a portable mechanism to reset a net.Conn that takes into account
// both TLS wrapping with any library and stdlib vs. netem concerns.
//
// Bug: netem is not WAI because there's no *gonet.TCPConn.SetLinger method.
func tcpMaybeResetNetConn(conn net.Conn) {
	// first, let's try to get the underlying conn, when we're using TLS
	type connUnwrapper interface {
		NetConn() net.Conn
	}
	if unwrapper, good := conn.(connUnwrapper); good {
		conn = unwrapper.NetConn()
	}

	// then, let's try to get the controller for disabling linger
	type connLingerSetter interface {
		SetLinger(sec int) error
	}
	if setter, good := conn.(connLingerSetter); good {
		setter.SetLinger(0)
	}

	// close the conn to trigger the reset (we MUST call Close here where
	// we're using the underlying conn and it doesn't suffice to call it
	// inside the http.Handler, where wrapping would not cause a RST)
	conn.Close()
}
