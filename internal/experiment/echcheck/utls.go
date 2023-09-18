package echcheck

import (
	"context"
	"crypto/tls"
	"net"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	utls "gitlab.com/yawning/utls.git"
)

type tlsHandshakerWithExtensions struct {
	extensions []utls.TLSExtension
	dl         model.DebugLogger
	id         *utls.ClientHelloID
}

var _ model.TLSHandshaker = &tlsHandshakerWithExtensions{}

// newHandshakerWithExtensions returns a NewHandshaker function for creating
// tlsHandshakerWithExtensions instances.
func newHandshakerWithExtensions(extensions []utls.TLSExtension) func(dl model.DebugLogger, id *utls.ClientHelloID) model.TLSHandshaker {
	return func(dl model.DebugLogger, id *utls.ClientHelloID) model.TLSHandshaker {
		return &tlsHandshakerWithExtensions{
			extensions: extensions,
			dl:         dl,
			id:         id,
		}
	}
}

func (t *tlsHandshakerWithExtensions) Handshake(
	ctx context.Context, tcpConn net.Conn, tlsConfig *tls.Config) (model.TLSConn, error) {
	tlsConn, err := netxlite.NewUTLSConn(tcpConn, tlsConfig, t.id)
	runtimex.Assert(err == nil, "unexpected error when creating UTLSConn")

	if t.extensions != nil && len(t.extensions) != 0 {
		tlsConn.BuildHandshakeState()
		tlsConn.Extensions = append(tlsConn.Extensions, t.extensions...)
	}

	if err := tlsConn.Handshake(); err != nil {
		return nil, err
	}

	return tlsConn, nil
}
