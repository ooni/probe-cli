package mockable

import (
	"io"

	"github.com/ooni/probe-cli/v3/internal/cmd/buildtool/internal/stdlibx"
)

// Command mocks stdlibx.Command.
type Command struct {
	MockAddArgs func(args ...string)

	MockAddEnv func(key string, value string)

	MockMustRun func()

	MockRun func() error

	MockSetStderr func(w io.Writer)

	MockSetStdout func(w io.Writer)
}

var _ stdlibx.Command = &Command{}

// AddArgs implements stdlibx.Command
func (c *Command) AddArgs(args ...string) {
	c.MockAddArgs(args...)
}

// AddEnv implements stdlibx.Command
func (c *Command) AddEnv(key string, value string) {
	c.MockAddEnv(key, value)
}

// MustRun implements stdlibx.Command
func (c *Command) MustRun() {
	c.MockMustRun()
}

// Run implements stdlibx.Command
func (c *Command) Run() error {
	return c.MockRun()
}

// SetStderr implements stdlibx.Command
func (c *Command) SetStderr(w io.Writer) {
	c.MockSetStderr(w)
}

// SetStdout implements stdlibx.Command
func (c *Command) SetStdout(w io.Writer) {
	c.MockSetStdout(w)
}
