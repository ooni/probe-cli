package shellx

import (
	"errors"
	"runtime"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// testGolangExe is the golang exe to use in this test suite
var testGolangExe string

func init() {
	switch runtime.GOOS {
	case "windows":
		testGolangExe = "go.exe"
	default:
		testGolangExe = "go"
	}
}

// testErrorIsExecutableNotFound returns whether the error
// is the one returned when an executable isn't found
func testErrorIsExecutableNotFound(err error) bool {
	return err != nil && strings.Contains(err.Error(), "executable file not found")
}

// testErrorIsCannotParseCmdLine returns whether the error
// is the one returned when you cannot parse a cmdline.
func testErrorIsCannotParseCmdLine(err error) bool {
	return err != nil && err.Error() == "EOF found when expecting closing quote"
}

func TestRun(t *testing.T) {
	t.Run("with a valid command", func(t *testing.T) {
		if err := Run(model.DiscardLogger, testGolangExe, "version"); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("with an invalid command", func(t *testing.T) {
		err := Run(model.DiscardLogger, "nonexistent", "version")
		if !testErrorIsExecutableNotFound(err) {
			t.Fatal("unexpected error", err)
		}
	})
}

func TestRunQuiet(t *testing.T) {
	t.Run("with a valid command", func(t *testing.T) {
		if err := RunQuiet(testGolangExe, "version"); err != nil {
			t.Fatal(err)
		}
	})

	t.Run("with an invalid command", func(t *testing.T) {
		err := RunQuiet("nonexistent", "version")
		if !testErrorIsExecutableNotFound(err) {
			t.Fatal("unexpected error", err)
		}
	})
}

func TestRunCommandline(t *testing.T) {
	t.Run("when the command does not parse", func(t *testing.T) {
		err := RunCommandLine(model.DiscardLogger, `"foobar`)
		if !testErrorIsCannotParseCmdLine(err) {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("when we have no arguments", func(t *testing.T) {
		err := RunCommandLine(model.DiscardLogger, "")
		if !errors.Is(err, ErrNoCommandToExecute) {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("when we have arguments", func(t *testing.T) {
		if err := RunCommandLine(model.DiscardLogger, testGolangExe+" version"); err != nil {
			t.Fatal(err)
		}
	})
}

func TestRunCommandlineQuiet(t *testing.T) {
	t.Run("when the command does not parse", func(t *testing.T) {
		err := RunCommandLineQuiet(`"foobar`)
		if !testErrorIsCannotParseCmdLine(err) {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("when we have no arguments", func(t *testing.T) {
		err := RunCommandLineQuiet("")
		if !errors.Is(err, ErrNoCommandToExecute) {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("when we have arguments", func(t *testing.T) {
		if err := RunCommandLineQuiet(testGolangExe + " version"); err != nil {
			t.Fatal(err)
		}
	})
}

func TestEnv(t *testing.T) {
	t.Run("Output", func(t *testing.T) {
		t.Run("with a valid command", func(t *testing.T) {
			env := &Env{}
			output, err := env.Output(model.DiscardLogger, testGolangExe, "env")
			if err != nil {
				t.Fatal(err)
			}
			if len(output) <= 0 {
				t.Fatal("expected to see output")
			}
		})

		t.Run("with an invalid command", func(t *testing.T) {
			env := &Env{}
			output, err := env.Output(model.DiscardLogger, "nonexistent", "env")
			if !testErrorIsExecutableNotFound(err) {
				t.Fatal("unexpected error", err)
			}
			if len(output) > 0 {
				t.Fatal("expected to see no output")
			}
		})
	})

	t.Run("OutputQuiet", func(t *testing.T) {
		t.Run("with a valid command", func(t *testing.T) {
			env := &Env{}
			output, err := env.OutputQuiet(testGolangExe, "env")
			if err != nil {
				t.Fatal(err)
			}
			if len(output) <= 0 {
				t.Fatal("expected to see output")
			}
		})

		t.Run("with an invalid command", func(t *testing.T) {
			env := &Env{}
			output, err := env.OutputQuiet("nonexistent", "env")
			if !testErrorIsExecutableNotFound(err) {
				t.Fatal("unexpected error", err)
			}
			if len(output) > 0 {
				t.Fatal("expected to see no output")
			}
		})
	})

	t.Run("OutputCommandLine", func(t *testing.T) {
		t.Run("with a valid command", func(t *testing.T) {
			env := &Env{}
			output, err := env.OutputCommandLine(model.DiscardLogger, testGolangExe+" env")
			if err != nil {
				t.Fatal(err)
			}
			if len(output) <= 0 {
				t.Fatal("expected to see output")
			}
		})

		t.Run("with an invalid command", func(t *testing.T) {
			env := &Env{}
			output, err := env.OutputCommandLine(model.DiscardLogger, "nonexistent env")
			if !testErrorIsExecutableNotFound(err) {
				t.Fatal("unexpected error", err)
			}
			if len(output) > 0 {
				t.Fatal("expected to see no output")
			}
		})

		t.Run("with empty command line", func(t *testing.T) {
			env := &Env{}
			output, err := env.OutputCommandLine(model.DiscardLogger, "")
			if !errors.Is(err, ErrNoCommandToExecute) {
				t.Fatal("unexpected error", err)
			}
			if len(output) > 0 {
				t.Fatal("expected to see no output")
			}
		})

		t.Run("with invalid command line", func(t *testing.T) {
			env := &Env{}
			output, err := env.OutputCommandLine(model.DiscardLogger, "\"foobar")
			if !testErrorIsCannotParseCmdLine(err) {
				t.Fatal("unexpected error", err)
			}
			if len(output) > 0 {
				t.Fatal("expected to see no output")
			}
		})

		t.Run("with environment variables", func(t *testing.T) {
			env := &Env{}
			env.Append("GOCACHE", "/foobar")
			output, err := env.OutputCommandLine(model.DiscardLogger, "go env GOCACHE")
			if err != nil {
				t.Fatal(err)
			}
			expected := []byte("/foobar\n")
			if diff := cmp.Diff(expected, output); diff != "" {
				t.Fatal(diff)
			}
		})
	})

	t.Run("OutputCommandLineQuiet", func(t *testing.T) {
		t.Run("with a valid command", func(t *testing.T) {
			env := &Env{}
			output, err := env.OutputCommandLineQuiet(testGolangExe + " env")
			if err != nil {
				t.Fatal(err)
			}
			if len(output) <= 0 {
				t.Fatal("expected to see output")
			}
		})

		t.Run("with an invalid command", func(t *testing.T) {
			env := &Env{}
			output, err := env.OutputCommandLineQuiet("nonexistent env")
			if !testErrorIsExecutableNotFound(err) {
				t.Fatal("unexpected error", err)
			}
			if len(output) > 0 {
				t.Fatal("expected to see no output")
			}
		})

		t.Run("with environment variables", func(t *testing.T) {
			env := &Env{}
			env.Append("GOCACHE", "/foobar")
			output, err := env.OutputCommandLineQuiet("go env GOCACHE")
			if err != nil {
				t.Fatal(err)
			}
			expected := []byte("/foobar\n")
			if diff := cmp.Diff(expected, output); diff != "" {
				t.Fatal(diff)
			}
		})
	})
}

func Test_maybeQuoteArg(t *testing.T) {
	type args struct {
		a string
	}
	tests := []struct {
		name string
		args args
		want string
	}{{
		name: "for empty string",
		args: args{},
		want: "",
	}, {
		name: "without spaces",
		args: args{
			a: "helloworld",
		},
		want: "helloworld",
	}, {
		name: "with spaces",
		args: args{
			a: "hello world",
		},
		want: "\"hello world\"",
	}, {
		name: "with quotes",
		args: args{
			a: "hello\"world",
		},
		want: "hello\\\"world",
	}, {
		name: "with quotes and spaces",
		args: args{
			a: "hello \" world",
		},
		want: "\"hello \\\" world\"",
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := maybeQuoteArg(tt.args.a); got != tt.want {
				t.Errorf("maybeQuoteArg() = %v, want %v", got, tt.want)
			}
		})
	}
}
