package oonimkall

import (
	"context"
	"errors"
	"fmt"
	"testing"

	engine "github.com/ooni/probe-cli/v3/internal/engine"
)

func TestMeasurementSubmissionEventName(t *testing.T) {
	if measurementSubmissionEventName(nil) != statusMeasurementSubmission {
		t.Fatal("unexpected submission event name")
	}
	if measurementSubmissionEventName(errors.New("mocked error")) != failureMeasurementSubmission {
		t.Fatal("unexpected submission event name")
	}
}

func TestMeasurementSubmissionFailure(t *testing.T) {
	if measurementSubmissionFailure(nil) != "" {
		t.Fatal("unexpected submission failure")
	}
	if measurementSubmissionFailure(errors.New("mocked error")) != "mocked error" {
		t.Fatal("unexpected submission failure")
	}
}

func TestRunnerMaybeLookupLocationFailure(t *testing.T) {
	if testing.Short() {
		// TODO(https://github.com/ooni/probe-cli/pull/518)
		t.Skip("skip test in short mode")
	}
	settings := &settings{
		AssetsDir: "../../testdata/oonimkall/assets",
		Name:      "Example",
		Options: settingsOptions{
			SoftwareName:    "oonimkall-test",
			SoftwareVersion: "0.1.0",
		},
		StateDir: "../../testdata/oonimkall/state",
		Version:  1,
	}
	emitter := &CollectorTaskEmitter{}
	r := newRunner(settings, emitter)
	expected := errors.New("mocked error")
	r.maybeLookupLocation = func(*engine.Session) error {
		return expected
	}
	r.Run(context.Background())
	var seen int
	for _, ev := range emitter.Collect() {
		switch ev.Key {
		case "failure.ip_lookup", "failure.asn_lookup",
			"failure.cc_lookup", "failure.resolver_lookup":
			seen++
		case "status.progress":
			evv := ev.Value.(eventStatusProgress)
			if evv.Percentage >= 0.2 {
				panic(fmt.Sprintf("too much progress: %+v", ev))
			}
		case "status.queued", "status.started", "status.end":
		default:
			panic(fmt.Sprintf("unexpected key: %s - %+v", ev.Key, ev.Value))
		}
	}
	if seen != 4 {
		t.Fatal("unexpected number of events")
	}
}
