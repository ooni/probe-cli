package main

import "io/fs"

// testBuildDeps contains the test buildDeps.
type testBuildDeps struct {
	MockGolangCheck func()

	MockLinuxReadGOVERSION func(filename string) []byte

	MockLinuxWriteDockerfile func(filename string, content []byte, mode fs.FileMode)

	MockPsiphonMaybeCopyConfigFiles func()

	MockPsiphonFilesExist func() bool

	MockWindowsMingwCheck func()
}

var _ buildDeps = &testBuildDeps{}

// golangCheck implements buildDeps
func (d *testBuildDeps) golangCheck() {
	d.MockGolangCheck()
}

// linuxReadGOVERSION implements buildDeps
func (d *testBuildDeps) linuxReadGOVERSION(filename string) []byte {
	return d.MockLinuxReadGOVERSION(filename)
}

// linuxWriteDockerfile implements buildDeps
func (d *testBuildDeps) linuxWriteDockerfile(filename string, content []byte, mode fs.FileMode) {
	d.MockLinuxWriteDockerfile(filename, content, mode)
}

// psiphonFilesExist implements buildDeps
func (d *testBuildDeps) psiphonFilesExist() bool {
	return d.MockPsiphonFilesExist()
}

// psiphonMaybeCopyConfigFiles implements buildDeps
func (d *testBuildDeps) psiphonMaybeCopyConfigFiles() {
	d.MockPsiphonMaybeCopyConfigFiles()
}

// windowsMingwCheck implements buildDeps
func (d *testBuildDeps) windowsMingwCheck() {
	d.MockWindowsMingwCheck()
}
