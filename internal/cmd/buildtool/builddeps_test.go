package main

// testBuildDeps contains the test buildDeps.
type testBuildDeps struct {
	MockGolangCheck func()

	MockPsiphonMaybeCopyConfigFiles func()

	MockPsiphonFilesExist func() bool

	MockWindowsMingwCheck func()
}

var _ buildDeps = &testBuildDeps{}

// golangCheck implements buildDeps
func (d *testBuildDeps) golangCheck() {
	d.MockGolangCheck()
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
