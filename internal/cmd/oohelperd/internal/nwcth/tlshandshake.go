package nwcth

import (
	"crypto/tls"
	"net"
)

func TLSDo(conn net.Conn, hostname string) (*tls.Conn, *TLSHandshakeMeasurement) {
	tlsConn := tls.Client(conn, &tls.Config{
		ServerName: hostname,
	})
	err := tlsConn.Handshake()
	if err != nil {
		s := err.Error()
		return nil, &TLSHandshakeMeasurement{
			Failure: &s,
		}
	}
	return tlsConn, &TLSHandshakeMeasurement{}
}
