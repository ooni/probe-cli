package main

import (
	"fmt"
	"os"
)

// Command is the common interface of commands.
type Command interface {
	// Main is the main function implementing the command.
	Main(args []string)

	// BriefDescription returns a brief description of the command.
	BriefDescription() string

	// Help returns the help string for the command.
	Help() string
}

// HelpCmd is the help command.
type HelpCmd struct{}

// Main is the main of the `jafar2 help` command.
func (cmd *HelpCmd) Main(args []string) {
	if len(args) < 2 {
		usage(os.Stdout)
		os.Exit(0)
	}
	if len(args) > 2 {
		fmt.Fprintf(os.Stderr, "%s\n", cmd.Help())
		os.Exit(1)
	}
	if cmd := Commands[args[1]]; cmd != nil {
		fmt.Printf("%s\n", cmd.Help())
		os.Exit(0)
	}
	fmt.Fprintf(os.Stderr, "jafar2 help: no such command: '%s'.\n", args[0])
	fmt.Fprint(os.Stderr, "Use `jafar2 help` for more comprehensive help.\n")
}

// BriefDescription returns a brief description of the command.
func (cmd *HelpCmd) BriefDescription() string {
	return "provides information about commands"
}

// Help returns the help string for the command.
func (cmd *HelpCmd) Help() string {
	return `
usage: jafar2 help [command]

If no command is specified, prints the general usage message for
jafar2. Otherwise, prints the usage message for <command>.
`
}

// Commands maps a command name to a command.
var Commands = map[string]Command{
	"create":  &CreateCmd{},
	"destroy": &DestroyCmd{},
	"help":    &HelpCmd{},
	"run":     &RunCmd{},
}

// usage prints the program usage.
func usage(fp *os.File) {
	fmt.Fprintf(fp, "usage: jafar2 command [options...]\n\n")
	fmt.Fprintf(fp, "This program emulates network censorship, latency, and losses.\n\n")
	fmt.Fprintf(fp, "Available commands:\n\n")
	for name, command := range Commands {
		fmt.Fprintf(fp, "- %s: %s.\n\n", name, command.BriefDescription())
	}
	fmt.Fprintf(fp, "Use `jafar2 help <command>` to get help about a command.\n\n")
}

func main() {
	if len(os.Args) < 2 {
		usage(os.Stdout)
		os.Exit(0)
	}
	command := os.Args[1]
	if command == "-h" || command == "--help" { // be extra friendly
		usage(os.Stdout)
		os.Exit(0)
	}
	if cmd := Commands[command]; cmd != nil {
		cmd.Main(os.Args[1:])
		os.Exit(0)
	}
	usage(os.Stderr)
	os.Exit(1)
}
