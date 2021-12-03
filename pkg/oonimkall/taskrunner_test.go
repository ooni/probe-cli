package oonimkall

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	engine "github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/internal/engine/model"
)

func TestMeasurementSubmissionEventName(t *testing.T) {
	if measurementSubmissionEventName(nil) != eventTypeStatusMeasurementSubmission {
		t.Fatal("unexpected submission event name")
	}
	if measurementSubmissionEventName(errors.New("mocked error")) != eventTypeFailureMeasurementSubmission {
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

func TestTaskRunnerRun(t *testing.T) {

	// newRunnerForTesting is a factory for creating a new
	// runner that wraps newRunner and also sets a specific
	// taskSessionBuilder for testing purposes.
	newRunnerForTesting := func() (*runnerForTask, *CollectorTaskEmitter) {
		settings := &settings{
			Name: "Example",
			Options: settingsOptions{
				SoftwareName:    "oonimkall-test",
				SoftwareVersion: "0.1.0",
			},
			StateDir: "testdata/state",
			Version:  1,
		}
		e := &CollectorTaskEmitter{}
		r := newRunner(settings, e)
		return r, e
	}

	// runAndCollectContext runs the task until completion
	// and collects the emitted events. Remember that
	// it's not race safe to modify the events.
	runAndCollectContext := func(ctx context.Context, r taskRunner, e *CollectorTaskEmitter) []*event {
		r.Run(ctx)
		return e.Collect()
	}

	// runAndCollect is like runAndCollectContext
	// but uses context.Background()
	runAndCollect := func(r taskRunner, e *CollectorTaskEmitter) []*event {
		return runAndCollectContext(context.Background(), r, e)
	}

	// countEventsByKey returns the number of events
	// with the given key inside of the list.
	countEventsByKey := func(events []*event, key string) (count int) {
		for _, ev := range events {
			if ev.Key == key {
				count++
			}
		}
		return
	}

	// assertCountEventsByKey fails is the number of events
	// of the given type is not the expected one.
	assertCountEventsByKey := func(events []*event, key string, count int) {
		if countEventsByKey(events, key) != count {
			t.Fatalf("unexpected number of '%s' events", key)
		}
	}

	t.Run("with unsupported settings", func(t *testing.T) {
		runner, emitter := newRunnerForTesting()
		runner.settings.Version = 0 // force unsupported version
		events := runAndCollect(runner, emitter)
		assertCountEventsByKey(events, eventTypeFailureStartup, 1)
	})

	t.Run("with failure when creating a new kvstore", func(t *testing.T) {
		runner, emitter := newRunnerForTesting()
		// override the kvstore builder to provoke an error
		runner.kvStoreBuilder = &MockableKVStoreFSBuilder{
			MockNewFS: func(path string) (model.KeyValueStore, error) {
				return nil, errors.New("generic error")
			},
		}
		events := runAndCollect(runner, emitter)
		assertCountEventsByKey(events, eventTypeFailureStartup, 1)
	})

	t.Run("with unparsable proxyURL", func(t *testing.T) {
		runner, emitter := newRunnerForTesting()
		runner.settings.Proxy = "\t" // invalid proxy URL
		events := runAndCollect(runner, emitter)
		assertCountEventsByKey(events, eventTypeFailureStartup, 1)
	})

	t.Run("with a parsable proxyURL", func(t *testing.T) {
		runner, emitter := newRunnerForTesting()
		// set a valid URL
		runner.settings.Proxy = "https://127.0.0.1/"
		// set a fake session builder that causes the startup to
		// fail but records the config passed to NewSession
		saver := &SessionBuilderConfigSaver{}
		runner.sessionBuilder = saver
		events := runAndCollect(runner, emitter)
		assertCountEventsByKey(events, eventTypeFailureStartup, 1)
		if saver.Config.ProxyURL.String() != runner.settings.Proxy {
			t.Fatal("invalid proxy URL")
		}
	})

	t.Run("with custom probe services URL", func(t *testing.T) {
		runner, emitter := newRunnerForTesting()
		// set a probe services URL
		runner.settings.Options.ProbeServicesBaseURL = "https://127.0.0.1"
		// set a fake session builder that causes the startup to
		// fail but records the config passed to NewSession
		saver := &SessionBuilderConfigSaver{}
		runner.sessionBuilder = saver
		events := runAndCollect(runner, emitter)
		assertCountEventsByKey(events, eventTypeFailureStartup, 1)
		psu := saver.Config.AvailableProbeServices
		if len(psu) != 1 {
			t.Fatal("invalid length")
		}
		if psu[0].Type != "https" {
			t.Fatal("invalid type")
		}
		if psu[0].Address != runner.settings.Options.ProbeServicesBaseURL {
			t.Fatal("invalid address")
		}
		if psu[0].Front != "" {
			t.Fatal("invalid front")
		}
	})

	type eventKeyCount struct {
		Key   string
		Count int
	}

	// reduceEventsKeysIgnoreLog reduces the list of event keys
	// counting equal subsequent keys and ignoring log events
	reduceEventsKeysIgnoreLog := func(events []*event) (out []eventKeyCount) {
		var current eventKeyCount
		for _, ev := range events {
			if ev.Key == eventTypeLog {
				continue
			}
			if current.Key == ev.Key {
				current.Count++
				continue
			}
			if current.Key != "" {
				out = append(out, current)
			}
			current.Key = ev.Key
			current.Count = 1
		}
		if current.Key != "" {
			out = append(out, current)
		}
		return
	}

	// fakeSuccessfulRun returns a new set of dependencies that
	// will perform a fully successful, but fake, run.
	fakeSuccessfulRun := func() *MockableTaskRunnerDependencies {
		return &MockableTaskRunnerDependencies{
			MockableKibiBytesReceived: func() float64 {
				return 10
			},
			MockableKibiBytesSent: func() float64 {
				return 4
			},
			MockableOpenReportContext: func(ctx context.Context) error {
				return nil
			},
			MockableReportID: func() string {
				return "20211202T074907Z_example_IT_30722_n1_axDLHNUfJaV1IbuU"
			},
			MockableMeasureWithContext: func(ctx context.Context, input string) (*model.Measurement, error) {
				return &model.Measurement{}, nil
			},
			MockableSubmitAndUpdateMeasurementContext: func(ctx context.Context, measurement *model.Measurement) error {
				return nil
			},
			MockableSetCallbacks: func(callbacks model.ExperimentCallbacks) {
			},
			MockableInputPolicy: func() engine.InputPolicy {
				return engine.InputNone
			},
			MockableInterruptible: func() bool {
				return false
			},
			MockClose: func() error {
				return nil
			},
			MockMaybeLookupBackendsContext: func(ctx context.Context) error {
				return nil
			},
			MockMaybeLookupLocationContext: func(ctx context.Context) error {
				return nil
			},
			MockProbeIP: func() string {
				return "130.192.91.211"
			},
			MockProbeASNString: func() string {
				return "AS137"
			},
			MockProbeCC: func() string {
				return "IT"
			},
			MockProbeNetworkName: func() string {
				return "GARR"
			},
			MockResolverASNString: func() string {
				return "AS137"
			},
			MockResolverIP: func() string {
				return "130.192.3.24"
			},
			MockResolverNetworkName: func() string {
				return "GARR"
			},
		}
	}

	assertReducedEventsLike := func(t *testing.T, expected, got []eventKeyCount) {
		if diff := cmp.Diff(expected, got); diff != "" {
			t.Fatal(diff)
		}
	}

	t.Run("with invalid experiment name", func(t *testing.T) {
		runner, emitter := newRunnerForTesting()
		fake := fakeSuccessfulRun()
		fake.MockNewExperimentBuilderByName = func(name string) (taskExperimentBuilder, error) {
			return nil, errors.New("invalid experiment name")
		}
		runner.sessionBuilder = fake
		events := runAndCollect(runner, emitter)
		reduced := reduceEventsKeysIgnoreLog(events)
		expect := []eventKeyCount{{
			Key:   eventTypeStatusQueued,
			Count: 1,
		}, {
			Key:   eventTypeStatusStarted,
			Count: 1,
		}, {
			Key:   eventTypeFailureStartup,
			Count: 1,
		}, {
			Key:   eventTypeStatusEnd,
			Count: 1,
		}}
		assertReducedEventsLike(t, expect, reduced)
	})

	t.Run("with error during backends lookup", func(t *testing.T) {
		runner, emitter := newRunnerForTesting()
		fake := fakeSuccessfulRun()
		fake.MockMaybeLookupBackendsContext = func(ctx context.Context) error {
			return errors.New("mocked error")
		}
		runner.sessionBuilder = fake
		events := runAndCollect(runner, emitter)
		reduced := reduceEventsKeysIgnoreLog(events)
		expect := []eventKeyCount{{
			Key:   eventTypeStatusQueued,
			Count: 1,
		}, {
			Key:   eventTypeStatusStarted,
			Count: 1,
		}, {
			Key:   eventTypeFailureStartup,
			Count: 1,
		}, {
			Key:   eventTypeStatusEnd,
			Count: 1,
		}}
		assertReducedEventsLike(t, expect, reduced)
	})

	t.Run("with error during location lookup", func(t *testing.T) {
		runner, emitter := newRunnerForTesting()
		fake := fakeSuccessfulRun()
		fake.MockMaybeLookupLocationContext = func(ctx context.Context) error {
			return errors.New("mocked error")
		}
		runner.sessionBuilder = fake
		events := runAndCollect(runner, emitter)
		reduced := reduceEventsKeysIgnoreLog(events)
		expect := []eventKeyCount{{
			Key:   eventTypeStatusQueued,
			Count: 1,
		}, {
			Key:   eventTypeStatusStarted,
			Count: 1,
		}, {
			Key:   eventTypeStatusProgress,
			Count: 1,
		}, {
			Key:   eventTypeFailureIPLookup,
			Count: 1,
		}, {
			Key:   eventTypeFailureASNLookup,
			Count: 1,
		}, {
			Key:   eventTypeFailureCCLookup,
			Count: 1,
		}, {
			Key:   eventTypeFailureResolverLookup,
			Count: 1,
		}, {
			Key:   eventTypeStatusEnd,
			Count: 1,
		}}
		assertReducedEventsLike(t, expect, reduced)
	})

	t.Run("with missing input and input or query backend", func(t *testing.T) {
		runner, emitter := newRunnerForTesting()
		fake := fakeSuccessfulRun()
		fake.MockableInputPolicy = func() engine.InputPolicy {
			return engine.InputOrQueryBackend
		}
		runner.sessionBuilder = fake
		events := runAndCollect(runner, emitter)
		reduced := reduceEventsKeysIgnoreLog(events)
		expect := []eventKeyCount{{
			Key:   eventTypeStatusQueued,
			Count: 1,
		}, {
			Key:   eventTypeStatusStarted,
			Count: 1,
		}, {
			Key:   eventTypeStatusProgress,
			Count: 3,
		}, {
			Key:   eventTypeStatusGeoIPLookup,
			Count: 1,
		}, {
			Key:   eventTypeStatusResolverLookup,
			Count: 1,
		}, {
			Key:   eventTypeFailureStartup,
			Count: 1,
		}, {
			Key:   eventTypeStatusEnd,
			Count: 1,
		}}
		assertReducedEventsLike(t, expect, reduced)
	})

	t.Run("with missing input and input strictly required", func(t *testing.T) {
		runner, emitter := newRunnerForTesting()
		fake := fakeSuccessfulRun()
		fake.MockableInputPolicy = func() engine.InputPolicy {
			return engine.InputStrictlyRequired
		}
		runner.sessionBuilder = fake
		events := runAndCollect(runner, emitter)
		reduced := reduceEventsKeysIgnoreLog(events)
		expect := []eventKeyCount{{
			Key:   eventTypeStatusQueued,
			Count: 1,
		}, {
			Key:   eventTypeStatusStarted,
			Count: 1,
		}, {
			Key:   eventTypeStatusProgress,
			Count: 3,
		}, {
			Key:   eventTypeStatusGeoIPLookup,
			Count: 1,
		}, {
			Key:   eventTypeStatusResolverLookup,
			Count: 1,
		}, {
			Key:   eventTypeFailureStartup,
			Count: 1,
		}, {
			Key:   eventTypeStatusEnd,
			Count: 1,
		}}
		assertReducedEventsLike(t, expect, reduced)
	})

	t.Run("with failure opening report", func(t *testing.T) {
		runner, emitter := newRunnerForTesting()
		fake := fakeSuccessfulRun()
		fake.MockableOpenReportContext = func(ctx context.Context) error {
			return errors.New("mocked error")
		}
		runner.sessionBuilder = fake
		events := runAndCollect(runner, emitter)
		reduced := reduceEventsKeysIgnoreLog(events)
		expect := []eventKeyCount{{
			Key:   eventTypeStatusQueued,
			Count: 1,
		}, {
			Key:   eventTypeStatusStarted,
			Count: 1,
		}, {
			Key:   eventTypeStatusProgress,
			Count: 3,
		}, {
			Key:   eventTypeStatusGeoIPLookup,
			Count: 1,
		}, {
			Key:   eventTypeStatusResolverLookup,
			Count: 1,
		}, {
			Key:   eventTypeFailureReportCreate,
			Count: 1,
		}, {
			Key:   eventTypeStatusEnd,
			Count: 1,
		}}
		assertReducedEventsLike(t, expect, reduced)
	})

	t.Run("with success and no input", func(t *testing.T) {
		runner, emitter := newRunnerForTesting()
		fake := fakeSuccessfulRun()
		runner.sessionBuilder = fake
		events := runAndCollect(runner, emitter)
		reduced := reduceEventsKeysIgnoreLog(events)
		expect := []eventKeyCount{{
			Key:   eventTypeStatusQueued,
			Count: 1,
		}, {
			Key:   eventTypeStatusStarted,
			Count: 1,
		}, {
			Key:   eventTypeStatusProgress,
			Count: 3,
		}, {
			Key:   eventTypeStatusGeoIPLookup,
			Count: 1,
		}, {
			Key:   eventTypeStatusResolverLookup,
			Count: 1,
		}, {
			Key:   eventTypeStatusProgress,
			Count: 1,
		}, {
			Key:   eventTypeStatusReportCreate,
			Count: 1,
		}, {
			Key:   eventTypeStatusMeasurementStart,
			Count: 1,
		}, {
			Key:   eventTypeMeasurement,
			Count: 1,
		}, {
			Key:   eventTypeStatusMeasurementSubmission,
			Count: 1,
		}, {
			Key:   eventTypeStatusMeasurementDone,
			Count: 1,
		}, {
			Key:   eventTypeStatusEnd,
			Count: 1,
		}}
		assertReducedEventsLike(t, expect, reduced)
	})

	t.Run("with measurement failure and no input", func(t *testing.T) {
		runner, emitter := newRunnerForTesting()
		fake := fakeSuccessfulRun()
		fake.MockableMeasureWithContext = func(ctx context.Context, input string) (measurement *model.Measurement, err error) {
			return nil, errors.New("preconditions error")
		}
		runner.sessionBuilder = fake
		events := runAndCollect(runner, emitter)
		reduced := reduceEventsKeysIgnoreLog(events)
		expect := []eventKeyCount{{
			Key:   eventTypeStatusQueued,
			Count: 1,
		}, {
			Key:   eventTypeStatusStarted,
			Count: 1,
		}, {
			Key:   eventTypeStatusProgress,
			Count: 3,
		}, {
			Key:   eventTypeStatusGeoIPLookup,
			Count: 1,
		}, {
			Key:   eventTypeStatusResolverLookup,
			Count: 1,
		}, {
			Key:   eventTypeStatusProgress,
			Count: 1,
		}, {
			Key:   eventTypeStatusReportCreate,
			Count: 1,
		}, {
			Key:   eventTypeStatusMeasurementStart,
			Count: 1,
		}, {
			Key:   eventTypeFailureMeasurement,
			Count: 1,
		}, {
			Key:   eventTypeStatusEnd,
			Count: 1,
		}}
		assertReducedEventsLike(t, expect, reduced)
	})

	t.Run("with success and input", func(t *testing.T) {
		runner, emitter := newRunnerForTesting()
		runner.settings.Inputs = []string{"a", "b", "c", "d"}
		fake := fakeSuccessfulRun()
		fake.MockableInputPolicy = func() engine.InputPolicy {
			return engine.InputStrictlyRequired
		}
		runner.sessionBuilder = fake
		events := runAndCollect(runner, emitter)
		reduced := reduceEventsKeysIgnoreLog(events)
		expect := []eventKeyCount{
			{Key: eventTypeStatusQueued, Count: 1},
			{Key: eventTypeStatusStarted, Count: 1},
			{Key: eventTypeStatusProgress, Count: 3},
			{Key: eventTypeStatusGeoIPLookup, Count: 1},
			{Key: eventTypeStatusResolverLookup, Count: 1},
			{Key: eventTypeStatusProgress, Count: 1},
			{Key: eventTypeStatusReportCreate, Count: 1},
			//
			{Key: eventTypeStatusMeasurementStart, Count: 1},
			{Key: eventTypeStatusProgress, Count: 1},
			{Key: eventTypeMeasurement, Count: 1},
			{Key: eventTypeStatusMeasurementSubmission, Count: 1},
			{Key: eventTypeStatusMeasurementDone, Count: 1},
			//
			{Key: eventTypeStatusMeasurementStart, Count: 1},
			{Key: eventTypeStatusProgress, Count: 1},
			{Key: eventTypeMeasurement, Count: 1},
			{Key: eventTypeStatusMeasurementSubmission, Count: 1},
			{Key: eventTypeStatusMeasurementDone, Count: 1},
			//
			{Key: eventTypeStatusMeasurementStart, Count: 1},
			{Key: eventTypeStatusProgress, Count: 1},
			{Key: eventTypeMeasurement, Count: 1},
			{Key: eventTypeStatusMeasurementSubmission, Count: 1},
			{Key: eventTypeStatusMeasurementDone, Count: 1},
			//
			{Key: eventTypeStatusMeasurementStart, Count: 1},
			{Key: eventTypeStatusProgress, Count: 1},
			{Key: eventTypeMeasurement, Count: 1},
			{Key: eventTypeStatusMeasurementSubmission, Count: 1},
			{Key: eventTypeStatusMeasurementDone, Count: 1},
			//
			{Key: eventTypeStatusEnd, Count: 1},
		}
		assertReducedEventsLike(t, expect, reduced)
	})

	t.Run("with succes and max runtime", func(t *testing.T) {
		runner, emitter := newRunnerForTesting()
		runner.settings.Inputs = []string{"a", "b", "c", "d"}
		runner.settings.Options.MaxRuntime = 2
		fake := fakeSuccessfulRun()
		fake.MockableInputPolicy = func() engine.InputPolicy {
			return engine.InputStrictlyRequired
		}
		fake.MockableMeasureWithContext = func(ctx context.Context, input string) (measurement *model.Measurement, err error) {
			time.Sleep(1 * time.Second)
			return &model.Measurement{}, nil
		}
		runner.sessionBuilder = fake
		events := runAndCollect(runner, emitter)
		reduced := reduceEventsKeysIgnoreLog(events)
		expect := []eventKeyCount{
			{Key: eventTypeStatusQueued, Count: 1},
			{Key: eventTypeStatusStarted, Count: 1},
			{Key: eventTypeStatusProgress, Count: 3},
			{Key: eventTypeStatusGeoIPLookup, Count: 1},
			{Key: eventTypeStatusResolverLookup, Count: 1},
			{Key: eventTypeStatusProgress, Count: 1},
			{Key: eventTypeStatusReportCreate, Count: 1},
			//
			{Key: eventTypeStatusMeasurementStart, Count: 1},
			{Key: eventTypeStatusProgress, Count: 1},
			{Key: eventTypeMeasurement, Count: 1},
			{Key: eventTypeStatusMeasurementSubmission, Count: 1},
			{Key: eventTypeStatusMeasurementDone, Count: 1},
			//
			{Key: eventTypeStatusMeasurementStart, Count: 1},
			{Key: eventTypeStatusProgress, Count: 1},
			{Key: eventTypeMeasurement, Count: 1},
			{Key: eventTypeStatusMeasurementSubmission, Count: 1},
			{Key: eventTypeStatusMeasurementDone, Count: 1},
			//
			{Key: eventTypeStatusEnd, Count: 1},
		}
		assertReducedEventsLike(t, expect, reduced)
	})

	t.Run("with interrupted experiment", func(t *testing.T) {
		runner, emitter := newRunnerForTesting()
		runner.settings.Inputs = []string{"a", "b", "c", "d"}
		runner.settings.Options.MaxRuntime = 2
		fake := fakeSuccessfulRun()
		fake.MockableInputPolicy = func() engine.InputPolicy {
			return engine.InputStrictlyRequired
		}
		fake.MockableInterruptible = func() bool {
			return true
		}
		ctx, cancel := context.WithCancel(context.Background())
		fake.MockableMeasureWithContext = func(ctx context.Context, input string) (measurement *model.Measurement, err error) {
			cancel()
			return &model.Measurement{}, nil
		}
		runner.sessionBuilder = fake
		events := runAndCollectContext(ctx, runner, emitter)
		reduced := reduceEventsKeysIgnoreLog(events)
		expect := []eventKeyCount{
			{Key: eventTypeStatusQueued, Count: 1},
			{Key: eventTypeStatusStarted, Count: 1},
			{Key: eventTypeStatusProgress, Count: 3},
			{Key: eventTypeStatusGeoIPLookup, Count: 1},
			{Key: eventTypeStatusResolverLookup, Count: 1},
			{Key: eventTypeStatusProgress, Count: 1},
			{Key: eventTypeStatusReportCreate, Count: 1},
			//
			{Key: eventTypeStatusMeasurementStart, Count: 1},
			{Key: eventTypeStatusProgress, Count: 1},
			//
			{Key: eventTypeStatusEnd, Count: 1},
		}
		assertReducedEventsLike(t, expect, reduced)
	})

	t.Run("with measurement submission failure", func(t *testing.T) {
		runner, emitter := newRunnerForTesting()
		runner.settings.Inputs = []string{"a"}
		fake := fakeSuccessfulRun()
		fake.MockableInputPolicy = func() engine.InputPolicy {
			return engine.InputStrictlyRequired
		}
		fake.MockableSubmitAndUpdateMeasurementContext = func(ctx context.Context, measurement *model.Measurement) error {
			return errors.New("cannot submit")
		}
		runner.sessionBuilder = fake
		events := runAndCollect(runner, emitter)
		reduced := reduceEventsKeysIgnoreLog(events)
		expect := []eventKeyCount{
			{Key: eventTypeStatusQueued, Count: 1},
			{Key: eventTypeStatusStarted, Count: 1},
			{Key: eventTypeStatusProgress, Count: 3},
			{Key: eventTypeStatusGeoIPLookup, Count: 1},
			{Key: eventTypeStatusResolverLookup, Count: 1},
			{Key: eventTypeStatusProgress, Count: 1},
			{Key: eventTypeStatusReportCreate, Count: 1},
			//
			{Key: eventTypeStatusMeasurementStart, Count: 1},
			{Key: eventTypeStatusProgress, Count: 1},
			{Key: eventTypeMeasurement, Count: 1},
			{Key: eventTypeFailureMeasurementSubmission, Count: 1},
			{Key: eventTypeStatusMeasurementDone, Count: 1},
			//
			{Key: eventTypeStatusEnd, Count: 1},
		}
		assertReducedEventsLike(t, expect, reduced)
	})

	t.Run("with success and progress", func(t *testing.T) {
		runner, emitter := newRunnerForTesting()
		fake := fakeSuccessfulRun()
		var callbacks model.ExperimentCallbacks
		fake.MockableSetCallbacks = func(cbs model.ExperimentCallbacks) {
			callbacks = cbs
		}
		fake.MockableMeasureWithContext = func(ctx context.Context, input string) (measurement *model.Measurement, err error) {
			callbacks.OnProgress(1, "hello from the fake experiment")
			return &model.Measurement{}, nil
		}
		runner.sessionBuilder = fake
		events := runAndCollect(runner, emitter)
		reduced := reduceEventsKeysIgnoreLog(events)
		expect := []eventKeyCount{{
			Key:   eventTypeStatusQueued,
			Count: 1,
		}, {
			Key:   eventTypeStatusStarted,
			Count: 1,
		}, {
			Key:   eventTypeStatusProgress,
			Count: 3,
		}, {
			Key:   eventTypeStatusGeoIPLookup,
			Count: 1,
		}, {
			Key:   eventTypeStatusResolverLookup,
			Count: 1,
		}, {
			Key:   eventTypeStatusProgress,
			Count: 1,
		}, {
			Key:   eventTypeStatusReportCreate,
			Count: 1,
		}, {
			Key:   eventTypeStatusMeasurementStart,
			Count: 1,
		}, {
			Key:   eventTypeStatusProgress,
			Count: 1,
		}, {
			Key:   eventTypeMeasurement,
			Count: 1,
		}, {
			Key:   eventTypeStatusMeasurementSubmission,
			Count: 1,
		}, {
			Key:   eventTypeStatusMeasurementDone,
			Count: 1,
		}, {
			Key:   eventTypeStatusEnd,
			Count: 1,
		}}
		assertReducedEventsLike(t, expect, reduced)
	})
}
