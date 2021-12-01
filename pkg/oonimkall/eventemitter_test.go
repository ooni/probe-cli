package oonimkall

import "testing"

func TestDisabledEvents(t *testing.T) {
	out := make(chan *event)
	eof := make(chan interface{})
	emitter := newEventEmitter([]string{"log"}, out, eof)
	go func() {
		emitter.Emit("log", eventLog{Message: "foo"})
		close(eof)
	}()
	var count int64
Loop:
	for {
		select {
		case ev := <-out:
			if ev.Key == "log" {
				count++
			}
		case <-eof:
			break Loop
		}
	}
	if count > 0 {
		t.Fatal("cannot disable events")
	}
}

func TestEmitFailureStartup(t *testing.T) {
	out := make(chan *event)
	eof := make(chan interface{})
	emitter := newEventEmitter([]string{}, out, eof)
	go func() {
		emitter.EmitFailureStartup("mocked error")
		close(eof)
	}()
	var found bool
Loop:
	for {
		select {
		case ev := <-out:
			if ev.Key == "failure.startup" {
				evv := ev.Value.(eventFailure) // panic if not castable
				if evv.Failure == "mocked error" {
					found = true
				}
			}
		case <-eof:
			break Loop
		}
	}
	if !found {
		t.Fatal("did not see expected event")
	}
}

func TestEmitStatusProgress(t *testing.T) {
	out := make(chan *event)
	eof := make(chan interface{})
	emitter := newEventEmitter([]string{}, out, eof)
	go func() {
		emitter.EmitStatusProgress(0.7, "foo")
		close(eof)
	}()
	var found bool
Loop:
	for {
		select {
		case ev := <-out:
			if ev.Key == "status.progress" {
				evv := ev.Value.(eventStatusProgress) // panic if not castable
				if evv.Message == "foo" && evv.Percentage == 0.7 {
					found = true
				}
			}
		case <-eof:
			break Loop
		}
	}
	if !found {
		t.Fatal("did not see expected event")
	}
}
