package dslvm

import "github.com/ooni/probe-cli/v3/internal/model"

// Closer is something that [Drop] should explicitly close.
type Closer interface {
	Close(logger model.Logger) error
}
