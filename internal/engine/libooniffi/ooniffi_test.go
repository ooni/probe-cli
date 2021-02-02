package main

import (
	"testing"
)

func TestTaskStartNullPointer(t *testing.T) {
	if ooniffi_task_start_(nil) != nil {
		t.Fatal("expected nil result here")
	}
}

func TestTaskStartInvalidJSON(t *testing.T) {
	settings := cstring("{")
	defer freestring(settings)
	if ooniffi_task_start_(settings) != nil {
		t.Fatal("expected nil result here")
	}
}

func TestTaskStartIdxWrapping(t *testing.T) {
	settings := cstring(`{
		"assets_dir": "../testdata/oonimkall/assets",
		"log_level": "DEBUG",
		"name": "Example",
		"options": {
			"software_name": "oonimkall-test",
			"software_version": "0.1.0"
		},
		"state_dir": "../testdata/oonimkall/state",
		"temp_dir": "../testdata/oonimkall/tmp"
	}`)
	defer freestring(settings)
	o := setmaxidx()
	// do twice and see if it's idempotent
	if task := ooniffi_task_start_(settings); task != nil {
		t.Fatal("expected nil task here")
	}
	if task := ooniffi_task_start_(settings); task != nil {
		t.Fatal("expected nil task here")
	}
	restoreidx(o)
}

func TestTaskWaitForNextEventNullPointer(t *testing.T) {
	if ooniffi_task_wait_for_next_event(nil) != nil {
		t.Fatal("expected nil result here")
	}
}

func TestTaskIsDoneNullPointer(t *testing.T) {
	if ooniffi_task_is_done(nil) == 0 {
		t.Fatal("expected true-ish result here")
	}
}

func TestTaskInterruptNullPointer(t *testing.T) {
	ooniffi_task_interrupt(nil) // mainly: we don't crash :^)
}

func TestEventSerializationNullPointer(t *testing.T) {
	if ooniffi_event_serialization_(nil) != nil {
		t.Fatal("expected nil result here")
	}
}

func TestEventDestroyNullPointer(t *testing.T) {
	ooniffi_event_destroy(nil) // mainly: we don't crash
}

func TestTaskDestroyNullPointer(t *testing.T) {
	ooniffi_task_destroy(nil) // mainly: we don't crash
}

func TestExampleNormalUsage(t *testing.T) {
	settings := cstring(`{
		"assets_dir": "../testdata/oonimkall/assets",
		"log_level": "DEBUG",
		"name": "Example",
		"options": {
			"software_name": "oonimkall-test",
			"software_version": "0.1.0"
		},
		"state_dir": "../testdata/oonimkall/state",
		"temp_dir": "../testdata/oonimkall/tmp"
	}`)
	defer freestring(settings)
	task := ooniffi_task_start_(settings)
	if task == nil {
		t.Fatal("expected non-nil task here")
	}
	for ooniffi_task_is_done(task) == 0 {
		event := ooniffi_task_wait_for_next_event(task)
		t.Logf("%s", gostring(ooniffi_event_serialization_(event)))
		ooniffi_event_destroy(event)
	}
	ooniffi_task_destroy(task)
}

func TestExampleInterruptAndDestroy(t *testing.T) {
	settings := cstring(`{
		"assets_dir": "../testdata/oonimkall/assets",
		"log_level": "DEBUG",
		"name": "Example",
		"options": {
			"software_name": "oonimkall-test",
			"software_version": "0.1.0"
		},
		"state_dir": "../testdata/oonimkall/state",
		"temp_dir": "../testdata/oonimkall/tmp"
	}`)
	defer freestring(settings)
	task := ooniffi_task_start_(settings)
	if task == nil {
		t.Fatal("expected non-nil task here")
	}
	ooniffi_task_interrupt(task)
	ooniffi_task_destroy(task)
}

func TestExampleDestroyImmediately(t *testing.T) {
	settings := cstring(`{
		"assets_dir": "../testdata/oonimkall/assets",
		"log_level": "DEBUG",
		"name": "Example",
		"options": {
			"software_name": "oonimkall-test",
			"software_version": "0.1.0"
		},
		"state_dir": "../testdata/oonimkall/state",
		"temp_dir": "../testdata/oonimkall/tmp"
	}`)
	defer freestring(settings)
	task := ooniffi_task_start_(settings)
	if task == nil {
		t.Fatal("expected non-nil task here")
	}
	ooniffi_task_destroy(task)
}
