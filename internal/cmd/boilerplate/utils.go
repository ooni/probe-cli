package main

//
// Utility functions
//

import (
	"fmt"
	"os"
	"text/template"

	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// Permissions with which we create new directories
const newDirPermissions = 0755

// Helper to write less when printing to stdout
func printf(format string, args ...any) {
	fmt.Fprintf(os.Stdout, format, args...)
}

// Permissions with which we create new files
const newFilePermissions = 0644

// Creates a file for writing
func openForWriting(filepath string) *os.File {
	filep, err := os.OpenFile(filepath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, newFilePermissions)
	runtimex.PanicOnError(err, "os.OpenFile failed")
	return filep
}

// Ensures that we close a file without I/O errors
func closeFile(fp *os.File) {
	err := fp.Close()
	runtimex.PanicOnError(err, "fp.Close failed")
}

// Generic function for writing a template.
func writeTemplate(fullpath string, tmpl *template.Template, info any) {
	printf("🚧 generating %s...\n", fullpath)
	filep := openForWriting(fullpath)
	defer closeFile(filep)
	err := tmpl.Execute(filep, info)
	runtimex.PanicOnError(err, "cannot execute a text/template")
}

// Creates directories recursively
func mkdirP(fulldir string) {
	printf("🐚 mkdir -p %s\n", fulldir)
	err := os.MkdirAll(fulldir, newDirPermissions)
	runtimex.PanicOnError(err, "o}s.MkdirAll failed")
}
