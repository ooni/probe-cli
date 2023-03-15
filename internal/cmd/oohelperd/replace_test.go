package main

import (
	"io/fs"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestReplaceRunningInstance(t *testing.T) {
	// create the default dependencies
	deps := newReplaceDeps()

	// override the CopyFile func
	var (
		asource string
		adest   string
		aperms  fs.FileMode
	)
	deps.CopyFile = func(source, dest string, perms fs.FileMode) error {
		asource = source
		adest = dest
		aperms = perms
		return nil
	}

	// override the Run func
	var commands [][]string
	deps.Run = func(logger model.Logger, command string, args ...string) error {
		argv := []string{command}
		argv = append(argv, args...)
		commands = append(commands, argv)
		return nil
	}

	// execute with fake deps
	replaceRunningInstance(deps)

	// make sure we copied with the correct arguments
	if asource == "" {
		t.Fatal("expected to see valid asource")
	}
	if adest != string(filepath.Separator)+filepath.Join("usr", "bin", "oohelperd") {
		t.Fatal("invalid adest", adest)
	}
	if aperms != 0755 {
		t.Fatal("invalid aperms", aperms)
	}

	// expectCommands contains the expected commands.
	expectCommands := [][]string{{
		"systemctl", "stop", "oohelperd.service",
	}, {
		"systemctl", "start", "oohelperd.service",
	}}

	// make sure we've seen the expected commands
	if diff := cmp.Diff(expectCommands, commands); diff != "" {
		t.Fatal(diff)
	}
}
