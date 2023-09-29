package pnet

import (
	"io"
	"net"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// NetConn is an established network connection.
type NetConn struct {
	// Conn is the MANDATORY conn.
	Conn net.Conn

	// Logger is the MANDATORY logger to use.
	Logger model.Logger
}

var _ io.Closer = NetConn{}

// Close implements io.Closer.
func (nc NetConn) Close() error {
	return nc.Conn.Close()
}
