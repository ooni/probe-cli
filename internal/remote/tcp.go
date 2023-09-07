package remote

import (
	"context"
	"net"
)

// TCPListenerFactory is a factory creating a [net.Listener] encapsulating
// IP packets directly into the payload of created connections.
type TCPListenerFactory struct {
	// Endpoint is the endpoint where to listen.
	Endpoint string
}

var _ ListenerFactory = &TCPListenerFactory{}

// Listen implements ListenerFactory.
func (tlf *TCPListenerFactory) Listen() (net.Listener, error) {
	return net.Listen("tcp", tlf.Endpoint)
}

// DialTCP dials a TCP connection with the remote endpoint.
func DialTCP(ctx context.Context, epnt string) (net.Conn, error) {
	return (&net.Dialer{}).DialContext(ctx, "tcp", epnt)
}
