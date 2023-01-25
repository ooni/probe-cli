package shellx

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/fsx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
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

func TestVerifyWeCanAppendToArgv(t *testing.T) {
	argv1, err := NewArgv(testGolangExe, "run", "./testdata/checkenv.go")
	if err != nil {
		t.Fatal(err)
	}
	argv2, err := NewArgv(testGolangExe)
	if err != nil {
		t.Fatal(err)
	}
	argv2.Append("run")
	argv2.Append("./testdata/checkenv.go")
	if diff := cmp.Diff(argv1, argv2); diff != "" {
		t.Fatal(diff)
	}
}

func TestVerifyWeAddEnvironmentVariables(t *testing.T) {
	env := &Envp{}

	// Add the expected environment variables. The command we're
	// going to run will exit with nonzero exit code if it cannot find them.
	env.Append("ANTANI", "antani")
	env.Append("MASCETTI", "mascetti")
	env.Append("STUZZICA", "stuzzica")

	argv, err := NewArgv(testGolangExe, "run", "./testdata/checkenv.go")
	if err != nil {
		t.Fatal(err)
	}

	config := &Config{
		Logger: model.DiscardLogger,
		Flags:  FlagShowStdoutStderr,
	}

	t.Run("for OutputEx", func(t *testing.T) {
		out, err := OutputEx(config, argv, env)
		if err != nil {
			t.Fatal(err)
		}
		if len(out) > 0 {
			t.Fatal("expected no output")
		}
	})

	t.Run("for RunEx", func(t *testing.T) {
		if err := RunEx(config, argv, env); err != nil {
			t.Fatal(err)
		}
	})
}

func TestOutput(t *testing.T) {
	t.Run("with a valid command", func(t *testing.T) {
		log, count := testLogger()
		output, err := Output(log, testGolangExe, "env")
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
		output, err := Output(log, "nonexistent", "env")
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
}

func TestOutputQuiet(t *testing.T) {
	t.Run("with a valid command", func(t *testing.T) {
		output, err := OutputQuiet(testGolangExe, "env")
		if err != nil {
			t.Fatal(err)
		}
		if len(output) <= 0 {
			t.Fatal("expected to see output")
		}
	})

	t.Run("with an invalid command", func(t *testing.T) {
		output, err := OutputQuiet("nonexistent", "env")
		if !testErrorIsExecutableNotFound(err) {
			t.Fatal("unexpected error", err)
		}
		if len(output) > 0 {
			t.Fatal("expected to see no output")
		}
	})
}

func TestRunQuiet(t *testing.T) {
	t.Run("with a valid command", func(t *testing.T) {
		err := RunQuiet(testGolangExe, "env")
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("with an invalid command", func(t *testing.T) {
		err := RunQuiet("nonexistent", "env")
		if !testErrorIsExecutableNotFound(err) {
			t.Fatal("unexpected error", err)
		}
	})
}

func TestRun(t *testing.T) {
	t.Run("with a valid command", func(t *testing.T) {
		log, count := testLogger()
		err := Run(log, testGolangExe, "env")
		if err != nil {
			t.Fatal(err)
		}
		if n := count.Load(); n != 1 {
			t.Fatal("expected one log message, got", n)
		}
	})

	t.Run("with an invalid command", func(t *testing.T) {
		log, count := testLogger()
		err := Run(log, "nonexistent", "env")
		if !testErrorIsExecutableNotFound(err) {
			t.Fatal("unexpected error", err)
		}
		if n := count.Load(); n != 0 {
			t.Fatal("expected zero log messages, got", n)
		}
	})
}

func TestRunCommandLine(t *testing.T) {
	t.Run("with a valid command line", func(t *testing.T) {
		log, count := testLogger()
		err := RunCommandLine(log, testGolangExe+" env")
		if err != nil {
			t.Fatal(err)
		}
		if n := count.Load(); n != 1 {
			t.Fatal("expected one log message, got", n)
		}
	})

	t.Run("with an invalid command line", func(t *testing.T) {
		log, count := testLogger()
		err := RunCommandLine(log, "nonexistent env")
		if !testErrorIsExecutableNotFound(err) {
			t.Fatal("unexpected error", err)
		}
		if n := count.Load(); n != 0 {
			t.Fatal("expected zero log messages, got", n)
		}
	})

	t.Run("with empty command line", func(t *testing.T) {
		log, count := testLogger()
		err := RunCommandLine(log, "")
		if !errors.Is(err, ErrNoCommandToExecute) {
			t.Fatal("unexpected error", err)
		}
		if n := count.Load(); n != 0 {
			t.Fatal("expected zero log messages, got", n)
		}
	})

	t.Run("with a command line that does not parse", func(t *testing.T) {
		log, count := testLogger()
		err := RunCommandLine(log, "\"foobar")
		if !testErrorIsCannotParseCmdLine(err) {
			t.Fatal("unexpected error", err)
		}
		if n := count.Load(); n != 0 {
			t.Fatal("expected zero log messages, got", n)
		}
	})

func TestRunCommandLineQuiet(t *testing.T) {
	t.Run("with a valid command line", func(t *testing.T) {
		err := RunCommandLineQuiet(testGolangExe + " env")
		if err != nil {
			t.Fatal(err)
		}
	})

	t.Run("with an invalid command line", func(t *testing.T) {
		err := RunCommandLineQuiet("nonexistent env")
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
		err := RunCommandLineQuiet("")
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
		err := RunCommandLineQuiet("\"foobar")
		if !testErrorIsCannotParseCmdLine(err) {
			t.Fatal("unexpected error", err)
		}
	})
}

func TestOutputCommandLine(t *testing.T) {
	t.Run("with a valid command line", func(t *testing.T) {
		log, count := testLogger()
		output, err := OutputCommandLine(log, testGolangExe+" env")
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

	t.Run("with an invalid command line", func(t *testing.T) {
		log, count := testLogger()
		output, err := OutputCommandLine(log, "nonexistent env")
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
		log, count := testLogger()
		output, err := OutputCommandLine(log, "")
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
		log, count := testLogger()
		output, err := OutputCommandLine(log, "\"foobar")
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
}

func TestOutputCommandLineQuiet(t *testing.T) {
	t.Run("with a valid command", func(t *testing.T) {
		output, err := OutputCommandLineQuiet(testGolangExe + " env")
		if err != nil {
			t.Fatal(err)
		}
		if len(output) <= 0 {
			t.Fatal("expected to see output")
		}
	})

	t.Run("with an invalid command", func(t *testing.T) {
		output, err := OutputCommandLineQuiet("nonexistent env")
		if !testErrorIsExecutableNotFound(err) {
			t.Fatal("unexpected error", err)
		}
		if len(output) > 0 {
			t.Fatal("expected to see no output")
		}
	})

	t.Run("with empty command line", func(t *testing.T) {
		output, err := OutputCommandLineQuiet("")
		if !errors.Is(err, ErrNoCommandToExecute) {
			t.Fatal("unexpected error", err)
		}
		if len(output) > 0 {
			t.Fatal("expected to see no output")
		}
	})

	t.Run("with a command line that does not parse", func(t *testing.T) {
		output, err := OutputCommandLineQuiet("\"foobar")
		if !testErrorIsCannotParseCmdLine(err) {
			t.Fatal("unexpected error", err)
		}
		if len(output) > 0 {
			t.Fatal("expected to see no output")
		}
	})
}

func Test_maybeQuoteArgUnsafe(t *testing.T) {
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
			if got := maybeQuoteArgUnsafe(tt.args.a); got != tt.want {
				t.Errorf("maybeQuoteArg() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCopyFile(t *testing.T) {
	t.Run("in case of success", func(t *testing.T) {
		source := filepath.Join("testdata", "checkenv.go")
		expected, err := os.ReadFile(source)
		if err != nil {
			t.Fatal(err)
		}
		dest := filepath.Join("testdata", "copy.txt")
		defer os.Remove(dest)
		if err := CopyFile(source, dest, 0600); err != nil {
			t.Fatal(err)
		}
		got, err := os.ReadFile(dest)
		if err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(expected, got); diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("if we cannot open the source file", func(t *testing.T) {
		source := filepath.Join("testdata", "checkenv.go")
		dest := filepath.Join("testdata", "copy.txt")
		defer os.Remove(dest)
		expected := errors.New("mocked error")
		fsxOpenFile = func(pathname string) (fs.File, error) {
			return nil, expected
		}
		defer func() {
			fsxOpenFile = fsx.OpenFile
		}()
		if err := CopyFile(source, dest, 0600); !errors.Is(err, expected) {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("if we cannot open the dest file", func(t *testing.T) {
		source := filepath.Join("testdata", "checkenv.go")
		dest := filepath.Join("testdata", "copy.txt")
		defer os.Remove(dest)
		expected := errors.New("mocked error")
		osOpenFile = func(name string, flag int, perm os.FileMode) (*os.File, error) {
			return nil, expected
		}
		defer func() {
			osOpenFile = os.OpenFile
		}()
		if err := CopyFile(source, dest, 0600); !errors.Is(err, expected) {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("if we cannot copy", func(t *testing.T) {
		source := filepath.Join("testdata", "checkenv.go")
		dest := filepath.Join("testdata", "copy.txt")
		defer os.Remove(dest)
		expected := errors.New("mocked error")
		netxliteCopyContext = func(ctx context.Context, dst io.Writer, src io.Reader) (int64, error) {
			return 0, expected
		}
		defer func() {
			netxliteCopyContext = netxlite.CopyContext
		}()
		if err := CopyFile(source, dest, 0600); !errors.Is(err, expected) {
			t.Fatal("unexpected error", err)
		}
	})
}
