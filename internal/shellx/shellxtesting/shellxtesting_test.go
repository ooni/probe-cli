package shellxtesting

import (
	"errors"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/shellx"
	"golang.org/x/sys/execabs"
)

func TestCmdOutput(t *testing.T) {
	expected := errors.New("mocked error")
	lib := &Library{
		MockCmdOutput: func(c *execabs.Cmd) ([]byte, error) {
			return nil, expected
		},
	}
	data, err := lib.CmdOutput(&execabs.Cmd{})
	if !errors.Is(err, expected) {
		t.Fatal("unexpected error", err)
	}
	if len(data) != 0 {
		t.Fatal("expected zero-length data")
	}
}

func TestCmdRun(t *testing.T) {
	expected := errors.New("mocked error")
	lib := &Library{
		MockCmdRun: func(c *execabs.Cmd) error {
			return expected
		},
	}
	err := lib.CmdRun(&execabs.Cmd{})
	if !errors.Is(err, expected) {
		t.Fatal("unexpected error", err)
	}
}

func TestLookPath(t *testing.T) {
	expected := errors.New("mocked error")
	lib := &Library{
		MockLookPath: func(file string) (string, error) {
			return "", expected
		},
	}
	binary, err := lib.LookPath("go")
	if !errors.Is(err, expected) {
		t.Fatal("unexpected error", err)
	}
	if len(binary) != 0 {
		t.Fatal("expected zero-length string")
	}
}

func TestMustArgv(t *testing.T) {
	cmd := &execabs.Cmd{
		Path: "/usr/bin/go",
		Args: []string{"go", "env", "GOPATH"},
	}
	argv := MustArgv(cmd)
	expected := []string{"/usr/bin/go", "env", "GOPATH"}
	if diff := cmp.Diff(expected, argv); diff != "" {
		t.Fatal(diff)
	}
}

func TestWithCustomLibrary(t *testing.T) {
	expected := errors.New("mocked error")
	library := &Library{
		MockCmdRun: func(c *execabs.Cmd) error {
			return expected
		},
		MockLookPath: func(file string) (string, error) {
			return "go", nil
		},
	}
	var err error
	WithCustomLibrary(library, func() {
		err = shellx.RunQuiet("go", "version")
	})
	if !errors.Is(err, expected) {
		t.Fatal("unexpected error", err)
	}
}

func TestRemoveCommonEnvironmentVariables(t *testing.T) {
	cmd := &execabs.Cmd{
		Env: os.Environ(),
	}
	expected := map[string]bool{
		"ANTANI=1":        true,
		"MASCETTI=10":     true,
		"FOO=55":          true,
		"PATH=/bin":       true,
		"HOME=/var/empty": true,
	}
	for key := range expected {
		cmd.Env = append(cmd.Env, key)
	}
	got := RemoveCommonEnvironmentVariables(cmd)
	m := map[string]bool{}
	for _, entry := range got {
		m[entry] = true
	}
	if diff := cmp.Diff(expected, m); diff != "" {
		t.Fatal(diff)
	}
}
