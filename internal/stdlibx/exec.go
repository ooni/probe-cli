package stdlibx

//
// Wrappers for os/exec
//

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
)

// MustNewCommand implements Stdlib.
func (sp *stdlib) MustNewCommand(binpath string, args ...string) Command {
	sp.MustFprintf(os.Stderr, "checking for %s...", binpath)
	p, err := exec.LookPath(binpath)
	sp.ExitOnError(err, "exec.LookPath")
	sp.MustFprintf(os.Stderr, "%s\n", p)
	return &commandWrapper{
		cmd: &exec.Cmd{
			Args:   append([]string{p}, args...),
			Env:    os.Environ(),
			Path:   p,
			Stdout: os.Stdout,
			Stderr: os.Stderr,
		},
		sp: sp,
	}
}

// commandWrapper wraps exec.Cmd.
type commandWrapper struct {
	cmd *exec.Cmd
	sp  *stdlib
}

// AddArgs implements Command.
func (cmd *commandWrapper) AddArgs(arg ...string) {
	cmd.cmd.Args = append(cmd.cmd.Args, arg...)
}

// AddEnv implements Command.
func (cmd *commandWrapper) AddEnv(key, value string) {
	v := fmt.Sprintf("%s=%s", key, value)
	cmd.sp.MustFprintf(os.Stderr, "+ export %s\n", v)
	cmd.cmd.Env = append(cmd.cmd.Env, v)
}

// Run implements Command.
func (cmd *commandWrapper) Run() error {
	cmd.logExecution()
	return cmd.cmd.Run()
}

// MustRun implements Command.
func (cmd *commandWrapper) MustRun() {
	err := cmd.Run()
	cmd.sp.ExitOnError(err, "cc.Run")
}

// SetStdout implements Command.
func (cmd *commandWrapper) SetStdout(w io.Writer) {
	cmd.cmd.Stdout = w
}

// SetStderr implements Command.
func (cmd *commandWrapper) SetStderr(w io.Writer) {
	cmd.cmd.Stderr = w
}

// logExecution logs the command's execution.
func (cmd *commandWrapper) logExecution() {
	argv := []string{}
	argv = append(argv, cmd.maybeQuote(cmd.cmd.Path))
	for _, a := range cmd.cmd.Args[1:] {
		argv = append(argv, cmd.maybeQuote(a))
	}
	cmd.sp.MustFprintf(os.Stderr, "+ %s\n", strings.Join(argv, " "))
}

// maybeQuote attempts to quote the given argument.
func (cmd *commandWrapper) maybeQuote(a string) string {
	if strings.Contains(a, "\"") {
		a = strings.ReplaceAll(a, "\"", "\\\"")
	}
	if strings.Contains(a, " ") {
		a = "\"" + a + "\""
	}
	return a
}

// Run implements Stdlib.
func (sp *stdlib) Run(binpath string, args ...string) error {
	return sp.MustNewCommand(binpath, args...).Run()
}

// MustRun implements Stdlib.
func (sp *stdlib) MustRun(binpath string, args ...string) {
	sp.MustNewCommand(binpath, args...).MustRun()
}

// MustRunAndReadFirstLine implements Stdlib.
func (sp *stdlib) MustRunAndReadFirstLine(binpath string, args ...string) string {
	cmd := exec.Command(binpath, args...)
	data, err := cmd.Output()
	sp.ExitOnError(err, "cmd.Output")
	return mustReadFirstLine(data)
}
