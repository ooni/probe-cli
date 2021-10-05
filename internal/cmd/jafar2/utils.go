package main

import (
	"os"

	"github.com/alessio/shellescape"
	"github.com/apex/log"
	"github.com/google/shlex"
	"golang.org/x/sys/execabs"
)

// NewCommandWithStdio calls execabs.Command and sets its Stdin, Stdout,
// and Stderr to point to os.Stdin, os.Stdout, os.Stderr.
func NewCommandWithStdio(command string, args ...string) *execabs.Cmd {
	cmd := execabs.Command(command, args...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	return cmd
}

// FatalOnError emits an error and then calls panic. We use panic rather
// than log.Fatal because this allows to run the registered cleanups.
func FatalOnError(err error, message string) {
	if err != nil {
		log.Errorf("%s: %s", message, err.Error())
		panic(message)
	}
}

// QuoteShellArgs quotes arguments for the shell.
func QuoteShellArgs(args []string) string {
	return shellescape.QuoteCommand(args)
}

// SplitShellArgs splits arguments for the shell.
func SplitShellArgs(command string) []string {
	args, err := shlex.Split(command)
	FatalOnError(err, "cannot split shell command")
	return args
}

// FatalOnPanic calls os.Exit(1) if the current function
// terminates with a panic rather than normally.
func FatalOnPanic() {
	if recover() != nil {
		os.Exit(1)
	}
}
