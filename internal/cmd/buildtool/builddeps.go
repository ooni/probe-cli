package main

//
// Allows unit testing build rules by abstracting
// specific activities through an interface.
//

import (
	"io/fs"
	"runtime"

	"github.com/ooni/probe-cli/v3/internal/cmd/buildtool/internal/buildtoolmodel"
	"github.com/ooni/probe-cli/v3/internal/must"
)

// buildDeps is the default buildDeps implementation
type buildDeps struct{}

var _ buildtoolmodel.Dependencies = &buildDeps{}

// AbsoluteCurDir implements buildtoolmodel.Dependencies
func (*buildDeps) AbsoluteCurDir() string {
	return cdepsMustAbsoluteCurdir()
}

// AndroidNDKCheck implements buildtoolmodel.Dependencies
func (*buildDeps) AndroidNDKCheck(androidHome string) string {
	return androidNDKCheck(androidHome)
}

// AndroidSDKCheck implements buildtoolmodel.Dependencies
func (*buildDeps) AndroidSDKCheck() string {
	return androidSDKCheck()
}

// GOPATH implements buildtoolmodel.Dependencies
func (*buildDeps) GOPATH() string {
	return golangGOPATH()
}

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

// MustChdir implements buildtoolmodel.Dependencies
func (*buildDeps) MustChdir(dirname string) func() {
	return cdepsMustChdir(dirname)
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
	windowsMingwCheck()
}

// GOOS implements buildtoolmodel.Dependencies
func (*buildDeps) GOOS() string {
	return runtime.GOOS
}

// VerifySHA256 implements buildtoolmodel.Dependencies
func (*buildDeps) VerifySHA256(expectedSHA256 string, tarball string) {
	cdepsMustVerifySHA256(expectedSHA256, tarball)
}
