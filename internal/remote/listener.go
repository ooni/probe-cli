package remote

import "net"

// ListenerFactory is a factory for creating a listener.
type ListenerFactory interface {
	Listen() (net.Listener, error)
}
