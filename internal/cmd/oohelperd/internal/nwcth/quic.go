package nwcth

import (
	"context"
	"crypto/tls"

	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/quicdialer"
	"github.com/ooni/probe-cli/v3/internal/errorsx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// newQUICDialer contructs a new dialer for QUIC connections,
// with default, errorwrapping and resolve functionalities
func newQUICDialerResolver(resolver netxlite.Resolver) netxlite.QUICContextDialer {
	var ql quicdialer.QUICListener = &netxlite.QUICListenerStdlib{}
	ql = &errorsx.ErrorWrapperQUICListener{QUICListener: ql}
	var d quicdialer.ContextDialer = &netxlite.QUICDialerQUICGo{
		QUICListener: ql,
	}
	d = &errorsx.ErrorWrapperQUICDialer{Dialer: d}
	d = &netxlite.QUICDialerResolver{Resolver: resolver, Dialer: d}
	return d
}

// QUICDo performs the QUIC check.
func QUICDo(ctx context.Context, endpoint string, tlsConf *tls.Config, quicdialer netxlite.QUICContextDialer) (quic.EarlySession, error) {
	return quicdialer.DialContext(ctx, "udp", endpoint, tlsConf, &quic.Config{})
}
