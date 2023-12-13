package dsljson

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/x/dslvm"
)

// Run runs the DSL represented by the given [*RootNode].
func Run(ctx context.Context, rtx dslvm.Runtime, root *RootNode) error {
	lx := newLoader()
	if err := lx.load(rtx.Logger(), root); err != nil {
		return err
	}
	for _, stage := range lx.stages {
		go stage.Run(ctx, rtx)
	}
	dslvm.Wait(lx.toWait...)
	return nil
}
