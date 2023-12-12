//go:build go1.21 || ooni_feature_disable_oohttp

package ootlsfeat

import (
	"crypto/tls"
	"net"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// NewClientConnStdlib returns a new client connection using the standard library's TLS stack.
func NewClientConnStdlib(conn net.Conn, config *tls.Config) (model.TLSConn, error) {
	return tls.Client(conn, config), nil
}
