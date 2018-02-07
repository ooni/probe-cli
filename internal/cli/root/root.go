package root

import (
	"github.com/alecthomas/kingpin"
	"github.com/openobservatory/gooni/internal/util"
)

// Cmd is the root command
var Cmd = kingpin.New("ooni", "")

// Command is syntax sugar for defining sub-commands
var Command = Cmd.Command

func init() {
	Cmd.PreAction(func(ctx *kingpin.ParseContext) error {
		util.Log("Running pre-action")
		return nil
	})
}
