package app

import (
	"os"

	ooni "github.com/ooni/probe-cli"
	"github.com/ooni/probe-cli/internal/cli/root"
)

// Run the app. This is the main app entry point
func Run() error {
	root.Cmd.Version(ooni.Version)
	_, err := root.Cmd.Parse(os.Args[1:])
	return err
}
