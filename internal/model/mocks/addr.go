package mocks

import "net"

// Addr allows mocking net.Addr.
type Addr struct {
	MockString  func() string
	MockNetwork func() string
}

var _ net.Addr = &Addr{}

// String calls MockString.
func (a *Addr) String() string {
	return a.MockString()
}

// Network calls MockNetwork.
func (a *Addr) Network() string {
	return a.MockNetwork()
}
