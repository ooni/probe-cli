package oonimkall

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/model"
)

type eventlike struct {
	Key   string                 `json:"key"`
	Value map[string]interface{} `json:"value"`
}

func TestStartTaskGood(t *testing.T) {
	task, err := StartTask(`{
		"log_level": "DEBUG",
		"name": "Example",
		"options": {
			"software_name": "oonimkall-test",
			"software_version": "0.1.0"
		},
		"state_dir": "testdata/state",
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
		if event.Key == "task_terminated" {
			break
		}
		t.Fatalf("unexpected event.Key: %s", event.Key)
	}
}

func TestStartTaskInvalidJSON(t *testing.T) {
	task, err := StartTask(`{`)
	var syntaxerr *json.SyntaxError
	if !errors.As(err, &syntaxerr) {
		t.Fatal("not the expected error")
	}
	if task != nil {
		t.Fatal("task is not nil")
	}
}

func TestStartTaskCountBytesForExample(t *testing.T) {
	task, err := StartTask(`{
		"name": "Example",
		"options": {
			"software_name": "oonimkall-test",
			"software_version": "0.1.0"
		},
		"state_dir": "testdata/state",
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
	task, err := StartTask(`{
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

func TestNonblockWithFewEvents(t *testing.T) {
	// This test tests whether we won't block for a small
	// number of events emitted by the task
	task, err := StartTask(`{
		"name": "Example",
		"options": {
			"software_name": "oonimkall-test",
			"software_version": "0.1.0"
		},
		"state_dir": "testdata/state",
		"version": 1
	}`)
	if err != nil {
		t.Fatal(err)
	}
	// Wait for the task thread to start
	<-task.isstarted
	// Wait for the task thread to complete
	<-task.isstopped
	var count int
	for !task.IsDone() {
		task.WaitForNextEvent()
		count++
	}
	if count < 5 {
		t.Fatal("too few events")
	}
}
