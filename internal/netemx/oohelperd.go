package netemx

import (
	"fmt"
	"net/http"

	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/logx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/oohelperd"
)

// OOHelperDFactory is the factory to create an [http.Handler] implementing the OONI Web Connectivity
// test helper using a specific [netem.UnderlyingNetwork].
type OOHelperDFactory struct{}

var _ HTTPHandlerFactory = &OOHelperDFactory{}

// NewHandler implements QAEnvHTTPHandlerFactory.NewHandler.
func (f *OOHelperDFactory) NewHandler(env NetStackServerFactoryEnv, unet *netem.UNetStack) http.Handler {
	netx := &netxlite.Netx{Underlying: &netxlite.NetemUnderlyingNetworkAdapter{UNet: unet}}
	handler := oohelperd.NewHandler()

	handler.BaseLogger = &logx.PrefixLogger{
		Prefix: fmt.Sprintf("%-16s", "TH_HANDLER"),
		Logger: handler.BaseLogger,
	}

	handler.NewDialer = func(logger model.Logger) model.Dialer {
		return netx.NewDialerWithResolver(logger, netx.NewStdlibResolver(logger))
	}

	handler.NewQUICDialer = func(logger model.Logger) model.QUICDialer {
		return netx.NewQUICDialerWithResolver(
			netx.NewUDPListener(),
			logger,
			netx.NewStdlibResolver(logger),
		)
	}

	handler.NewResolver = func(logger model.Logger) model.Resolver {
		return netx.NewStdlibResolver(logger)
	}

	handler.NewHTTPClient = func(logger model.Logger) model.HTTPClient {
		return oohelperd.NewHTTPClientWithTransportFactory(
			netx, logger,
			func(netx *netxlite.Netx, dl model.DebugLogger, r model.Resolver) model.HTTPTransport {
				dialer := netx.NewDialerWithResolver(dl, r)
				tlsDialer := netxlite.NewTLSDialer(dialer, netx.NewTLSHandshakerStdlib(dl))
				// TODO(https://github.com/ooni/probe/issues/2534): NewHTTPTransport is QUIRKY but
				// we probably don't care about using a QUIRKY function here
				return netxlite.NewHTTPTransport(dl, dialer, tlsDialer)
			},
		)
	}

	handler.NewHTTP3Client = func(logger model.Logger) model.HTTPClient {
		return oohelperd.NewHTTPClientWithTransportFactory(
			netx, logger,
			func(netx *netxlite.Netx, dl model.DebugLogger, r model.Resolver) model.HTTPTransport {
				qd := netx.NewQUICDialerWithResolver(netx.NewUDPListener(), dl, r)
				return netxlite.NewHTTP3Transport(dl, qd, nil)
			},
		)
	}

	return handler
}
