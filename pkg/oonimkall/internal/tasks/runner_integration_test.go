package tasks_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/pkg/oonimkall/internal/tasks"
)

func TestRunnerMaybeLookupBackendsFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer server.Close()
	out := make(chan *tasks.Event)
	settings := &tasks.Settings{
		AssetsDir: "../../testdata/oonimkall/assets",
		Name:      "Example",
		Options: tasks.SettingsOptions{
			ProbeServicesBaseURL: server.URL,
			SoftwareName:         "oonimkall-test",
			SoftwareVersion:      "0.1.0",
		},
		StateDir: "../../testdata/oonimkall/state",
		Version:  1,
	}
	go func() {
		tasks.Run(context.Background(), settings, out)
		close(out)
	}()
	var failures []string
	for ev := range out {
		switch ev.Key {
		case "failure.startup":
			failure := ev.Value.(tasks.EventFailure).Failure
			failures = append(failures, failure)
		case "status.queued", "status.started", "log", "status.end":
		default:
			panic(fmt.Sprintf("unexpected key: %s", ev.Key))
		}
	}
	if len(failures) != 1 {
		t.Fatal("unexpected number of failures")
	}
	if failures[0] != "all available probe services failed" {
		t.Fatal("invalid failure")
	}
}

func TestRunnerOpenReportFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	var (
		nreq int64
		mu   sync.Mutex
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		nreq++
		if nreq == 1 {
			w.Write([]byte(`{}`))
			return
		}
		w.WriteHeader(500)
	}))
	defer server.Close()
	out := make(chan *tasks.Event)
	settings := &tasks.Settings{
		AssetsDir: "../../testdata/oonimkall/assets",
		Name:      "Example",
		Options: tasks.SettingsOptions{
			ProbeServicesBaseURL: server.URL,
			SoftwareName:         "oonimkall-test",
			SoftwareVersion:      "0.1.0",
		},
		StateDir: "../../testdata/oonimkall/state",
		Version:  1,
	}
	seench := make(chan int64)
	go func() {
		var seen int64
		for ev := range out {
			switch ev.Key {
			case "failure.report_create":
				seen++
			case "status.progress":
				evv := ev.Value.(tasks.EventStatusProgress)
				if evv.Percentage >= 0.4 {
					panic(fmt.Sprintf("too much progress: %+v", ev))
				}
			case "status.queued", "status.started", "log", "status.end",
				"status.geoip_lookup", "status.resolver_lookup":
			default:
				panic(fmt.Sprintf("unexpected key: %s", ev.Key))
			}
		}
		seench <- seen
	}()
	tasks.Run(context.Background(), settings, out)
	close(out)
	if n := <-seench; n != 1 {
		t.Fatal("unexpected number of events")
	}
}

func TestRunnerGood(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	out := make(chan *tasks.Event)
	settings := &tasks.Settings{
		AssetsDir: "../../testdata/oonimkall/assets",
		LogLevel:  "DEBUG",
		Name:      "Example",
		Options: tasks.SettingsOptions{
			SoftwareName:    "oonimkall-test",
			SoftwareVersion: "0.1.0",
		},
		StateDir: "../../testdata/oonimkall/state",
		Version:  1,
	}
	go func() {
		tasks.Run(context.Background(), settings, out)
		close(out)
	}()
	var found bool
	for ev := range out {
		if ev.Key == "status.end" {
			found = true
		}
	}
	if !found {
		t.Fatal("status.end event not found")
	}
}

func TestRunnerWithUnsupportedSettings(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	out := make(chan *tasks.Event)
	settings := &tasks.Settings{
		AssetsDir: "../../testdata/oonimkall/assets",
		LogLevel:  "DEBUG",
		Name:      "Example",
		Options: tasks.SettingsOptions{
			SoftwareName:    "oonimkall-test",
			SoftwareVersion: "0.1.0",
		},
		StateDir: "../../testdata/oonimkall/state",
	}
	go func() {
		tasks.Run(context.Background(), settings, out)
		close(out)
	}()
	var failures []string
	for ev := range out {
		if ev.Key == "failure.startup" {
			failure := ev.Value.(tasks.EventFailure).Failure
			failures = append(failures, failure)
		}
	}
	if len(failures) != 1 {
		t.Fatal("invalid number of failures")
	}
	if failures[0] != tasks.FailureInvalidVersion {
		t.Fatal("not the failure we expected")
	}
}

func TestRunnerWithInvalidKVStorePath(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	out := make(chan *tasks.Event)
	settings := &tasks.Settings{
		AssetsDir: "../../testdata/oonimkall/assets",
		LogLevel:  "DEBUG",
		Name:      "Example",
		Options: tasks.SettingsOptions{
			SoftwareName:    "oonimkall-test",
			SoftwareVersion: "0.1.0",
		},
		StateDir: "", // must be empty to cause the failure below
		Version:  1,
	}
	go func() {
		tasks.Run(context.Background(), settings, out)
		close(out)
	}()
	var failures []string
	for ev := range out {
		if ev.Key == "failure.startup" {
			failure := ev.Value.(tasks.EventFailure).Failure
			failures = append(failures, failure)
		}
	}
	if len(failures) != 1 {
		t.Fatal("invalid number of failures")
	}
	if failures[0] != "mkdir : no such file or directory" {
		t.Fatal("not the failure we expected")
	}
}

func TestRunnerWithInvalidExperimentName(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	out := make(chan *tasks.Event)
	settings := &tasks.Settings{
		AssetsDir: "../../testdata/oonimkall/assets",
		LogLevel:  "DEBUG",
		Name:      "Nonexistent",
		Options: tasks.SettingsOptions{
			SoftwareName:    "oonimkall-test",
			SoftwareVersion: "0.1.0",
		},
		StateDir: "../../testdata/oonimkall/state",
		Version:  1,
	}
	go func() {
		tasks.Run(context.Background(), settings, out)
		close(out)
	}()
	var failures []string
	for ev := range out {
		if ev.Key == "failure.startup" {
			failure := ev.Value.(tasks.EventFailure).Failure
			failures = append(failures, failure)
		}
	}
	if len(failures) != 1 {
		t.Fatal("invalid number of failures")
	}
	if failures[0] != "no such experiment: Nonexistent" {
		t.Fatalf("not the failure we expected: %s", failures[0])
	}
}

func TestRunnerWithMissingInput(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	out := make(chan *tasks.Event)
	settings := &tasks.Settings{
		AssetsDir: "../../testdata/oonimkall/assets",
		LogLevel:  "DEBUG",
		Name:      "ExampleWithInput",
		Options: tasks.SettingsOptions{
			SoftwareName:    "oonimkall-test",
			SoftwareVersion: "0.1.0",
		},
		StateDir: "../../testdata/oonimkall/state",
		Version:  1,
	}
	go func() {
		tasks.Run(context.Background(), settings, out)
		close(out)
	}()
	var failures []string
	for ev := range out {
		if ev.Key == "failure.startup" {
			failure := ev.Value.(tasks.EventFailure).Failure
			failures = append(failures, failure)
		}
	}
	if len(failures) != 1 {
		t.Fatal("invalid number of failures")
	}
	if failures[0] != "no input provided" {
		t.Fatalf("not the failure we expected: %s", failures[0])
	}
}

func TestRunnerWithDefaultInput(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	out := make(chan *tasks.Event)
	settings := &tasks.Settings{
		AssetsDir: "../../testdata/oonimkall/assets",
		LogLevel:  "DEBUG",
		Name:      "ExampleWithDefaultInput",
		Options: tasks.SettingsOptions{
			SoftwareName:    "oonimkall-test",
			SoftwareVersion: "0.1.0",
		},
		StateDir: "../../testdata/oonimkall/state",
		Version:  1,
	}
	go func() {
		tasks.Run(context.Background(), settings, out)
		close(out)
	}()
	var found bool
	for ev := range out {
		if ev.Key == "status.end" {
			found = true
			continue
		}
		if ev.Key == "failure.startup" {
			t.Fatal("seen a failure.startup event")
		}
	}
	if !found {
		t.Fatal("status.end event not found")
	}
}

func TestRunnerWithDefaultInputWronglyConfigured(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	out := make(chan *tasks.Event)
	settings := &tasks.Settings{
		AssetsDir: "../../testdata/oonimkall/assets",
		LogLevel:  "DEBUG",
		Name:      "ExampleWithDefaultInputWronglyConfigured",
		Options: tasks.SettingsOptions{
			SoftwareName:    "oonimkall-test",
			SoftwareVersion: "0.1.0",
		},
		StateDir: "../../testdata/oonimkall/state",
		Version:  1,
	}
	go func() {
		tasks.Run(context.Background(), settings, out)
		close(out)
	}()
	var failures []string
	for ev := range out {
		if ev.Key == "failure.startup" {
			failure := ev.Value.(tasks.EventFailure).Failure
			failures = append(failures, failure)
		}
	}
	if len(failures) != 1 {
		t.Fatal("invalid number of failures")
	}
	if failures[0] != "no input provided" {
		t.Fatalf("not the failure we expected: %s", failures[0])
	}
}

func TestRunnerWithMaxRuntime(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	out := make(chan *tasks.Event)
	settings := &tasks.Settings{
		AssetsDir: "../../testdata/oonimkall/assets",
		Inputs:    []string{"a", "b", "c", "d", "e", "f", "g", "h", "i"},
		LogLevel:  "DEBUG",
		Name:      "ExampleWithInput",
		Options: tasks.SettingsOptions{
			MaxRuntime:      1,
			SoftwareName:    "oonimkall-test",
			SoftwareVersion: "0.1.0",
		},
		StateDir: "../../testdata/oonimkall/state",
		Version:  1,
	}
	begin := time.Now()
	go func() {
		tasks.Run(context.Background(), settings, out)
		close(out)
	}()
	var found bool
	for ev := range out {
		if ev.Key == "status.end" {
			found = true
		}
	}
	if !found {
		t.Fatal("status.end event not found")
	}
	// The runtime is long because of ancillary operations and is even more
	// longer because of self shaping we may be performing (especially in
	// CI builds) using `-tags shaping`). We have experimentally determined
	// that ~10 seconds is the typical CI test run time. See:
	//
	// 1. https://github.com/ooni/probe-cli/v3/internal/engine/pull/588/checks?check_run_id=667263788
	//
	// 2. https://github.com/ooni/probe-cli/v3/internal/engine/pull/588/checks?check_run_id=667263855
	if time.Now().Sub(begin) > 10*time.Second {
		t.Fatal("expected shorter runtime")
	}
}

func TestRunnerWithMaxRuntimeNonInterruptible(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	out := make(chan *tasks.Event)
	settings := &tasks.Settings{
		AssetsDir: "../../testdata/oonimkall/assets",
		Inputs:    []string{"a", "b", "c", "d", "e", "f", "g", "h", "i"},
		LogLevel:  "DEBUG",
		Name:      "ExampleWithInputNonInterruptible",
		Options: tasks.SettingsOptions{
			MaxRuntime:      1,
			SoftwareName:    "oonimkall-test",
			SoftwareVersion: "0.1.0",
		},
		StateDir: "../../testdata/oonimkall/state",
		Version:  1,
	}
	begin := time.Now()
	go func() {
		tasks.Run(context.Background(), settings, out)
		close(out)
	}()
	var found bool
	for ev := range out {
		if ev.Key == "status.end" {
			found = true
		}
	}
	if !found {
		t.Fatal("status.end event not found")
	}
	// The runtime is long because of ancillary operations and is even more
	// longer because of self shaping we may be performing (especially in
	// CI builds) using `-tags shaping`). We have experimentally determined
	// that ~10 seconds is the typical CI test run time. See:
	//
	// 1. https://github.com/ooni/probe-cli/v3/internal/engine/pull/588/checks?check_run_id=667263788
	//
	// 2. https://github.com/ooni/probe-cli/v3/internal/engine/pull/588/checks?check_run_id=667263855
	if time.Now().Sub(begin) > 10*time.Second {
		t.Fatal("expected shorter runtime")
	}
}

func TestRunnerWithFailedMeasurement(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	out := make(chan *tasks.Event)
	settings := &tasks.Settings{
		AssetsDir: "../../testdata/oonimkall/assets",
		LogLevel:  "DEBUG",
		Name:      "ExampleWithFailure",
		Options: tasks.SettingsOptions{
			MaxRuntime:      1,
			SoftwareName:    "oonimkall-test",
			SoftwareVersion: "0.1.0",
		},
		StateDir: "../../testdata/oonimkall/state",
		Version:  1,
	}
	go func() {
		tasks.Run(context.Background(), settings, out)
		close(out)
	}()
	var found bool
	for ev := range out {
		if ev.Key == "failure.measurement" {
			found = true
		}
	}
	if !found {
		t.Fatal("failure.measurement event not found")
	}
}
