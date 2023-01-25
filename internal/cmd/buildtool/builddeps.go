package main

import (
	"io/fs"

	"github.com/ooni/probe-cli/v3/internal/must"
)

// buildDeps abstracts the commands and checks required
// to perform all the builds in this package.
type buildDeps interface {
	// psiphonMaybeCopyConfigFiles copies psiphon
	// config files if possible
	psiphonMaybeCopyConfigFiles()

	// golangCheck ensures we have the correct
	// version of golang as the "go" binary.
	golangCheck()

	// linuxWriteDockerfile writes the dockerfile for linux.
	linuxWriteDockerfile(filename string, content []byte, mode fs.FileMode)

	// linuxReadGOVERSION reads the GOVERSION file for linux.
	linuxReadGOVERSION(filename string) []byte

	// psiphonFilesExist returns true if the psiphon
	// config files are in the correct location.
	psiphonFilesExist() bool

	// windowsMingwCheck makes sure we're using the
	// expected version of mingw-w64.
	windowsMingwCheck()
}

// buildDependencies is the default buildDeps implementation
type buildDependencies struct{}

var _ buildDeps = &buildDependencies{}

// golangCheck implements buildDeps
func (*buildDependencies) golangCheck() {
	golangCheck()
}

// linuxReadGOVERSION implements buildDeps
func (*buildDependencies) linuxReadGOVERSION(filename string) []byte {
	return must.ReadFile(filename)
}

// linuxWriteDockerfile implements buildDeps
func (*buildDependencies) linuxWriteDockerfile(filename string, content []byte, mode fs.FileMode) {
	must.WriteFile(filename, content, mode)
}

// psiphonFilesExist implements buildDeps
func (*buildDependencies) psiphonFilesExist() bool {
	return psiphonFilesExist()
}

// psiphonMaybeCopyConfigFiles implements buildDeps
func (*buildDependencies) psiphonMaybeCopyConfigFiles() {
	psiphonMaybeCopyConfigFiles()
}

// windowsMingwCheck implements buildDeps
func (*buildDependencies) windowsMingwCheck() {
	windowsMingwCheck()
}
