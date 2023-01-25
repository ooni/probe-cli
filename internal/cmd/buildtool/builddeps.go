package main

import (
	"io/fs"

	"github.com/ooni/probe-cli/v3/internal/cmd/buildtool/internal/buildtoolmodel"
	"github.com/ooni/probe-cli/v3/internal/must"
)

// buildDeps is the default buildDeps implementation
type buildDeps struct{}

var _ buildtoolmodel.Dependencies = &buildDeps{}

// GolangCheck implements buildtoolmodel.Dependencies
func (*buildDeps) GolangCheck() {
	golangCheck("GOVERSION")
}

// LinuxReadGOVERSION implements buildtoolmodel.Dependencies
func (*buildDeps) LinuxReadGOVERSION(filename string) []byte {
	return must.ReadFile(filename)
}

// LinuxWriteDockerfile implements buildtoolmodel.Dependencies
func (*buildDeps) LinuxWriteDockerfile(filename string, content []byte, mode fs.FileMode) {
	must.WriteFile(filename, content, mode)
}

// PsiphonFilesExist implements buildtoolmodel.Dependencies
func (*buildDeps) PsiphonFilesExist() bool {
	return psiphonFilesExist()
}

// PsiphonMaybeCopyConfigFiles implements buildtoolmodel.Dependencies
func (*buildDeps) PsiphonMaybeCopyConfigFiles() {
	psiphonMaybeCopyConfigFiles()
}

// WindowsMingwCheck implements buildtoolmodel.Dependencies
func (*buildDeps) WindowsMingwCheck() {
	//windowsMingwCheck() /* TODO(bassosimone) */
}
