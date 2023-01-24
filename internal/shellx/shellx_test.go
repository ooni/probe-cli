package shellx

import (
	"errors"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
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

// testLogger returns a test logger and a counter incremented
// each time the logger logs at infof level.
func testLogger() (model.Logger, *atomic.Int64) {
	n := &atomic.Int64{}
	log := &mocks.Logger{
		MockInfof: func(format string, v ...interface{}) {
			n.Add(1)
		},
	}
	return log, n
}

func TestRun(t *testing.T) {
	t.Run("with a valid command", func(t *testing.T) {
		log, count := testLogger()
		if err := Run(log, testGolangExe, "version"); err != nil {
			t.Fatal(err)
		}
		if n := count.Load(); n != 1 {
			t.Fatal("expected one log message, got", n)
		}
	})

	t.Run("with an invalid command", func(t *testing.T) {
		log, count := testLogger()
		err := Run(log, "nonexistent", "version")
		if !testErrorIsExecutableNotFound(err) {
			t.Fatal("unexpected error", err)
		}
		if n := count.Load(); n != 0 {
			t.Fatal("expected zero log messages, got", n)
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
		log, count := testLogger()
		err := RunCommandLine(log, `"foobar`)
		if !testErrorIsCannotParseCmdLine(err) {
			t.Fatal("unexpected error", err)
		}
		if n := count.Load(); n != 0 {
			t.Fatal("expected zero log messages, got", n)
		}
	})

	t.Run("when we have no arguments", func(t *testing.T) {
		log, count := testLogger()
		err := RunCommandLine(log, "")
		if !errors.Is(err, ErrNoCommandToExecute) {
			t.Fatal("unexpected error", err)
		}
		if n := count.Load(); n != 0 {
			t.Fatal("expected zero log messages, got", n)
		}
	})

	t.Run("when we have arguments", func(t *testing.T) {
		log, count := testLogger()
		if err := RunCommandLine(log, testGolangExe+" version"); err != nil {
			t.Fatal(err)
		}
		if n := count.Load(); n != 1 {
			t.Fatal("expected one log message, got", n)
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

	t.Run("we verify we can add environment variables", func(t *testing.T) {
		env := &Env{}

		// Add the expected environment variables. The command we're
		// going to run will exit(1) if it cannot find them.
		env.Append("ANTANI", "antani")
		env.Append("MASCETTI", "mascetti")
		env.Append("STUZZICA", "stuzzica")

		t.Run("for OutputQuiet", func(t *testing.T) {
			_, err := env.OutputQuiet(testGolangExe, "run", "./testdata/checkenv.go")
			if err != nil {
				t.Fatal(err)
			}
		})

		t.Run("for Output", func(t *testing.T) {
			_, err := env.Output(model.DiscardLogger, testGolangExe, "run", "./testdata/checkenv.go")
			if err != nil {
				t.Fatal(err)
			}
		})

		t.Run("for RunQuiet", func(t *testing.T) {
			t.Log(env.Vars)
			err := env.RunQuiet(testGolangExe, "run", "./testdata/checkenv.go")
			if err != nil {
				t.Fatal(err)
			}
		})

		t.Run("for Run", func(t *testing.T) {
			err := env.Run(model.DiscardLogger, testGolangExe, "run", "./testdata/checkenv.go")
			if err != nil {
				t.Fatal(err)
			}
		})

		t.Run("for RunCommandLineQuiet", func(t *testing.T) {
			err := env.RunCommandLineQuiet(testGolangExe + " run ./testdata/checkenv.go")
			if err != nil {
				t.Fatal(err)
			}
		})

		t.Run("for RunCommandLine", func(t *testing.T) {
			err := env.RunCommandLine(model.DiscardLogger, testGolangExe+" run ./testdata/checkenv.go")
			if err != nil {
				t.Fatal(err)
			}
		})
	})

	t.Run("Output", func(t *testing.T) {
		t.Run("with a valid command", func(t *testing.T) {
			log, count := testLogger()
			env := &Env{}
			output, err := env.Output(log, testGolangExe, "env")
			if err != nil {
				t.Fatal(err)
			}
			if len(output) <= 0 {
				t.Fatal("expected to see output")
			}
			if n := count.Load(); n != 1 {
				t.Fatal("expected one log message, got", n)
			}
		})

		t.Run("with an invalid command", func(t *testing.T) {
			log, count := testLogger()
			env := &Env{}
			output, err := env.Output(log, "nonexistent", "env")
			if !testErrorIsExecutableNotFound(err) {
				t.Fatal("unexpected error", err)
			}
			if len(output) > 0 {
				t.Fatal("expected to see no output")
			}
			if n := count.Load(); n != 0 {
				t.Fatal("expected zero log messages, got", n)
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

	t.Run("RunQuiet", func(t *testing.T) {
		t.Run("with a valid command", func(t *testing.T) {
			env := &Env{}
			err := env.RunQuiet(testGolangExe, "env")
			if err != nil {
				t.Fatal(err)
			}
		})

		t.Run("with an invalid command", func(t *testing.T) {
			env := &Env{}
			err := env.RunQuiet("nonexistent", "env")
			if !testErrorIsExecutableNotFound(err) {
				t.Fatal("unexpected error", err)
			}
		})
	})

	t.Run("Run", func(t *testing.T) {
		t.Run("with a valid command", func(t *testing.T) {
			env := &Env{}
			log, count := testLogger()
			err := env.Run(log, testGolangExe, "env")
			if err != nil {
				t.Fatal(err)
			}
			if n := count.Load(); n != 1 {
				t.Fatal("expected one log message, got", n)
			}
		})

		t.Run("with an invalid command", func(t *testing.T) {
			env := &Env{}
			log, count := testLogger()
			err := env.Run(log, "nonexistent", "env")
			if !testErrorIsExecutableNotFound(err) {
				t.Fatal("unexpected error", err)
			}
			if n := count.Load(); n != 0 {
				t.Fatal("expected zero log messages, got", n)
			}
		})
	})

	t.Run("RunCommandLine", func(t *testing.T) {
		t.Run("with a valid command", func(t *testing.T) {
			env := &Env{}
			log, count := testLogger()
			err := env.RunCommandLine(log, testGolangExe+" env")
			if err != nil {
				t.Fatal(err)
			}
			if n := count.Load(); n != 1 {
				t.Fatal("expected one log message, got", n)
			}
		})

		t.Run("with an invalid command", func(t *testing.T) {
			env := &Env{}
			log, count := testLogger()
			err := env.RunCommandLine(log, "nonexistent env")
			if !testErrorIsExecutableNotFound(err) {
				t.Fatal("unexpected error", err)
			}
			if n := count.Load(); n != 0 {
				t.Fatal("expected zero log messages, got", n)
			}
		})

		t.Run("with empty command line", func(t *testing.T) {
			env := &Env{}
			log, count := testLogger()
			err := env.RunCommandLine(log, "")
			if !errors.Is(err, ErrNoCommandToExecute) {
				t.Fatal("unexpected error", err)
			}
			if n := count.Load(); n != 0 {
				t.Fatal("expected zero log messages, got", n)
			}
		})

		t.Run("with invalid command line", func(t *testing.T) {
			env := &Env{}
			log, count := testLogger()
			err := env.RunCommandLine(log, "\"foobar")
			if !testErrorIsCannotParseCmdLine(err) {
				t.Fatal("unexpected error", err)
			}
			if n := count.Load(); n != 0 {
				t.Fatal("expected zero log messages, got", n)
			}
		})
	})

	t.Run("RunCommandLineQuiet", func(t *testing.T) {
		t.Run("with a valid command", func(t *testing.T) {
			env := &Env{}
			err := env.RunCommandLineQuiet(testGolangExe + " env")
			if err != nil {
				t.Fatal(err)
			}
		})

		t.Run("with an invalid command", func(t *testing.T) {
			env := &Env{}
			err := env.RunCommandLineQuiet("nonexistent env")
			if !testErrorIsExecutableNotFound(err) {
				t.Fatal("unexpected error", err)
			}
		})
	})

	t.Run("OutputCommandLine", func(t *testing.T) {
		t.Run("with a valid command", func(t *testing.T) {
			env := &Env{}
			log, count := testLogger()
			output, err := env.OutputCommandLine(log, testGolangExe+" env")
			if err != nil {
				t.Fatal(err)
			}
			if len(output) <= 0 {
				t.Fatal("expected to see output")
			}
			if n := count.Load(); n != 1 {
				t.Fatal("expected one log message, got", n)
			}
		})

		t.Run("with an invalid command", func(t *testing.T) {
			env := &Env{}
			log, count := testLogger()
			output, err := env.OutputCommandLine(log, "nonexistent env")
			if !testErrorIsExecutableNotFound(err) {
				t.Fatal("unexpected error", err)
			}
			if len(output) > 0 {
				t.Fatal("expected to see no output")
			}
			if n := count.Load(); n != 0 {
				t.Fatal("expected zero log messages, got", n)
			}
		})

		t.Run("with empty command line", func(t *testing.T) {
			env := &Env{}
			log, count := testLogger()
			output, err := env.OutputCommandLine(log, "")
			if !errors.Is(err, ErrNoCommandToExecute) {
				t.Fatal("unexpected error", err)
			}
			if len(output) > 0 {
				t.Fatal("expected to see no output")
			}
			if n := count.Load(); n != 0 {
				t.Fatal("expected zero log messages, got", n)
			}
		})

		t.Run("with a command line that does not parse", func(t *testing.T) {
			env := &Env{}
			log, count := testLogger()
			output, err := env.OutputCommandLine(log, "\"foobar")
			if !testErrorIsCannotParseCmdLine(err) {
				t.Fatal("unexpected error", err)
			}
			if len(output) > 0 {
				t.Fatal("expected to see no output")
			}
			if n := count.Load(); n != 0 {
				t.Fatal("expected zero log messages, got", n)
			}
		})

		t.Run("with environment variables", func(t *testing.T) {
			env := &Env{}
			log, count := testLogger()
			env.Append("GOCACHE", "/foobar")
			env.Append("FOO", "/foobar")
			output, err := env.OutputCommandLine(log, "go env GOCACHE")
			if err != nil {
				t.Fatal(err)
			}
			expected := []byte("/foobar\n")
			if diff := cmp.Diff(expected, output); diff != "" {
				t.Fatal(diff)
			}
			if n := count.Load(); n != 3 {
				t.Fatal("expected three log messages, got", n)
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
