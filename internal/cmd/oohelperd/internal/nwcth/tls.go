package nwcth

import (
	"crypto/tls"
	"net"
)

func TLSDo(conn net.Conn, hostname string) (*tls.Conn, *TLSHandshakeMeasurement) {
	tlsConn := tls.Client(conn, &tls.Config{
		ServerName: hostname,
		NextProtos: []string{"h2"},
	})
	err := tlsConn.Handshake()
	return nil, &TLSHandshakeMeasurement{
		Failure: newfailure(err),
	}
}
