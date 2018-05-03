package app

import (
	"os"

	"github.com/ooni/probe-cli/internal/cli/root"
	"github.com/ooni/probe-cli/internal/cli/version"
)

// Run the app. This is the main app entry point
func Run() error {
	root.Cmd.Version(version.Version)
	_, err := root.Cmd.Parse(os.Args[1:])
	return err
}
