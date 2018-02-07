package app

import (
	"os"

	"github.com/openobservatory/gooni/internal/cli/root"
	"github.com/openobservatory/gooni/internal/cli/version"
	"github.com/openobservatory/gooni/internal/util"
)

// Run the app. This is the main app entry point
func Run() error {
	util.Log("Running")
	root.Cmd.Version(version.Version)
	_, err := root.Cmd.Parse(os.Args[1:])
	return err
}
