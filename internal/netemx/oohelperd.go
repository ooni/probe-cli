package netemx

import (
	"fmt"
	"net/http"

	"github.com/apex/log"
	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/logx"
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
	return handler
}
