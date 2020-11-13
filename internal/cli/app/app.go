package app

import (
	"os"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/internal/cli/root"
	"github.com/ooni/probe-cli/internal/version"
)

// Run the app. This is the main app entry point
func Run() {
	root.Cmd.Version(version.Version)
	_, err := root.Cmd.Parse(os.Args[1:])
	if err != nil {
		log.WithError(err).Error("failure in main command")
		os.Exit(2)
	}
	return
}
