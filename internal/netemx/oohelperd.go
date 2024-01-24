package netemx

import (
	"fmt"
	"net/http"

	"github.com/apex/log"
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
	logger := &logx.PrefixLogger{
		Prefix: fmt.Sprintf("%-16s", "TH_HANDLER"),
		Logger: log.Log,
	}
	handler := oohelperd.NewHandler(logger, netx)

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
			netxlite.NewHTTPTransportWithResolver,
		)
	}

	handler.NewHTTP3Client = func(logger model.Logger) model.HTTPClient {
		return oohelperd.NewHTTPClientWithTransportFactory(
			netx, logger,
			netxlite.NewHTTP3TransportWithResolver,
		)
	}

	return handler
}
