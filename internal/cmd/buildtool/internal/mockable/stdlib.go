package mockable

import (
	"io"
	"io/fs"

	"github.com/ooni/probe-cli/v3/internal/cmd/buildtool/internal/stdlibx"
)

// Stdlib mocks stdlibx.Stdlib.
type Stdlib struct {
	MockCopyFile func(source string, dest string) error

	MockExitOnError func(err error, message string)

	MockMustFprintf func(w io.Writer, format string, v ...any)

	MockMustNewCommand func(binary string, args ...string) stdlibx.Command

	MockMustReadFileFirstLine func(filename string) string

	MockMustRun func(binpath string, args ...string)

	MockMustRunAndReadFirstLine func(binpath string, args ...string) string

	MockMustWriteFile func(filename string, data []byte, perms fs.FileMode)

	MockRegularFileExists func(filename string) bool

	MockRun func(binpath string, args ...string) error
}

var _ stdlibx.Stdlib = &Stdlib{}

// CopyFile implements stdlibx.Stdlib
func (s *Stdlib) CopyFile(source string, dest string) error {
	return s.MockCopyFile(source, dest)
}

// ExitOnError implements stdlibx.Stdlib
func (s *Stdlib) ExitOnError(err error, message string) {
	s.MockExitOnError(err, message)
}

// MustFprintf implements stdlibx.Stdlib
func (s *Stdlib) MustFprintf(w io.Writer, format string, v ...any) {
	s.MockMustFprintf(w, format, v...)
}

// MustNewCommand implements stdlibx.Stdlib
func (s *Stdlib) MustNewCommand(binary string, args ...string) stdlibx.Command {
	return s.MockMustNewCommand(binary, args...)
}

// MustReadFileFirstLine implements stdlibx.Stdlib
func (s *Stdlib) MustReadFileFirstLine(filename string) string {
	return s.MockMustReadFileFirstLine(filename)
}

// MustRun implements stdlibx.Stdlib
func (s *Stdlib) MustRun(binpath string, args ...string) {
	s.MockMustRun(binpath, args...)
}

// MustRunAndReadFirstLine implements stdlibx.Stdlib
func (s *Stdlib) MustRunAndReadFirstLine(binpath string, args ...string) string {
	return s.MockMustRunAndReadFirstLine(binpath, args...)
}

// MustWriteFile implements stdlibx.Stdlib
func (s *Stdlib) MustWriteFile(filename string, data []byte, perms fs.FileMode) {
	s.MockMustWriteFile(filename, data, perms)
}

// RegularFileExists implements stdlibx.Stdlib
func (s *Stdlib) RegularFileExists(filename string) bool {
	return s.MockRegularFileExists(filename)
}

// Run implements stdlibx.Stdlib
func (s *Stdlib) Run(binpath string, args ...string) error {
	return s.MockRun(binpath, args...)
}
