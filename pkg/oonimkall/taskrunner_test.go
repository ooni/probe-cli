package oonimkall

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/targetloading"
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

// MockableTaskRunnerDependencies is the mockable struct used by [TestTaskRunnerRun].
type MockableTaskRunnerDependencies struct {
	Builder    *mocks.ExperimentBuilder
	Experiment *mocks.Experiment
	Loader     *mocks.ExperimentTargetLoader
	Session    *mocks.Session
}

// NewSession is the method that returns the new fake session.
func (dep *MockableTaskRunnerDependencies) NewSession(ctx context.Context, config engine.SessionConfig) (taskSession, error) {
	return dep.Session, nil
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
		runner.newKVStore = func(path string) (model.KeyValueStore, error) {
			return nil, errors.New("generic error")
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
		var savedConfig engine.SessionConfig
		runner.newSession = func(ctx context.Context, config engine.SessionConfig) (taskSession, error) {
			savedConfig = config
			return nil, errors.New("generic error")
		}
		events := runAndCollect(runner, emitter)
		assertCountEventsByKey(events, eventTypeFailureStartup, 1)
		if savedConfig.ProxyURL.String() != runner.settings.Proxy {
			t.Fatal("invalid proxy URL")
		}
	})

	t.Run("with custom probe services URL", func(t *testing.T) {
		runner, emitter := newRunnerForTesting()
		// set a probe services URL
		runner.settings.Options.ProbeServicesBaseURL = "https://127.0.0.1"
		// set a fake session builder that causes the startup to
		// fail but records the config passed to NewSession
		var savedConfig engine.SessionConfig
		runner.newSession = func(ctx context.Context, config engine.SessionConfig) (taskSession, error) {
			savedConfig = config
			return nil, errors.New("generic error")
		}
		events := runAndCollect(runner, emitter)
		assertCountEventsByKey(events, eventTypeFailureStartup, 1)
		psu := savedConfig.AvailableProbeServices
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
	reduceEventsKeysIgnoreLog := func(t *testing.T, events []*event) (out []eventKeyCount) {
		var current eventKeyCount
		for _, ev := range events {
			t.Log(ev)
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

	// fakeSuccessfulDeps returns a new set of dependencies that
	// will perform a fully successful, but fake, run.
	//
	// You MAY override some functions to provoke specific errors
	// or generally change the operating conditions.
	fakeSuccessfulDeps := func() *MockableTaskRunnerDependencies {
		deps := &MockableTaskRunnerDependencies{

			// Configure the fake experiment
			Experiment: &mocks.Experiment{
				MockKibiBytesReceived: func() float64 {
					return 10
				},
				MockKibiBytesSent: func() float64 {
					return 4
				},
				MockOpenReportContext: func(ctx context.Context) error {
					return nil
				},
				MockReportID: func() string {
					return "20211202T074907Z_example_IT_30722_n1_axDLHNUfJaV1IbuU"
				},
				MockMeasureWithContext: func(ctx context.Context, target model.ExperimentTarget) (*model.Measurement, error) {
					return &model.Measurement{}, nil
				},
				MockSubmitAndUpdateMeasurementContext: func(ctx context.Context, measurement *model.Measurement) error {
					return nil
				},
			},

			// Configure the fake experiment builder
			Builder: &mocks.ExperimentBuilder{
				MockSetCallbacks: func(callbacks model.ExperimentCallbacks) {},
				MockInputPolicy: func() model.InputPolicy {
					return model.InputNone
				},
				MockInterruptible: func() bool {
					return false
				},
			},

			// Configure the fake session
			Session: &mocks.Session{
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
			},
		}

		// The fake session MUST return the fake experiment builder
		deps.Session.MockNewExperimentBuilder = func(name string) (model.ExperimentBuilder, error) {
			return deps.Builder, nil
		}

		// The fake experiment builder MUST return the fake target loader
		deps.Builder.MockNewTargetLoader = func(config *model.ExperimentTargetLoaderConfig) model.ExperimentTargetLoader {
			return deps.Loader
		}

		// The fake experiment builder MUST return the fake experiment
		deps.Builder.MockNewExperiment = func() model.Experiment {
			return deps.Experiment
		}

		return deps
	}

	assertReducedEventsLike := func(t *testing.T, expected, got []eventKeyCount) {
		if diff := cmp.Diff(expected, got); diff != "" {
			t.Fatal(diff)
		}
	}

	t.Run("with invalid experiment name", func(t *testing.T) {
		runner, emitter := newRunnerForTesting()
		fake := fakeSuccessfulDeps()
		fake.Session.MockNewExperimentBuilder = func(name string) (model.ExperimentBuilder, error) {
			return nil, errors.New("invalid experiment name")
		}
		runner.newSession = fake.NewSession
		events := runAndCollect(runner, emitter)
		reduced := reduceEventsKeysIgnoreLog(t, events)
		expect := []eventKeyCount{
			{Key: eventTypeStatusQueued, Count: 1},
			{Key: eventTypeStatusStarted, Count: 1},
			{Key: eventTypeFailureStartup, Count: 1},
			{Key: eventTypeStatusEnd, Count: 1},
		}
		assertReducedEventsLike(t, expect, reduced)
	})

	t.Run("with error during backends lookup", func(t *testing.T) {
		runner, emitter := newRunnerForTesting()
		fake := fakeSuccessfulDeps()
		fake.Session.MockMaybeLookupBackendsContext = func(ctx context.Context) error {
			return errors.New("mocked error")
		}
		runner.newSession = fake.NewSession
		events := runAndCollect(runner, emitter)
		reduced := reduceEventsKeysIgnoreLog(t, events)
		expect := []eventKeyCount{
			{Key: eventTypeStatusQueued, Count: 1},
			{Key: eventTypeStatusStarted, Count: 1},
			{Key: eventTypeFailureStartup, Count: 1},
			{Key: eventTypeStatusEnd, Count: 1},
		}
		assertReducedEventsLike(t, expect, reduced)
	})

	t.Run("with error during location lookup", func(t *testing.T) {
		runner, emitter := newRunnerForTesting()
		fake := fakeSuccessfulDeps()
		fake.Session.MockMaybeLookupLocationContext = func(ctx context.Context) error {
			return errors.New("mocked error")
		}
		runner.newSession = fake.NewSession
		events := runAndCollect(runner, emitter)
		reduced := reduceEventsKeysIgnoreLog(t, events)
		expect := []eventKeyCount{
			{Key: eventTypeStatusQueued, Count: 1},
			{Key: eventTypeStatusStarted, Count: 1},
			{Key: eventTypeStatusProgress, Count: 1},
			{Key: eventTypeFailureIPLookup, Count: 1},
			{Key: eventTypeFailureASNLookup, Count: 1},
			{Key: eventTypeFailureCCLookup, Count: 1},
			{Key: eventTypeFailureResolverLookup, Count: 1},
			{Key: eventTypeStatusEnd, Count: 1},
		}
		assertReducedEventsLike(t, expect, reduced)
	})

	t.Run("with missing input and InputOrQueryBackend policy", func(t *testing.T) {
		runner, emitter := newRunnerForTesting()
		fake := fakeSuccessfulDeps()
		fake.Builder.MockInputPolicy = func() model.InputPolicy {
			return model.InputOrQueryBackend
		}
		runner.newSession = fake.NewSession
		events := runAndCollect(runner, emitter)
		reduced := reduceEventsKeysIgnoreLog(t, events)
		expect := []eventKeyCount{
			{Key: eventTypeStatusQueued, Count: 1},
			{Key: eventTypeStatusStarted, Count: 1},
			{Key: eventTypeStatusProgress, Count: 3},
			{Key: eventTypeStatusGeoIPLookup, Count: 1},
			{Key: eventTypeStatusResolverLookup, Count: 1},
			{Key: eventTypeFailureStartup, Count: 1},
			{Key: eventTypeStatusEnd, Count: 1},
		}
		assertReducedEventsLike(t, expect, reduced)
	})

	t.Run("with missing input and InputStrictlyRequired policy", func(t *testing.T) {
		runner, emitter := newRunnerForTesting()
		fake := fakeSuccessfulDeps()
		fake.Builder.MockInputPolicy = func() model.InputPolicy {
			return model.InputStrictlyRequired
		}
		runner.newSession = fake.NewSession
		events := runAndCollect(runner, emitter)
		reduced := reduceEventsKeysIgnoreLog(t, events)
		expect := []eventKeyCount{
			{Key: eventTypeStatusQueued, Count: 1},
			{Key: eventTypeStatusStarted, Count: 1},
			{Key: eventTypeStatusProgress, Count: 3},
			{Key: eventTypeStatusGeoIPLookup, Count: 1},
			{Key: eventTypeStatusResolverLookup, Count: 1},
			{Key: eventTypeFailureStartup, Count: 1},
			{Key: eventTypeStatusEnd, Count: 1},
		}
		assertReducedEventsLike(t, expect, reduced)
	})

	t.Run("with InputOrStaticDefault policy and experiment with no static input",
		func(t *testing.T) {
			runner, emitter := newRunnerForTesting()
			runner.settings.Name = "Antani" // no input for this experiment
			fake := fakeSuccessfulDeps()
			fake.Builder.MockInputPolicy = func() model.InputPolicy {
				return model.InputOrStaticDefault
			}
			runner.newSession = fake.NewSession
			events := runAndCollect(runner, emitter)
			reduced := reduceEventsKeysIgnoreLog(t, events)
			expect := []eventKeyCount{
				{Key: eventTypeStatusQueued, Count: 1},
				{Key: eventTypeStatusStarted, Count: 1},
				{Key: eventTypeStatusProgress, Count: 3},
				{Key: eventTypeStatusGeoIPLookup, Count: 1},
				{Key: eventTypeStatusResolverLookup, Count: 1},
				{Key: eventTypeFailureStartup, Count: 1},
				{Key: eventTypeStatusEnd, Count: 1},
			}
			assertReducedEventsLike(t, expect, reduced)
		})

	t.Run("with InputNone policy and provided input", func(t *testing.T) {
		runner, emitter := newRunnerForTesting()
		runner.settings.Inputs = append(runner.settings.Inputs, "https://x.org/")
		fake := fakeSuccessfulDeps()
		fake.Builder.MockInputPolicy = func() model.InputPolicy {
			return model.InputNone
		}
		runner.newSession = fake.NewSession
		events := runAndCollect(runner, emitter)
		reduced := reduceEventsKeysIgnoreLog(t, events)
		expect := []eventKeyCount{
			{Key: eventTypeStatusQueued, Count: 1},
			{Key: eventTypeStatusStarted, Count: 1},
			{Key: eventTypeStatusProgress, Count: 3},
			{Key: eventTypeStatusGeoIPLookup, Count: 1},
			{Key: eventTypeStatusResolverLookup, Count: 1},
			{Key: eventTypeFailureStartup, Count: 1},
			{Key: eventTypeStatusEnd, Count: 1},
		}
		assertReducedEventsLike(t, expect, reduced)
	})

	t.Run("with failure opening report", func(t *testing.T) {
		runner, emitter := newRunnerForTesting()
		fake := fakeSuccessfulDeps()
		fake.Experiment.MockOpenReportContext = func(ctx context.Context) error {
			return errors.New("mocked error")
		}
		runner.newSession = fake.NewSession
		events := runAndCollect(runner, emitter)
		reduced := reduceEventsKeysIgnoreLog(t, events)
		expect := []eventKeyCount{
			{Key: eventTypeStatusQueued, Count: 1},
			{Key: eventTypeStatusStarted, Count: 1},
			{Key: eventTypeStatusProgress, Count: 3},
			{Key: eventTypeStatusGeoIPLookup, Count: 1},
			{Key: eventTypeStatusResolverLookup, Count: 1},
			{Key: eventTypeFailureReportCreate, Count: 1},
			{Key: eventTypeStatusEnd, Count: 1},
		}
		assertReducedEventsLike(t, expect, reduced)
	})

	t.Run("with success and InputNone policy", func(t *testing.T) {
		runner, emitter := newRunnerForTesting()
		fake := fakeSuccessfulDeps()
		fake.Builder.MockInputPolicy = func() model.InputPolicy {
			return model.InputNone
		}
		runner.newSession = fake.NewSession
		events := runAndCollect(runner, emitter)
		reduced := reduceEventsKeysIgnoreLog(t, events)
		expect := []eventKeyCount{
			{Key: eventTypeStatusQueued, Count: 1},
			{Key: eventTypeStatusStarted, Count: 1},
			{Key: eventTypeStatusProgress, Count: 3},
			{Key: eventTypeStatusGeoIPLookup, Count: 1},
			{Key: eventTypeStatusResolverLookup, Count: 1},
			{Key: eventTypeStatusProgress, Count: 1},
			{Key: eventTypeStatusReportCreate, Count: 1},
			{Key: eventTypeStatusMeasurementStart, Count: 1},
			{Key: eventTypeMeasurement, Count: 1},
			{Key: eventTypeStatusMeasurementSubmission, Count: 1},
			{Key: eventTypeStatusMeasurementDone, Count: 1},
			{Key: eventTypeStatusEnd, Count: 1},
		}
		assertReducedEventsLike(t, expect, reduced)
	})

	t.Run("with measurement failure and InputNone policy", func(t *testing.T) {
		runner, emitter := newRunnerForTesting()
		fake := fakeSuccessfulDeps()
		fake.Builder.MockInputPolicy = func() model.InputPolicy {
			return model.InputNone
		}
		fake.Experiment.MockMeasureWithContext = func(ctx context.Context, target model.ExperimentTarget) (measurement *model.Measurement, err error) {
			return nil, errors.New("preconditions error")
		}
		runner.newSession = fake.NewSession
		events := runAndCollect(runner, emitter)
		reduced := reduceEventsKeysIgnoreLog(t, events)
		expect := []eventKeyCount{
			{Key: eventTypeStatusQueued, Count: 1},
			{Key: eventTypeStatusStarted, Count: 1},
			{Key: eventTypeStatusProgress, Count: 3},
			{Key: eventTypeStatusGeoIPLookup, Count: 1},
			{Key: eventTypeStatusResolverLookup, Count: 1},
			{Key: eventTypeStatusProgress, Count: 1},
			{Key: eventTypeStatusReportCreate, Count: 1},
			{Key: eventTypeStatusMeasurementStart, Count: 1},
			{Key: eventTypeFailureMeasurement, Count: 1},
			{Key: eventTypeStatusEnd, Count: 1},
		}
		assertReducedEventsLike(t, expect, reduced)
	})

	t.Run("with measurement failure and annotations", func(t *testing.T) {
		// See https://github.com/ooni/probe/issues/2173. We want to be sure that
		// we are not crashing when the measurement fails and there are annotations,
		// which is what was happening in the above referenced issue.
		runner, emitter := newRunnerForTesting()
		fake := fakeSuccessfulDeps()
		fake.Builder.MockInputPolicy = func() model.InputPolicy {
			return model.InputNone
		}
		fake.Experiment.MockMeasureWithContext = func(ctx context.Context, target model.ExperimentTarget) (measurement *model.Measurement, err error) {
			return nil, errors.New("preconditions error")
		}
		runner.newSession = fake.NewSession
		runner.settings.Annotations = map[string]string{
			"architecture": "arm64",
		}
		events := runAndCollect(runner, emitter)
		reduced := reduceEventsKeysIgnoreLog(t, events)
		expect := []eventKeyCount{
			{Key: eventTypeStatusQueued, Count: 1},
			{Key: eventTypeStatusStarted, Count: 1},
			{Key: eventTypeStatusProgress, Count: 3},
			{Key: eventTypeStatusGeoIPLookup, Count: 1},
			{Key: eventTypeStatusResolverLookup, Count: 1},
			{Key: eventTypeStatusProgress, Count: 1},
			{Key: eventTypeStatusReportCreate, Count: 1},
			{Key: eventTypeStatusMeasurementStart, Count: 1},
			{Key: eventTypeFailureMeasurement, Count: 1},
			{Key: eventTypeStatusEnd, Count: 1},
		}
		assertReducedEventsLike(t, expect, reduced)
	})

	t.Run("with success and InputStrictlyRequired", func(t *testing.T) {
		runner, emitter := newRunnerForTesting()
		runner.settings.Inputs = []string{"a", "b", "c", "d"}
		fake := fakeSuccessfulDeps()
		fake.Builder.MockInputPolicy = func() model.InputPolicy {
			return model.InputStrictlyRequired
		}
		runner.newSession = fake.NewSession
		events := runAndCollect(runner, emitter)
		reduced := reduceEventsKeysIgnoreLog(t, events)
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

	t.Run("with success and InputOptional and input", func(t *testing.T) {
		runner, emitter := newRunnerForTesting()
		runner.settings.Inputs = []string{"a", "b", "c", "d"}
		fake := fakeSuccessfulDeps()
		fake.Builder.MockInputPolicy = func() model.InputPolicy {
			return model.InputOptional
		}
		runner.newSession = fake.NewSession
		events := runAndCollect(runner, emitter)
		reduced := reduceEventsKeysIgnoreLog(t, events)
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

	t.Run("with success and InputOptional and no input", func(t *testing.T) {
		runner, emitter := newRunnerForTesting()
		fake := fakeSuccessfulDeps()
		fake.Builder.MockInputPolicy = func() model.InputPolicy {
			return model.InputOptional
		}
		runner.newSession = fake.NewSession
		events := runAndCollect(runner, emitter)
		reduced := reduceEventsKeysIgnoreLog(t, events)
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
			{Key: eventTypeMeasurement, Count: 1},
			{Key: eventTypeStatusMeasurementSubmission, Count: 1},
			{Key: eventTypeStatusMeasurementDone, Count: 1},
			//
			{Key: eventTypeStatusEnd, Count: 1},
		}
		assertReducedEventsLike(t, expect, reduced)
	})

	t.Run("with success and InputOrStaticDefault", func(t *testing.T) {
		experimentName := "DNSCheck"
		runner, emitter := newRunnerForTesting()
		runner.settings.Name = experimentName
		fake := fakeSuccessfulDeps()
		fake.Builder.MockInputPolicy = func() model.InputPolicy {
			return model.InputOrStaticDefault
		}
		runner.newSession = fake.NewSession
		events := runAndCollect(runner, emitter)
		reduced := reduceEventsKeysIgnoreLog(t, events)
		expect := []eventKeyCount{
			{Key: eventTypeStatusQueued, Count: 1},
			{Key: eventTypeStatusStarted, Count: 1},
			{Key: eventTypeStatusProgress, Count: 3},
			{Key: eventTypeStatusGeoIPLookup, Count: 1},
			{Key: eventTypeStatusResolverLookup, Count: 1},
			{Key: eventTypeStatusProgress, Count: 1},
			{Key: eventTypeStatusReportCreate, Count: 1},
		}
		allEntries, err := targetloading.StaticBareInputForExperiment(experimentName)
		if err != nil {
			t.Fatal(err)
		}
		// write the correct entries for each expected measurement.
		for idx := 0; idx < len(allEntries); idx++ {
			expect = append(expect, eventKeyCount{Key: eventTypeStatusMeasurementStart, Count: 1})
			expect = append(expect, eventKeyCount{Key: eventTypeStatusProgress, Count: 1})
			expect = append(expect, eventKeyCount{Key: eventTypeMeasurement, Count: 1})
			expect = append(expect, eventKeyCount{Key: eventTypeStatusMeasurementSubmission, Count: 1})
			expect = append(expect, eventKeyCount{Key: eventTypeStatusMeasurementDone, Count: 1})
		}
		expect = append(expect, eventKeyCount{Key: eventTypeStatusEnd, Count: 1})
		assertReducedEventsLike(t, expect, reduced)
	})

	t.Run("with success and max runtime", func(t *testing.T) {
		runner, emitter := newRunnerForTesting()
		runner.settings.Inputs = []string{"a", "b", "c", "d"}
		runner.settings.Options.MaxRuntime = 2
		fake := fakeSuccessfulDeps()
		fake.Builder.MockInputPolicy = func() model.InputPolicy {
			return model.InputStrictlyRequired
		}
		fake.Experiment.MockMeasureWithContext = func(ctx context.Context, target model.ExperimentTarget) (measurement *model.Measurement, err error) {
			time.Sleep(1 * time.Second)
			return &model.Measurement{}, nil
		}
		runner.newSession = fake.NewSession
		events := runAndCollect(runner, emitter)
		reduced := reduceEventsKeysIgnoreLog(t, events)
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
		fake := fakeSuccessfulDeps()
		fake.Builder.MockInputPolicy = func() model.InputPolicy {
			return model.InputStrictlyRequired
		}
		fake.Builder.MockInterruptible = func() bool {
			return true
		}
		ctx, cancel := context.WithCancel(context.Background())
		fake.Experiment.MockMeasureWithContext = func(ctx context.Context, target model.ExperimentTarget) (measurement *model.Measurement, err error) {
			cancel()
			return &model.Measurement{}, nil
		}
		runner.newSession = fake.NewSession
		events := runAndCollectContext(ctx, runner, emitter)
		reduced := reduceEventsKeysIgnoreLog(t, events)
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
		fake := fakeSuccessfulDeps()
		fake.Builder.MockInputPolicy = func() model.InputPolicy {
			return model.InputStrictlyRequired
		}
		fake.Experiment.MockSubmitAndUpdateMeasurementContext = func(ctx context.Context, measurement *model.Measurement) error {
			return errors.New("cannot submit")
		}
		runner.newSession = fake.NewSession
		events := runAndCollect(runner, emitter)
		reduced := reduceEventsKeysIgnoreLog(t, events)
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
		fake := fakeSuccessfulDeps()
		var callbacks model.ExperimentCallbacks
		fake.Builder.MockSetCallbacks = func(cbs model.ExperimentCallbacks) {
			callbacks = cbs
		}
		fake.Experiment.MockMeasureWithContext = func(ctx context.Context, target model.ExperimentTarget) (measurement *model.Measurement, err error) {
			callbacks.OnProgress(1, "hello from the fake experiment")
			return &model.Measurement{}, nil
		}
		runner.newSession = fake.NewSession
		events := runAndCollect(runner, emitter)
		reduced := reduceEventsKeysIgnoreLog(t, events)
		expect := []eventKeyCount{
			{Key: eventTypeStatusQueued, Count: 1},
			{Key: eventTypeStatusStarted, Count: 1},
			{Key: eventTypeStatusProgress, Count: 3},
			{Key: eventTypeStatusGeoIPLookup, Count: 1},
			{Key: eventTypeStatusResolverLookup, Count: 1},
			{Key: eventTypeStatusProgress, Count: 1},
			{Key: eventTypeStatusReportCreate, Count: 1},
			{Key: eventTypeStatusMeasurementStart, Count: 1},
			{Key: eventTypeStatusProgress, Count: 1},
			{Key: eventTypeMeasurement, Count: 1},
			{Key: eventTypeStatusMeasurementSubmission, Count: 1},
			{Key: eventTypeStatusMeasurementDone, Count: 1},
			{Key: eventTypeStatusEnd, Count: 1},
		}
		assertReducedEventsLike(t, expect, reduced)
	})
}
