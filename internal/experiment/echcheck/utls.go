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
	conn       *netxlite.UTLSConn
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
	ctx context.Context, conn net.Conn, tlsConfig *tls.Config) (model.TLSConn, error) {
	var err error
	t.conn, err = netxlite.NewUTLSConn(conn, tlsConfig, t.id)
	runtimex.Assert(err == nil, "unexpected error when creating UTLSConn")

	if t.extensions != nil && len(t.extensions) != 0 {
		t.conn.BuildHandshakeState()
		t.conn.Extensions = append(t.conn.Extensions, t.extensions...)
	}

	if err := t.conn.Handshake(); err != nil {
		return nil, err
	}

	// TODO(bassosimone): I don't understand why we're storing the conn inside
	// of the TLS handshaker with extensions structure, but I don't like it
	return t.conn, nil
}
