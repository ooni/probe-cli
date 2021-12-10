package oonimkall

import "testing"

func TestTaskEmitterUsingChan(t *testing.T) {
	t.Run("ordinary emit", func(t *testing.T) {
		out := make(chan *event)
		emitter := newTaskEmitterUsingChan(out)
		go func() {
			emitter.Emit("foo", nil)
		}()
		ev := <-out
		if ev.Key != "foo" {
			t.Fatal("invalid key")
		}
		if ev.Value != nil {
			t.Fatal("invalid value")
		}
	})

	t.Run("emit after close", func(t *testing.T) {
		out := make(chan *event)
		emitter := newTaskEmitterUsingChan(out)
		emitter.Close()
		done := make(chan interface{})
		go func() {
			emitter.Emit("foo", nil)
			close(done)
		}()
		<-done
		select {
		case <-out:
			t.Fatal("should not receive event here")
		default:
		}
	})

	t.Run("close is idempotent", func(t *testing.T) {
		out := make(chan *event)
		emitter := newTaskEmitterUsingChan(out)
		for i := 0; i < 4; i++ {
			emitter.Close()
		}
	})
}

func TestTaskEmitterWrapper(t *testing.T) {
	t.Run("emit failureStartup", func(t *testing.T) {
		expect := "antani"
		collector := &CollectorTaskEmitter{}
		emitter := &taskEmitterWrapper{collector}
		emitter.EmitFailureStartup(expect)
		events := collector.Collect()
		if len(events) != 1 {
			t.Fatal("invalid number of events")
		}
		ev := events[0]
		if ev.Key != eventTypeFailureStartup {
			t.Fatal("invalid key")
		}
		value := ev.Value.(eventFailure)
		if value.Failure != expect {
			t.Fatal("invalid failure value")
		}
	})

	t.Run("emit failureGeneric", func(t *testing.T) {
		expectName := "mascetti"
		expectFailure := "antani"
		collector := &CollectorTaskEmitter{}
		emitter := &taskEmitterWrapper{collector}
		emitter.EmitFailureGeneric(expectName, expectFailure)
		events := collector.Collect()
		if len(events) != 1 {
			t.Fatal("invalid number of events")
		}
		ev := events[0]
		if ev.Key != expectName {
			t.Fatal("invalid key")
		}
		value := ev.Value.(eventFailure)
		if value.Failure != expectFailure {
			t.Fatal("invalid failure value")
		}
	})

	t.Run("emit statusProgress", func(t *testing.T) {
		percentage := 0.66
		message := "mascetti"
		collector := &CollectorTaskEmitter{}
		emitter := &taskEmitterWrapper{collector}
		emitter.EmitStatusProgress(percentage, message)
		events := collector.Collect()
		if len(events) != 1 {
			t.Fatal("invalid number of events")
		}
		ev := events[0]
		if ev.Key != eventTypeStatusProgress {
			t.Fatal("invalid key")
		}
		value := ev.Value.(eventStatusProgress)
		if value.Percentage != percentage {
			t.Fatal("invalid percentage value")
		}
		if value.Message != message {
			t.Fatal("invalid message value")
		}
	})
}
