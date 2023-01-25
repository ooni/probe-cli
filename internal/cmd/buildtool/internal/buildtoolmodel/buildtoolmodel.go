// Package buildtoolmodel contains the model for buildtool.
package buildtoolmodel

import "io/fs"

// Dependencies abstracts the commands and checks required
// to perform all the builds in this package.
type Dependencies interface {
	// PsiphonMaybeCopyConfigFiles copies psiphon
	// config files if possible
	PsiphonMaybeCopyConfigFiles()

	// GolangCheck ensures we have the correct
	// version of golang as the "go" binary.
	GolangCheck()

	// LinuxWriteDockerfile writes the dockerfile for linux.
	LinuxWriteDockerfile(filename string, content []byte, mode fs.FileMode)

	// LinuxReadGOVERSION reads the GOVERSION file for linux.
	LinuxReadGOVERSION(filename string) []byte

	// PsiphonFilesExist returns true if the psiphon
	// config files are in the correct location.
	PsiphonFilesExist() bool

	// WindowsMingwCheck makes sure we're using the
	// expected version of mingw-w64.
	WindowsMingwCheck()
}
