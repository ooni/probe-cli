package websteps

import (
	"crypto/tls"
	"net"
)

// TLSDo performs the TLS check.
func TLSDo(conn net.Conn, hostname string) (*tls.Conn, error) {
	tlsConn := tls.Client(conn, &tls.Config{
		ServerName: hostname,
		NextProtos: []string{"h2", "http/1.1"},
	})
	err := tlsConn.Handshake()
	return tlsConn, err
}
