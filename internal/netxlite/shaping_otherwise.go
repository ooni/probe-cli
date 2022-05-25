//go:build !shaping

package netxlite

import (
	"github.com/ooni/probe-cli/v3/internal/model"
)

func newMaybeShapingDialer(dialer model.Dialer) model.Dialer {
	return dialer
}
