// Package buildtoolmodel contains the model for buildtool.
package buildtoolmodel

import "io/fs"

// Dependencies abstracts the commands and checks required
// to perform all the builds in this package.
type Dependencies interface {
	// AbsoluteCurDir returns the absolute current directory.
	AbsoluteCurDir() string

	// AndroidNDKCheck checks we have the right NDK
	// inside the SDK and returns its dir.
	AndroidNDKCheck(androidHome string) string

	// AndroidSDKCheck ensures we have the right
	// tools installed to build for Android. This
	// function returns the Android home path.
	AndroidSDKCheck() string

	// GOPATH returns the current GOPATH.
	GOPATH() string

	// GolangCheck ensures we have the correct
	// version of golang as the "go" binary.
	GolangCheck()

	// LinuxWriteDockerfile writes the dockerfile for linux.
	LinuxWriteDockerfile(filename string, content []byte, mode fs.FileMode)

	// LinuxReadGOVERSION reads the GOVERSION file for linux.
	LinuxReadGOVERSION(filename string) []byte

	// MustChdir changes the current working directory and returns the
	// function to return to the original working directory.
	MustChdir(dirname string) func()

	// PsiphonFilesExist returns true if the psiphon
	// config files are in the correct location.
	PsiphonFilesExist() bool

	// PsiphonMaybeCopyConfigFiles copies psiphon
	// config files if possible
	PsiphonMaybeCopyConfigFiles()

	// VerifySHA256 verifies that the tarball has the given checksum.
	VerifySHA256(expectedSHA256, tarball string)

	// WindowsMingwCheck makes sure we're using the
	// expected version of mingw-w64.
	WindowsMingwCheck()

	// GOOS returns the current GOOS.
	GOOS() string
}
