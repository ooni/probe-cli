package netemx

import (
	"fmt"
	"net/http"
	"net/http/cookiejar"

	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/logx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/oohelperd"
	"golang.org/x/net/publicsuffix"
)

// OOHelperDFactory is the factory to create an [http.Handler] implementing the OONI Web Connectivity
// test helper using a specific [netem.UnderlyingNetwork].
type OOHelperDFactory struct{}

var _ QAEnvHTTPHandlerFactory = &OOHelperDFactory{}

// NewHandler implements QAEnvHTTPHandlerFactory.NewHandler.
func (f *OOHelperDFactory) NewHandler(unet netem.UnderlyingNetwork) http.Handler {
	netx := netxlite.Netx{Underlying: &netxlite.NetemUnderlyingNetworkAdapter{UNet: unet}}
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
			netx.NewQUICListener(),
			logger,
			netx.NewStdlibResolver(logger),
		)
	}

	handler.NewResolver = func(logger model.Logger) model.Resolver {
		return netx.NewStdlibResolver(logger)
	}

	handler.NewHTTPClient = func(logger model.Logger) model.HTTPClient {
		cookieJar, _ := cookiejar.New(&cookiejar.Options{
			PublicSuffixList: publicsuffix.List,
		})
		return &http.Client{
			Transport:     netx.NewHTTPTransportStdlib(logger),
			CheckRedirect: nil,
			Jar:           cookieJar,
			Timeout:       0,
		}
	}

	handler.NewHTTP3Client = func(logger model.Logger) model.HTTPClient {
		cookieJar, _ := cookiejar.New(&cookiejar.Options{
			PublicSuffixList: publicsuffix.List,
		})
		return &http.Client{
			Transport:     netx.NewHTTP3TransportStdlib(logger),
			CheckRedirect: nil,
			Jar:           cookieJar,
			Timeout:       0,
		}
	}

	return handler
}
