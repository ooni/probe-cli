package tasks_test

import (
	"testing"

	"github.com/ooni/probe-cli/v3/pkg/oonimkall/tasks"
)

func TestDisabledEvents(t *testing.T) {
	out := make(chan *tasks.Event)
	emitter := tasks.NewEventEmitter([]string{"log"}, out)
	go func() {
		emitter.Emit("log", tasks.EventLog{Message: "foo"})
		close(out)
	}()
	var count int64
	for ev := range out {
		if ev.Key == "log" {
			count++
		}
	}
	if count > 0 {
		t.Fatal("cannot disable events")
	}
}

func TestEmitFailureStartup(t *testing.T) {
	out := make(chan *tasks.Event)
	emitter := tasks.NewEventEmitter([]string{}, out)
	go func() {
		emitter.EmitFailureStartup("mocked error")
		close(out)
	}()
	var found bool
	for ev := range out {
		if ev.Key == "failure.startup" {
			evv := ev.Value.(tasks.EventFailure) // panic if not castable
			if evv.Failure == "mocked error" {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("did not see expected event")
	}
}

func TestEmitStatusProgress(t *testing.T) {
	out := make(chan *tasks.Event)
	emitter := tasks.NewEventEmitter([]string{}, out)
	go func() {
		emitter.EmitStatusProgress(0.7, "foo")
		close(out)
	}()
	var found bool
	for ev := range out {
		if ev.Key == "status.progress" {
			evv := ev.Value.(tasks.EventStatusProgress) // panic if not castable
			if evv.Message == "foo" && evv.Percentage == 0.7 {
				found = true
			}
		}
	}
	if !found {
		t.Fatal("did not see expected event")
	}
}
