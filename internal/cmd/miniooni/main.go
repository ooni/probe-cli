// Command miniooni is a simple binary for research and QA purposes
// with a CLI interface similar to MK and OONI Probe v2.x.
//
// See also libminiooni, which is where we implement this CLI.
package main

import (
	"fmt"
	"os"

	"github.com/ooni/probe-cli/v3/internal/libminiooni"
)

func main() {
	defer func() {
		if s := recover(); s != nil {
			fmt.Fprintf(os.Stderr, "%s", s)
		}
	}()
	libminiooni.Main()
}
