package dialer

import (
	"net"
	"time"
)

// systemDialer is the underlying net.Dialer.
var systemDialer = &net.Dialer{
	Timeout:   15 * time.Second,
	KeepAlive: 15 * time.Second,
}
