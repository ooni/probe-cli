package stdlibx

import (
	"errors"
	"runtime"
	"strings"
	"testing"
)

func TestMustRun(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping test when not on Unix")
	}

	t.Run("for successful command", func(t *testing.T) {
		stdlib := NewStdlib()
		stdlib.MustRun("go", "version")
	})

	t.Run("for nonexisting command", func(t *testing.T) {
		expected := errors.New("mocked error")
		var got error
		func() {
			defer func() {
				if r := recover(); r != nil {
					got = r.(error)
				}
			}()
			stdlib := NewStdlib().(*stdlib)
			stdlib.exiter = &testExiter{
				err: expected,
			}
			stdlib.MustRun("nonexistent", "version")
		}()
		if expected != got {
			t.Fatal("did not call exit")
		}
	})
}

func TestMustRunAndReadFirstLine(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("skipping test when not on Unix")
	}

	t.Run("for successful command", func(t *testing.T) {
		stdlib := NewStdlib()
		firstline := stdlib.MustRunAndReadFirstLine("go", "version")
		v := strings.Split(firstline, " ")
		if len(v) != 4 {
			t.Fatal("expected exactly four tokens")
		}
		if v[0] != "go" && v[1] != "version" {
			t.Fatal("unexpected value for the first two tokens")
		}
	})

	t.Run("for nonexisting command", func(t *testing.T) {
		expected := errors.New("mocked error")
		var got error
		func() {
			defer func() {
				if r := recover(); r != nil {
					got = r.(error)
				}
			}()
			stdlib := NewStdlib().(*stdlib)
			stdlib.exiter = &testExiter{
				err: expected,
			}
			_ = stdlib.MustRunAndReadFirstLine("nonexistent", "version")
		}()
		if expected != got {
			t.Fatal("did not call exit")
		}
	})
}

func TestExecWeCanSetEnvironmentVariables(t *testing.T) {
	stdlib := NewStdlib()
	cmd := stdlib.MustNewCommand("go", "env", "GOCACHE")
	cmd.AddEnv("GOCACHE", "/antani")
}
