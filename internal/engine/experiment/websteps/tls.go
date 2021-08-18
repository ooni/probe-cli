package websteps

import (
	"context"
	"crypto/tls"
	"net"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
	utls "gitlab.com/yawning/utls.git"
)

// TLSDo performs the TLS check.
func TLSDo(ctx context.Context, conn net.Conn, hostname string) (net.Conn, error) {
	tlsConf := &tls.Config{
		ServerName: hostname,
		NextProtos: []string{"h2", "http/1.1"},
	}
	h := &netxlite.TLSHandshakerConfigurable{
		NewConn: netxlite.NewConnUTLS(&utls.HelloChrome_Auto),
	}
	tlsConn, _, err := h.Handshake(ctx, conn, tlsConf)
	return tlsConn, err
}
