package dslvm

import "context"

// Stage is a stage in the DSL graph.
type Stage interface {
	Run(ctx context.Context, rtx Runtime)
}
