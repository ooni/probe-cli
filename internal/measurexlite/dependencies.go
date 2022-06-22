package measurexlite

import (
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// dependencies allows to mock dependencies in testing.
//
// The nil dependencies struct calls actual constructors inside
// netxlite or other dependent packages.
//
// When dependencies is not nil, we check whether the function
// pointer corresponding to the correct constructor is not nil. In
// such a case, we call the given function pointer, otherwise we
// fall back to calling the actual dependent constructor. For
// example, dependencies.NewDialerWithoutResolver calls its
// newDialerWithoutResolver function, if not nil. Otherwise, it
// calls netxlite.NewDialerWithoutResolver.
type dependencies struct {
	// newDialerWithoutResolver allows to hijack calls to the
	// netxlite.NewDialerWithoutResolver factory.
	newDialerWithoutResolver func(dl model.DebugLogger) model.Dialer
}

// NewDialerWithoutResolver calls netxlite.NewDialerWithoutResolver
// if the struct receiver is nil or newDialerWithoutResolver is
// nil. Otherwise, it calls newDialerWithoutResolver.
func (d *dependencies) NewDialerWithoutResolver(dl model.DebugLogger) model.Dialer {
	if d != nil && d.newDialerWithoutResolver != nil {
		return d.newDialerWithoutResolver(dl)
	}
	return netxlite.NewDialerWithoutResolver(dl)
}
