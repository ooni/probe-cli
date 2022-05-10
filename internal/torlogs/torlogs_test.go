package torlogs

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
)

func TestReadBootstrapLogs(t *testing.T) {
	t.Run("with empty file path", func(t *testing.T) {
		out, err := ReadBootstrapLogs("")
		if !errors.Is(err, ErrEmptyLogFilePath) {
			t.Fatal("unexpected err", err)
		}
		if len(out) > 0 {
			t.Fatal("expected no tor logs")
		}
	})

	t.Run("with nonexistent file path", func(t *testing.T) {
		out, err := ReadBootstrapLogs(filepath.Join("testdata", "nonexistent"))
		if !errors.Is(err, ErrCannotReadLogFile) {
			t.Fatal("unexpected err", err)
		}
		if len(out) != 0 {
			t.Fatal("expected no tor logs")
		}
	})

	t.Run("with existing file path not containing bootstrap logs", func(t *testing.T) {
		out, err := ReadBootstrapLogs(filepath.Join("testdata", "empty.log"))
		if !errors.Is(err, ErrNoBootstrapLogs) {
			t.Fatal("unexpected err", err)
		}
		if len(out) != 0 {
			t.Fatal("expected no tor logs")
		}
	})

	t.Run("with existing file path containing bootstrap logs", func(t *testing.T) {
		out, err := ReadBootstrapLogs(filepath.Join("testdata", "tor.log"))
		if err != nil {
			t.Fatal(err)
		}
		if count := len(out); count != 9 {
			t.Fatal("unexpected number of tor logs", count)
		}
	})
}

func TestReadBootstrapLogsOrWarn(t *testing.T) {
	t.Run("on success", func(t *testing.T) {
		filename := filepath.Join("testdata", "tor.log")
		logs := ReadBootstrapLogsOrWarn(model.DiscardLogger, filename)
		if count := len(logs); count != 9 {
			t.Fatal("unexpected number of tor logs", count)
		}
	})

	t.Run("on failure", func(t *testing.T) {
		var called bool
		logger := &mocks.Logger{
			MockWarnf: func(format string, v ...interface{}) {
				called = true
			},
		}
		filename := filepath.Join("testdata", "empty.log")
		logs := ReadBootstrapLogsOrWarn(logger, filename)
		if !called {
			t.Fatal("not called")
		}
		if len(logs) != 0 {
			t.Fatal("expected no tor logs")
		}
	})
}

func TestParseBootstrapLogLine(t *testing.T) {
	type args struct {
		logLine string
	}
	tests := []struct {
		name    string
		args    args
		want    *BootstrapInfo
		wantErr error
	}{{
		name: "with empty string",
		args: args{
			logLine: "",
		},
		want:    nil,
		wantErr: ErrCannotFindSubmatches,
	}, {
		name: "with correct line",
		args: args{
			logLine: "May 10 09:19:28.000 [notice] Bootstrapped 80% (ap_conn): Connecting to a relay to build circuits",
		},
		want: &BootstrapInfo{
			Progress: 80,
			Tag:      "ap_conn",
			Summary:  "Connecting to a relay to build circuits",
		},
		wantErr: nil,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseBootstrapLogLine(tt.args.logLine)
			if !errors.Is(err, tt.wantErr) {
				t.Fatal("unexpected err", err)
				return
			}
			if diff := cmp.Diff(tt.want, got); diff != "" {
				t.Fatal(diff)
			}
		})
	}
}
