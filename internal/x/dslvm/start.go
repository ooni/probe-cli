package dslvm

import "context"

// Start starts all the given [Stage] instances.
func Start(ctx context.Context, rtx Runtime, stages ...Stage) {
	for _, stage := range stages {
		go stage.Run(ctx, rtx)
	}
}
