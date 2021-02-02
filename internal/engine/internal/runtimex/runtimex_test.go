package runtimex_test

import (
	"errors"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/engine/internal/runtimex"
)

func TestGood(t *testing.T) {
	runtimex.PanicOnError(nil, "antani failed")
}

func TestBad(t *testing.T) {
	expected := errors.New("mocked error")
	if !errors.Is(badfunc(expected), expected) {
		t.Fatal("not the error we expected")
	}
}

func badfunc(in error) (out error) {
	defer func() {
		out = recover().(error)
	}()
	runtimex.PanicOnError(in, "antani failed")
	return
}
