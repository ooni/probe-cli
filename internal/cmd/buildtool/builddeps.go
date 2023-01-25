package main

import (
	"io/fs"

	"github.com/ooni/probe-cli/v3/internal/cmd/buildtool/internal/buildtoolmodel"
	"github.com/ooni/probe-cli/v3/internal/must"
)

// buildDependencies is the default buildDeps implementation
type buildDependencies struct{}

var _ buildtoolmodel.Dependencies = &buildDependencies{}

// GolangCheck implements buildtoolmodel.Dependencies
func (*buildDependencies) GolangCheck() {
	golangCheck()
}

// LinuxReadGOVERSION implements buildtoolmodel.Dependencies
func (*buildDependencies) LinuxReadGOVERSION(filename string) []byte {
	return must.ReadFile(filename)
}

// LinuxWriteDockerfile implements buildtoolmodel.Dependencies
func (*buildDependencies) LinuxWriteDockerfile(filename string, content []byte, mode fs.FileMode) {
	must.WriteFile(filename, content, mode)
}

// PsiphonFilesExist implements buildtoolmodel.Dependencies
func (*buildDependencies) PsiphonFilesExist() bool {
	return psiphonFilesExist()
}

// PsiphonMaybeCopyConfigFiles implements buildtoolmodel.Dependencies
func (*buildDependencies) PsiphonMaybeCopyConfigFiles() {
	psiphonMaybeCopyConfigFiles()
}

// WindowsMingwCheck implements buildtoolmodel.Dependencies
func (*buildDependencies) WindowsMingwCheck() {
	windowsMingwCheck()
}
