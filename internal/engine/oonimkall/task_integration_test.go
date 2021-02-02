package oonimkall_test

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/engine/model"
	"github.com/ooni/probe-cli/v3/internal/engine/oonimkall"
	"github.com/ooni/probe-cli/v3/internal/engine/oonimkall/tasks"
)

type eventlike struct {
	Key   string                 `json:"key"`
	Value map[string]interface{} `json:"value"`
}

func TestGood(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	task, err := oonimkall.StartTask(`{
		"assets_dir": "../testdata/oonimkall/assets",
		"log_level": "DEBUG",
		"name": "Example",
		"options": {
			"software_name": "oonimkall-test",
			"software_version": "0.1.0"
		},
		"state_dir": "../testdata/oonimkall/state",
		"version": 1
	}`)
	if err != nil {
		t.Fatal(err)
	}
	// interrupt the task so we also exercise this functionality
	go func() {
		<-time.After(time.Second)
		task.Interrupt()
	}()
	for !task.IsDone() {
		eventstr := task.WaitForNextEvent()
		var event eventlike
		if err := json.Unmarshal([]byte(eventstr), &event); err != nil {
			t.Fatal(err)
		}
		if event.Key == "failure.startup" {
			t.Fatal("unexpected failure.startup event")
		}
	}
	// make sure we only see task_terminated at this point
	for {
		eventstr := task.WaitForNextEvent()
		var event eventlike
		if err := json.Unmarshal([]byte(eventstr), &event); err != nil {
			t.Fatal(err)
		}
		if event.Key != "task_terminated" {
			t.Fatalf("unexpected event.Key: %s", event.Key)
		}
		break
	}
}

func TestWithMeasurementFailure(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	task, err := oonimkall.StartTask(`{
		"assets_dir": "../testdata/oonimkall/assets",
		"log_level": "DEBUG",
		"name": "ExampleWithFailure",
		"options": {
			"no_geoip": true,
			"no_resolver_lookup": true,
			"software_name": "oonimkall-test",
			"software_version": "0.1.0"
		},
		"state_dir": "../testdata/oonimkall/state",
		"version": 1
	}`)
	if err != nil {
		t.Fatal(err)
	}
	for !task.IsDone() {
		eventstr := task.WaitForNextEvent()
		var event eventlike
		if err := json.Unmarshal([]byte(eventstr), &event); err != nil {
			t.Fatal(err)
		}
		if event.Key == "failure.startup" {
			t.Fatal("unexpected failure.startup event")
		}
	}
}

func TestInvalidJSON(t *testing.T) {
	task, err := oonimkall.StartTask(`{`)
	var syntaxerr *json.SyntaxError
	if !errors.As(err, &syntaxerr) {
		t.Fatal("not the expected error")
	}
	if task != nil {
		t.Fatal("task is not nil")
	}
}

func TestUnsupportedSetting(t *testing.T) {
	task, err := oonimkall.StartTask(`{
		"assets_dir": "../testdata/oonimkall/assets",
		"log_level": "DEBUG",
		"name": "Example",
		"options": {
			"software_name": "oonimkall-test",
			"software_version": "0.1.0"
		},
		"state_dir": "../testdata/oonimkall/state"
	}`)
	if err != nil {
		t.Fatal(err)
	}
	var seen bool
	for !task.IsDone() {
		eventstr := task.WaitForNextEvent()
		var event eventlike
		if err := json.Unmarshal([]byte(eventstr), &event); err != nil {
			t.Fatal(err)
		}
		if event.Key == "failure.startup" {
			if strings.Contains(eventstr, tasks.FailureInvalidVersion) {
				seen = true
			}
		}
	}
	if !seen {
		t.Fatal("did not see failure.startup with invalid version info")
	}
}

func TestEmptyStateDir(t *testing.T) {
	task, err := oonimkall.StartTask(`{
		"assets_dir": "../testdata/oonimkall/assets",
		"log_level": "DEBUG",
		"name": "Example",
		"options": {
			"software_name": "oonimkall-test",
			"software_version": "0.1.0"
		},
		"version": 1
	}`)
	if err != nil {
		t.Fatal(err)
	}
	var seen bool
	for !task.IsDone() {
		eventstr := task.WaitForNextEvent()
		var event eventlike
		if err := json.Unmarshal([]byte(eventstr), &event); err != nil {
			t.Fatal(err)
		}
		if event.Key == "failure.startup" {
			if strings.Contains(eventstr, "mkdir : no such file or directory") {
				seen = true
			}
		}
	}
	if !seen {
		t.Fatal("did not see failure.startup with info that state dir is empty")
	}
}

func TestEmptyAssetsDir(t *testing.T) {
	task, err := oonimkall.StartTask(`{
		"log_level": "DEBUG",
		"name": "Example",
		"options": {
			"software_name": "oonimkall-test",
			"software_version": "0.1.0"
		},
		"state_dir": "../testdata/oonimkall/state",
		"version": 1
	}`)
	if err != nil {
		t.Fatal(err)
	}
	var seen bool
	for !task.IsDone() {
		eventstr := task.WaitForNextEvent()
		var event eventlike
		if err := json.Unmarshal([]byte(eventstr), &event); err != nil {
			t.Fatal(err)
		}
		if event.Key == "failure.startup" {
			if strings.Contains(eventstr, "AssetsDir is empty") {
				seen = true
			}
		}
	}
	if !seen {
		t.Fatal("did not see failure.startup")
	}
}

func TestUnknownExperiment(t *testing.T) {
	task, err := oonimkall.StartTask(`{
		"assets_dir": "../testdata/oonimkall/assets",
		"log_level": "DEBUG",
		"name": "Antani",
		"options": {
			"software_name": "oonimkall-test",
			"software_version": "0.1.0"
		},
		"state_dir": "../testdata/oonimkall/state",
		"version": 1
	}`)
	if err != nil {
		t.Fatal(err)
	}
	var seen bool
	for !task.IsDone() {
		eventstr := task.WaitForNextEvent()
		var event eventlike
		if err := json.Unmarshal([]byte(eventstr), &event); err != nil {
			t.Fatal(err)
		}
		if event.Key == "failure.startup" {
			if strings.Contains(eventstr, "no such experiment: ") {
				seen = true
			}
		}
	}
	if !seen {
		t.Fatal("did not see failure.startup")
	}
}

func TestInputIsRequired(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	task, err := oonimkall.StartTask(`{
		"assets_dir": "../testdata/oonimkall/assets",
		"log_level": "DEBUG",
		"name": "ExampleWithInput",
		"options": {
			"software_name": "oonimkall-test",
			"software_version": "0.1.0"
		},
		"state_dir": "../testdata/oonimkall/state",
		"version": 1
	}`)
	if err != nil {
		t.Fatal(err)
	}
	var seen bool
	for !task.IsDone() {
		eventstr := task.WaitForNextEvent()
		var event eventlike
		if err := json.Unmarshal([]byte(eventstr), &event); err != nil {
			t.Fatal(err)
		}
		if event.Key == "failure.startup" {
			if strings.Contains(eventstr, "no input provided") {
				seen = true
			}
		}
	}
	if !seen {
		t.Fatal("did not see failure.startup")
	}
}

func TestMaxRuntime(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	begin := time.Now()
	task, err := oonimkall.StartTask(`{
		"assets_dir": "../testdata/oonimkall/assets",
		"inputs": ["a", "b", "c"],
		"name": "ExampleWithInput",
		"options": {
			"max_runtime": 1,
			"software_name": "oonimkall-test",
			"software_version": "0.1.0"
		},
		"state_dir": "../testdata/oonimkall/state",
		"version": 1
	}`)
	if err != nil {
		t.Fatal(err)
	}
	for !task.IsDone() {
		eventstr := task.WaitForNextEvent()
		var event eventlike
		if err := json.Unmarshal([]byte(eventstr), &event); err != nil {
			t.Fatal(err)
		}
		if event.Key == "failure.startup" {
			t.Fatal(eventstr)
		}
	}
	// The runtime is long because of ancillary operations and is even more
	// longer because of self shaping we may be performing (especially in
	// CI builds) using `-tags shaping`). We have experimentally determined
	// that ~10 seconds is the typical CI test run time. See:
	//
	// 1. https://github.com/ooni/probe-cli/v3/internal/engine/pull/588/checks?check_run_id=667263788
	//
	// 2. https://github.com/ooni/probe-cli/v3/internal/engine/pull/588/checks?check_run_id=667263855
	//
	// In case there are further timeouts, e.g. in the sessionresolver, the
	// time used by the experiment will be much more. This is for example the
	// case in https://github.com/ooni/probe-cli/v3/internal/engine/issues/1005.
	if time.Now().Sub(begin) > 10*time.Second {
		t.Fatal("expected shorter runtime")
	}
}

func TestInterruptExampleWithInput(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	t.Skip("Skipping broken test; see https://github.com/ooni/probe-cli/v3/internal/engine/issues/992")
	task, err := oonimkall.StartTask(`{
		"assets_dir": "../testdata/oonimkall/assets",
		"inputs": [
			"http://www.kernel.org/",
			"http://www.x.org/",
			"http://www.microsoft.com/",
			"http://www.slashdot.org/",
			"http://www.repubblica.it/",
			"http://www.google.it/",
			"http://ooni.org/"
		],
		"name": "ExampleWithInputNonInterruptible",
		"options": {
			"software_name": "oonimkall-test",
			"software_version": "0.1.0"
		},
		"state_dir": "../testdata/oonimkall/state",
		"version": 1
	}`)
	if err != nil {
		t.Fatal(err)
	}
	var keys []string
	for !task.IsDone() {
		eventstr := task.WaitForNextEvent()
		var event eventlike
		if err := json.Unmarshal([]byte(eventstr), &event); err != nil {
			t.Fatal(err)
		}
		switch event.Key {
		case "failure.startup":
			t.Fatal(eventstr)
		case "status.measurement_start":
			go task.Interrupt()
		}
		// We compress the keys. What matters is basically that we
		// see just one of the many possible measurements here.
		if keys == nil || keys[len(keys)-1] != event.Key {
			keys = append(keys, event.Key)
		}
	}
	expect := []string{
		"status.queued",
		"status.started",
		"status.progress",
		"status.geoip_lookup",
		"status.resolver_lookup",
		"status.progress",
		"status.report_create",
		"status.measurement_start",
		"log",
		"status.progress",
		"measurement",
		"status.measurement_submission",
		"status.measurement_done",
		"status.end",
		"task_terminated",
	}
	if diff := cmp.Diff(expect, keys); diff != "" {
		t.Fatal(diff)
	}
}

func TestInterruptNdt7(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	task, err := oonimkall.StartTask(`{
		"assets_dir": "../testdata/oonimkall/assets",
		"name": "Ndt7",
		"options": {
			"software_name": "oonimkall-test",
			"software_version": "0.1.0"
		},
		"state_dir": "../testdata/oonimkall/state",
		"version": 1
	}`)
	if err != nil {
		t.Fatal(err)
	}
	go func() {
		<-time.After(11 * time.Second)
		task.Interrupt()
	}()
	var keys []string
	for !task.IsDone() {
		eventstr := task.WaitForNextEvent()
		var event eventlike
		if err := json.Unmarshal([]byte(eventstr), &event); err != nil {
			t.Fatal(err)
		}
		if event.Key == "failure.startup" {
			t.Fatal(eventstr)
		}
		// We compress the keys because we don't know how many
		// status.progress we will see. What matters is that we
		// don't see a measurement submission, since it means
		// that we have interrupted the measurement.
		if keys == nil || keys[len(keys)-1] != event.Key {
			keys = append(keys, event.Key)
		}
	}
	expect := []string{
		"status.queued",
		"status.started",
		"status.progress",
		"status.geoip_lookup",
		"status.resolver_lookup",
		"status.progress",
		"status.report_create",
		"status.measurement_start",
		"status.progress",
		"status.end",
		"task_terminated",
	}
	if diff := cmp.Diff(expect, keys); diff != "" {
		t.Fatal(diff)
	}
}

func TestCountBytesForExample(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	task, err := oonimkall.StartTask(`{
		"assets_dir": "../testdata/oonimkall/assets",
		"name": "Example",
		"options": {
			"software_name": "oonimkall-test",
			"software_version": "0.1.0"
		},
		"state_dir": "../testdata/oonimkall/state",
		"version": 1
	}`)
	if err != nil {
		t.Fatal(err)
	}
	var downloadKB, uploadKB float64
	for !task.IsDone() {
		eventstr := task.WaitForNextEvent()
		var event eventlike
		if err := json.Unmarshal([]byte(eventstr), &event); err != nil {
			t.Fatal(err)
		}
		switch event.Key {
		case "failure.startup":
			t.Fatal(eventstr)
		case "status.end":
			downloadKB = event.Value["downloaded_kb"].(float64)
			uploadKB = event.Value["uploaded_kb"].(float64)
		}
	}
	if downloadKB == 0 {
		t.Fatal("downloadKB is zero")
	}
	if uploadKB == 0 {
		t.Fatal("uploadKB is zero")
	}
}

func TestPrivacyAndScrubbing(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	task, err := oonimkall.StartTask(`{
			"assets_dir": "../testdata/oonimkall/assets",
			"name": "Example",
			"options": {
				"software_name": "oonimkall-test",
				"software_version": "0.1.0"
			},
			"state_dir": "../testdata/oonimkall/state",
			"version": 1
		}`)
	if err != nil {
		t.Fatal(err)
	}
	var m *model.Measurement
	for !task.IsDone() {
		eventstr := task.WaitForNextEvent()
		var event eventlike
		if err := json.Unmarshal([]byte(eventstr), &event); err != nil {
			t.Fatal(err)
		}
		switch event.Key {
		case "failure.startup":
			t.Fatal(eventstr)
		case "measurement":
			v := []byte(event.Value["json_str"].(string))
			m = new(model.Measurement)
			if err := json.Unmarshal(v, &m); err != nil {
				t.Fatal(err)
			}
		}
	}
	if m == nil {
		t.Fatal("measurement is nil")
	}
	if m.ProbeASN == "AS0" || m.ProbeCC == "ZZ" || m.ProbeIP != "127.0.0.1" {
		t.Fatal("unexpected result")
	}
}

func TestNonblock(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	task, err := oonimkall.StartTask(`{
		"assets_dir": "../testdata/oonimkall/assets",
		"name": "Example",
		"options": {
			"software_name": "oonimkall-test",
			"software_version": "0.1.0"
		},
		"state_dir": "../testdata/oonimkall/state",
		"version": 1
	}`)
	if err != nil {
		t.Fatal(err)
	}
	if !task.IsRunning() {
		t.Fatal("The runner should be running at this point")
	}
	// If the task blocks because it emits too much events, this test
	// will run forever and will be killed. Because we have room for up
	// to 128 events in the buffer, we should hopefully be fine.
	for task.IsRunning() {
		time.Sleep(time.Second)
	}
	for !task.IsDone() {
		task.WaitForNextEvent()
	}
}
