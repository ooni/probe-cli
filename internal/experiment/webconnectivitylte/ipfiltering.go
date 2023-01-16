package webconnectivity

//
// Filter out IP addresses to which we're not permitted to connect.
//

import (
	"errors"
	"fmt"
	"net"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// errNotAllowedToConnect indicates we're not allowed to connect.
var errNotAllowedToConnect = errors.New("webconnectivity: not allowed to connect")

// allowedToConnect returns whether we can connect to the given endpoint.
func allowedToConnect(endpoint string) error {
	addr, _, err := net.SplitHostPort(endpoint)
	if err != nil {
		return fmt.Errorf("%w: %s", errNotAllowedToConnect, err.Error())
	}
	// Implementation note: we don't remove bogons because accessing
	// them can lead us to discover block pages. This may change in
	// the future, see https://github.com/ooni/probe/issues/2327.
	//
	// We prevent connecting to localhost, however, as documented
	// inside https://github.com/ooni/probe/issues/2397.
	if netxlite.IsLoopback(addr) {
		return fmt.Errorf("%w: is loopback", errNotAllowedToConnect)
	}
	return nil
}
