package mocks

import "net"

// Listener allows mocking a net.Listener.
type Listener struct {
	// Accept allows mocking Accept.
	MockAccept func() (net.Conn, error)

	// Close allows mocking Close.
	MockClose func() error

	// Addr allows mocking Addr.
	MockAddr func() net.Addr
}

var _ net.Listener = &Listener{}

// Accept implements net.Listener.Accept
func (li *Listener) Accept() (net.Conn, error) {
	return li.MockAccept()
}

// Close implements net.Listener.Closer.
func (li *Listener) Close() error {
	return li.MockClose()
}

// Addr implements net.Listener.Addr
func (li *Listener) Addr() net.Addr {
	return li.MockAddr()
}
