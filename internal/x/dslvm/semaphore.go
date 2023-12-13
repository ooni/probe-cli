package dslvm

import (
	"log"
	"os"
	"time"

	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

var semaphoreDebug = os.Getenv("OONI_DEBUG_SEMAPHORE") == "1"

// Semaphore implements a semaphore.
//
// See https://en.wikipedia.org/wiki/Semaphore_(programming).
type Semaphore struct {
	name string
	ch   chan bool
}

// NewSemaphore creates a new [*Semaphore] with the given count of available resources. This
// function PANICS if the given count of available resources is zero or negative.
func NewSemaphore(name string, count int) *Semaphore {
	runtimex.Assert(count >= 1, "expected count to be >= 1")
	sema := &Semaphore{
		ch:   make(chan bool, count),
		name: name,
	}
	for idx := 0; idx < count; idx++ {
		sema.ch <- true
	}

	if semaphoreDebug {
		log.Printf("semaphore %s[%p]: NEW[%d]", sema.name, sema, count)
	}

	return sema
}

// Signal signals that a resource is now available.
func (sema *Semaphore) Signal() {
	if semaphoreDebug {
		log.Printf("semaphore %s[%p]: SIGNAL", sema.name, sema)
	}

	sema.ch <- true
}

// Wait waits for a resource to be available.
func (sema *Semaphore) Wait() {
	var t0 time.Time
	if semaphoreDebug {
		log.Printf("semaphore %s[%p]: WAIT", sema.name, sema)
		t0 = time.Now()
	}

	<-sema.ch

	if semaphoreDebug {
		log.Printf("semaphore %s[%p]: READY (%v)", sema.name, sema, time.Since(t0))
	}
}
