package buildtooltest

import (
	"testing"

	"golang.org/x/sys/execabs"
)

func TestTestCheckManyCommands(t *testing.T) {

	type testcase struct {
		name      string
		cmd       []*execabs.Cmd
		tee       []ExecExpectations
		expectErr bool
	}

	var testcases = []testcase{{
		name: "where everything is working as intended",
		cmd: []*execabs.Cmd{{
			Path: "/usr/local/bin/go",
			Args: []string{"go", "version"},
		}, {
			Path: "/usr/local/bin/go",
			Args: []string{"go", "env", "GOPATH"},
			Env: []string{
				"GOPATH=/tmp",
			},
		}},
		tee: []ExecExpectations{{
			Env:  []string{},
			Argv: []string{"go", "version"},
		}, {
			Env:  []string{"GOPATH=/tmp"},
			Argv: []string{"go", "env", "GOPATH"},
		}},
		expectErr: false,
	}, {
		name: "where the issue is with the environment",
		cmd: []*execabs.Cmd{{
			Path: "/usr/local/bin/go",
			Args: []string{"go", "version"},
		}},
		tee: []ExecExpectations{{
			Env:  []string{"GOPATH=/tmp"},
			Argv: []string{"go", "version"},
		}},
		expectErr: true,
	}, {
		name: "where the issue is with the argv",
		cmd: []*execabs.Cmd{{
			Path: "/usr/local/bin/go",
			Args: []string{"go", "version"},
		}},
		tee: []ExecExpectations{{
			Argv: []string{"go", "env"},
		}},
		expectErr: true,
	}}

	for _, c := range testcases {
		t.Run(c.name, func(t *testing.T) {
			err := CheckManyCommands(c.cmd, c.tee)
			if err != nil && !c.expectErr {
				t.Fatal("did not expect an error", err)
			}
			if err == nil && c.expectErr {
				t.Fatal("expected error but got nil")
			}
		})
	}
}

func TestTestSimpleCommandCollector(t *testing.T) {
	t.Run("LookPath", func(t *testing.T) {
		cc := &SimpleCommandCollector{}
		path, err := cc.LookPath("go")
		if err != nil {
			t.Fatal(err)
		}
		if path != "go" {
			t.Fatal("invalid path")
		}
	})

	t.Run("CmdOutput", func(t *testing.T) {
		cc := &SimpleCommandCollector{}
		cmd := &execabs.Cmd{}
		output, err := cc.CmdOutput(cmd)
		if err != nil {
			t.Fatal(err)
		}
		if len(output) != 0 {
			t.Fatal("invalid output")
		}
		if cc.Commands[0] != cmd {
			t.Fatal("did not save the command")
		}
	})

	t.Run("CmdRun", func(t *testing.T) {
		cc := &SimpleCommandCollector{}
		cmd := &execabs.Cmd{}
		if err := cc.CmdRun(cmd); err != nil {
			t.Fatal(err)
		}
		if cc.Commands[0] != cmd {
			t.Fatal("did not save the command")
		}
	})
}

func TestTestDependenciesCallCounter(t *testing.T) {
	t.Run("golangCheck", func(t *testing.T) {})

	t.Run("linuxReadGOVERSION", func(t *testing.T) {})

	t.Run("linuxWriteDOCKEFILE", func(t *testing.T) {})

	t.Run("psiphonFileExists", func(t *testing.T) {})

	t.Run("psiphonMaybeCopyConfigFiles", func(t *testing.T) {})

	t.Run("windowsMingwCheck", func(t *testing.T) {})
}
