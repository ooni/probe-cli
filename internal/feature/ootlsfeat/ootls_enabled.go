//go:build go1.20

package ootlsfeat

import (
	"crypto/tls"
	"net"

	ootls "github.com/ooni/oocrypto/tls"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// NewClientConnStdlib returns a new client connection using the standard library's TLS stack.
func NewClientConnStdlib(conn net.Conn, config *tls.Config) (model.TLSConn, error) {
	return ootls.NewClientConnStdlib(conn, config)
}
