package main

import "testing"

func TestSimple(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	MainWithConfiguration("example", &Options{
		Yes: true,
	})
}
