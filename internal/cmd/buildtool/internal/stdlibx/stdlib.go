package stdlibx

//
// Stdlib interface and implementation
//

import (
	"io"
	"io/fs"
	"os"
)

// Stdlib models the standard library.
type Stdlib interface {
	// CopyFile copies from source to dest and returns
	// the error that occurred on failure or nil.
	CopyFile(source, dest string) error

	// ExitOnError calls os.Exit in case of err is not nil.
	ExitOnError(err error, message string)

	// MustFprintf is like fmt.Fprintf but calls os.Exit on error.
	MustFprintf(w io.Writer, format string, v ...any)

	// MustNewCommand is like exec.Command but calls os.Exit if
	// we cannot locate the given binary. The new command will be
	// configured as follows:
	//
	// 1. we will execute the binary;
	//
	// 2. the binary and the args will be part of the argv;
	//
	// 3. the environment will be a copy of os.Environ;
	//
	// 4. the stdin, stdout, stderr will be connected to the
	// current program's stdin, stdout, and stderr.
	MustNewCommand(binary string, args ...string) Command

	// MustReadFileFirstLine reads and returns the first line of the
	// given file as an UTF-8 string or calls os.Exit on error.
	MustReadFileFirstLine(filename string) string

	// MustRun is like Run but calls os.Exit on error.
	MustRun(binpath string, args ...string)

	// MustRunAndReadFirstLine runs the given command and reads the first
	// line as an UTF-8 string or calls os.Exit on error.
	MustRunAndReadFirstLine(binpath string, args ...string) string

	// MustWriteFile is like os.WriteFile but calls os.Exit on error.
	MustWriteFile(filename string, data []byte, perms fs.FileMode)

	// RegularFileExists returns true when the given filename
	// exists and it is a regular file.
	RegularFileExists(filename string) bool

	// Run runs the given command and returns an error, in
	// case of failure, or nil, in case of success.
	Run(binpath string, args ...string) error
}

// Command is a command ready to execute created by the [Stdlib].
type Command interface {
	// AddArgs appends the given args to the command line.
	AddArgs(args ...string)

	// AddEnv adds the given environment variable to the
	// commands' environment. This function calls os.Exit if
	// we cannot write log messages to the stderr.
	AddEnv(key, value string)

	// MustRun is like Run but calls os.Exit if we cannot execute
	// the command or the command exits with a nonzero code.
	MustRun()

	// Run runs the command and returns the error that occurred
	// though this command might call os.Exit if we cannot
	// write log messages to the standard error.
	Run() error

	// SetStdout sets the command's stdout. You can use nil to
	// suppress using os.Stdout, which is the default.
	SetStdout(w io.Writer)

	// SetStderr sets the command's stderr. You can use nil to
	// suppress using os.Stderr, which is the default.
	SetStderr(w io.Writer)
}

// exiter is what allows you to call os.Exit.
type exiter interface {
	Exit(int)
}

// realExiter implements exiter using the os.Exit call.
type realExiter struct{}

func (*realExiter) Exit(code int) {
	os.Exit(code)
}

// NewStdlib creates a new Stdlib instance.
func NewStdlib() Stdlib {
	return &stdlib{
		exiter: &realExiter{},
	}
}

// stdlib implements Stdlib.
type stdlib struct {
	exiter exiter
}
