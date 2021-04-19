package main

import (
	"github.com/ooni/probe-cli/v3/internal/engine/shellx"
)

// mobileCmd contains common code for all mobile commands.
type mobileCmd struct{}

// gomobileInit ensures that we're using the latest version of gomobile.
func (cmd mobileCmd) gomobileInit(flags *GlobalFlags) {
	// Currently we are modifying the go.mod as a result of installing
	// the latest version of `go mobile` - how to avoid that?
	var args []string
	args = append(args, "get")
	args = append(args, "-u")
	if flags.Verbose {
		args = append(args, "-v")
	}
	args = append(args, "golang.org/x/mobile/cmd/gomobile@latest")
	must(shellx.Run("go", args...))
	must(shellx.Run("gomobile", "init"))
}
