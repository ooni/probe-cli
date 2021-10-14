package main

import (
	"fmt"
	"os"

	"github.com/pborman/getopt/v2"
)

// RunCmd implements the `jafar2 run` command.
type RunCmd struct{}

// Help returns the command help.
func (cmd *RunCmd) Help() string {
	return makeHelp(cmd, cmd.newGetoptParser(NewEnvironment()))
}

// BriefDescription returns a brief description of the command.
func (cmd *RunCmd) BriefDescription() string {
	return "runs a command in a previously-created namespace"
}

// Main is the main of the `jafar2 run` command.
func (cmd *RunCmd) Main(args []string) {
	env := NewEnvironment()
	getopt := cmd.newGetoptParser(env)
	getopt.Parse(args)
	if len(getopt.Args()) < 1 {
		fmt.Fprintf(os.Stderr, "jafar2 run: missing command to run.\n")
		fmt.Fprintf(os.Stderr, "Run `jafar2 help run` for more help.\n")
		os.Exit(1)
	}
	fatalOnError(cmd.run(env, getopt.Args(), NewShell(env)))
}

// newGetoptParser returns the getopt parser for the create command.
func (cmd *RunCmd) newGetoptParser(env *Environment) *getopt.Set {
	getopt := getopt.New()
	getopt.SetProgram("jafar2 run")
	getopt.SetParameters(" <command> [arguments...]")
	getopt.FlagLong(&env.DryRun, "dry-run", 'n', "show what would have been done")
	getopt.FlagLong(&env.NamespaceName, "namespace-name", 0,
		"name of the previously-created namespace")
	return getopt
}

// Run runs the create command.
func (cmd *RunCmd) run(env *Environment, args []string, sh Shell) error {
	if err := env.Validate(); err != nil {
		return err
	}
	arguments := []string{"ip", "netns", "exec", env.NamespaceName}
	arguments = append(arguments, args...)
	return sh.Runv(arguments)
}
