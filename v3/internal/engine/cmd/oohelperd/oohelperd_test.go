package main

import (
	"testing"
)

func TestSmoke(t *testing.T) {
	// Just check whether we can start and then tear down the server, so
	// we have coverage of this code and when we see that some lines aren't
	// covered we know these are genuine places where we're not testing
	// the code rather than just places like this simple main.
	go testableMain()
	srvcancel()  // kills the listener
	srvwg.Wait() // joined
}
