package measurexlite

import "github.com/ooni/probe-cli/v3/internal/model"

// NewUDPListener implements model.Measuring Network.
func (tx *Trace) NewUDPListener() model.UDPListener {
	return tx.Netx.NewUDPListener()
}
