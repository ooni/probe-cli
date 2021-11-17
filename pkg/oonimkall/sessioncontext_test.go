package oonimkall

import (
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/atomicx"
)

func TestClampTimeout(t *testing.T) {
	if clampTimeout(-1, maxTimeout) != -1 {
		t.Fatal("unexpected result here")
	}
	if clampTimeout(0, maxTimeout) != 0 {
		t.Fatal("unexpected result here")
	}
	if clampTimeout(60, maxTimeout) != 60 {
		t.Fatal("unexpected result here")
	}
	if clampTimeout(maxTimeout, maxTimeout) != maxTimeout {
		t.Fatal("unexpected result here")
	}
	if clampTimeout(maxTimeout+1, maxTimeout) != maxTimeout {
		t.Fatal("unexpected result here")
	}
}

func TestNewContextWithZeroTimeout(t *testing.T) {
	here := &atomicx.Int64{}
	ctx, cancel := newContext(0)
	defer cancel()
	go func() {
		<-time.After(250 * time.Millisecond)
		here.Add(1)
		cancel()
	}()
	<-ctx.Done()
	if here.Load() != 1 {
		t.Fatal("context timeout not working as intended")
	}
}

func TestNewContextWithNegativeTimeout(t *testing.T) {
	here := &atomicx.Int64{}
	ctx, cancel := newContext(-1)
	defer cancel()
	go func() {
		<-time.After(250 * time.Millisecond)
		here.Add(1)
		cancel()
	}()
	<-ctx.Done()
	if here.Load() != 1 {
		t.Fatal("context timeout not working as intended")
	}
}

func TestNewContextWithHugeTimeout(t *testing.T) {
	here := &atomicx.Int64{}
	ctx, cancel := newContext(maxTimeout + 1)
	defer cancel()
	go func() {
		<-time.After(250 * time.Millisecond)
		here.Add(1)
		cancel()
	}()
	<-ctx.Done()
	if here.Load() != 1 {
		t.Fatal("context timeout not working as intended")
	}
}

func TestNewContextWithReasonableTimeout(t *testing.T) {
	here := &atomicx.Int64{}
	ctx, cancel := newContext(1)
	defer cancel()
	go func() {
		<-time.After(5 * time.Second)
		here.Add(1)
		cancel()
	}()
	<-ctx.Done()
	if here.Load() != 0 {
		t.Fatal("context timeout not working as intended")
	}
}

func TestNewContextWithArtificiallyLowMaxTimeout(t *testing.T) {
	here := &atomicx.Int64{}
	const maxTimeout = 2
	ctx, cancel := newContextEx(maxTimeout+1, maxTimeout)
	defer cancel()
	go func() {
		<-time.After(30 * time.Second)
		here.Add(1)
		cancel()
	}()
	<-ctx.Done()
	if here.Load() != 0 {
		t.Fatal("context timeout not working as intended")
	}
}
