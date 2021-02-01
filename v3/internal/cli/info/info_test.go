package info

import (
	"errors"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/ooni"
	"github.com/ooni/probe-cli/v3/internal/oonitest"
)

func TestNewProbeCLIFailed(t *testing.T) {
	expected := errors.New("mocked error")
	handler := &oonitest.FakeLoggerHandler{}
	err := doinfo(doinfoconfig{
		NewProbeCLI: func() (ooni.ProbeCLI, error) {
			return nil, expected
		},
		Logger: &log.Logger{
			Handler: handler,
			Level:   log.DebugLevel,
		},
	})
	if !errors.Is(err, expected) {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if len(handler.FakeEntries) != 1 {
		t.Fatal("invalid number of log entries")
	}
	entry := handler.FakeEntries[0]
	if entry.Level != log.ErrorLevel {
		t.Fatal("invalid log level")
	}
	if entry.Message != "mocked error" {
		t.Fatal("invalid .Message")
	}
}

func TestSuccess(t *testing.T) {
	handler := &oonitest.FakeLoggerHandler{}
	cli := &oonitest.FakeProbeCLI{
		FakeHome:    "fakehome",
		FakeTempDir: "faketempdir",
	}
	err := doinfo(doinfoconfig{
		NewProbeCLI: func() (ooni.ProbeCLI, error) {
			return cli, nil
		},
		Logger: &log.Logger{
			Handler: handler,
			Level:   log.DebugLevel,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(handler.FakeEntries) != 2 {
		t.Fatal("invalid number of log entries")
	}
	entry := handler.FakeEntries[0]
	if entry.Level != log.InfoLevel {
		t.Fatal("invalid log level")
	}
	if entry.Message != "Home" {
		t.Fatal("invalid .Message")
	}
	if entry.Fields["path"].(string) != "fakehome" {
		t.Fatal("invalid path")
	}
	entry = handler.FakeEntries[1]
	if entry.Level != log.InfoLevel {
		t.Fatal("invalid log level")
	}
	if entry.Message != "TempDir" {
		t.Fatal("invalid .Message")
	}
	if entry.Fields["path"].(string) != "faketempdir" {
		t.Fatal("invalid path")
	}
}
