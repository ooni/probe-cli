package main

//
// Shell functions
//

import (
	"fmt"

	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"golang.org/x/sys/execabs"
)

// execute executes a command.
func execute(cmd string, args ...string) {
	c := execabs.Command(cmd, args...)
	fmt.Printf("+ %s\n", c.String())
	err := c.Run()
	runtimex.PanicOnError(err, "c.Run failed")
}

// chdirAndExecute executes a command inside a directory
func chdirAndExecute(workdir string, cmd string, args ...string) {
	c := execabs.Command(cmd, args...)
	c.Dir = workdir
	fmt.Printf("+ (cd %s && %s)\n", workdir, c.String())
	err := c.Run()
	runtimex.PanicOnError(err, "c.Run failed")
}
