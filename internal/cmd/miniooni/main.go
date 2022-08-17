// Command miniooni is a simple binary for research and QA purposes
// with a CLI interface similar to MK and OONI Probe v2.x.
package main

//
// Main function
//

import (
	"fmt"
	"os"
)

func main() {
	defer func() {
		if s := recover(); s != nil {
			fmt.Fprintf(os.Stderr, "FATAL: %s\n", s)
		}
	}()
	Main()
}
