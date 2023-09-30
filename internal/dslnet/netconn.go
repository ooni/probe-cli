package dslnet

import (
	"net"
)

// NetConn is an established network connection.
type NetConn struct {
	// Conn is the MANDATORY conn.
	net.Conn
}
