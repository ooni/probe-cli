package main

import (
	"context"
	"testing"
)

func TestReadLines(t *testing.T) {
	lines := readLines("testdata/testmeasurement.json")
	if lines == nil {
		t.Fatal("unexpected error")
	}
	if len(lines) != 2 {
		t.Fatal("unexpected number of measurements")
	}
}

func TestNewSessionAndSubmitter(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	ctx := context.Background()
	sess := newSession(ctx)
	if sess == nil {
		t.Fatal("unexpected nil session")
	}
	subm := newSubmitter(sess, ctx)
	if subm == nil {
		t.Fatal("unexpected nil submitter")
	}
}

func TestToMeasurement(t *testing.T) {
	lines := readLines("testdata/testmeasurement.json")
	line := lines[0]
	mm := toMeasurement(line)
	if mm == nil {
		t.Fatal("unexpected error")
	}
}

func TestMainMissingFile(t *testing.T) {
	defer func() {
		var s interface{}
		if s = recover(); s == nil {
			t.Fatal("expected a panic here")
		}
		if s != "Cannot open measurement file" {
			t.Fatal("unexpected panic message")
		}
	}()
	mainWithArgs([]string{"upload", "notexist.json"})
}

func TestMainEmptyFile(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	defer func() {
		var s interface{}
		if s = recover(); s != nil {
			t.Fatal("unexpected panic")
		}
	}()
	mainWithArgs([]string{"upload", "testdata/noentries.json"})
}

func TestSubmitAllFails(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	ctx := context.Background()
	sess := newSession(ctx)
	subm := newSubmitter(sess, ctx)
	lines := readLines("testdata/testmeasurement.json")

	ctx, cancel := context.WithCancel(ctx)
	cancel() // fail immediately

	n, err := submitAll(ctx, lines, subm)
	if err == nil {
		t.Fatal("expected an error here")
	}
	if n != 0 {
		t.Fatal("nothing should be submitted here")
	}
}
