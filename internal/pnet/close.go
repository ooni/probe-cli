package pnet

import (
	"context"
	"io"
	"log"
)

// Close returns a [Stage] that closes open resources.
func Close[T io.Closer]() Stage[T, Void] {
	return stageForAction[T, Void](actionFunc[T, Void](closeAction[T]))
}

// closeAction is the [action] that closes open resources.
func closeAction[T io.Closer](ctx context.Context, closer T, outputs chan<- Result[Void]) {
	// TODO(bassosimone): I would like to print close logs here
	log.Printf("ELLIOT: Close %v", closer)
	err := closer.Close()
	if err != nil {
		outputs <- NewResultError[Void](err)
		return
	}
	outputs <- NewResultValue(Void{})
}
