package measurexlite

//
// Support for parroting Chrome with ooni/utls-light
//

import "github.com/ooni/probe-cli/v3/internal/model"

// NewTLSHandshakerChrome is equivalent to netxlite.Netx.NewTLSHandshakerChrome
// except that it returns a model.TLSHandshaker that uses this trace.
func (tx *Trace) NewTLSHandshakerChrome(dl model.DebugLogger) model.TLSHandshaker {
	return &tlsHandshakerTrace{
		thx: tx.Netx.NewTLSHandshakerChrome(dl),
		tx:  tx,
	}
}
