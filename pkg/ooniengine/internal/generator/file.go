package main

//
// File I/O functions
//

import (
	"fmt"
	"io"
	"os"

	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// writeFile writes onto the given file.
func writeFile(w io.Writer, format string, v ...any) {
	_, err := fmt.Fprintf(w, format, v...)
	runtimex.PanicOnError(err, "fmt.Fprintf failed")
}

// openFile is a convenience function for opening a file for writing.
func openFile(name string) *os.File {
	fp, err := os.Create(name)
	runtimex.PanicOnError(err, "os.Create failed")
	return fp
}

// closeFile is a convenience function for closing a file.
func closeFile(fp *os.File) {
	err := fp.Close()
	runtimex.PanicOnError(err, "fp.Close failed")
}
