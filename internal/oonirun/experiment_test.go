package oonirun

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

func TestExperimentRunWithFailureToSubmitAndShuffle(t *testing.T) {
	shuffledInputsPrev := experimentShuffledInputs.Load()
	var calledSetOptionsAny int
	var calledSetOptionsJSON int
	var failedToSubmit int
	var calledKibiBytesReceived int
	var calledKibiBytesSent int
	ctx := context.Background()
	desc := &Experiment{
		Annotations: map[string]string{
			"platform": "linux",
		},
		ExtraOptions: map[string]any{
			"SleepTime": int64(10 * time.Millisecond),
		},
		InputFilePaths: []string{},
		MaxRuntime:     0,
		Name:           "example",
		NoCollector:    true,
		NoJSON:         true,
		Random:         true, // to test randomness
		ReportFile:     "",
		Session: &mocks.Session{
			MockNewExperimentBuilder: func(name string) (model.ExperimentBuilder, error) {
				eb := &mocks.ExperimentBuilder{
					MockInputPolicy: func() model.InputPolicy {
						return model.InputOptional
					},
					MockSetOptionsJSON: func(value json.RawMessage) error {
						calledSetOptionsJSON++
						return nil
					},
					MockSetOptionsAny: func(options map[string]any) error {
						calledSetOptionsAny++
						return nil
					},
					MockNewExperiment: func() model.Experiment {
						exp := &mocks.Experiment{
							MockMeasureWithContext: func(
								ctx context.Context, target model.ExperimentTarget) (*model.Measurement, error) {
								ff := &testingx.FakeFiller{}
								var meas model.Measurement
								ff.Fill(&meas)
								return &meas, nil
							},
							MockKibiBytesReceived: func() float64 {
								calledKibiBytesReceived++
								return 1.453
							},
							MockKibiBytesSent: func() float64 {
								calledKibiBytesSent++
								return 1.648
							},
						}
						return exp
					},
					MockNewTargetLoader: func(config *model.ExperimentTargetLoaderConfig) model.ExperimentTargetLoader {
						return &mocks.ExperimentTargetLoader{
							MockLoad: func(ctx context.Context) ([]model.ExperimentTarget, error) {
								results := []model.ExperimentTarget{
									model.NewOOAPIURLInfoWithDefaultCategoryAndCountry("a"),
									model.NewOOAPIURLInfoWithDefaultCategoryAndCountry("b"),
									model.NewOOAPIURLInfoWithDefaultCategoryAndCountry("c"),
								}
								return results, nil
							},
						}
					},
				}
				return eb, nil
			},
			MockLogger: func() model.Logger {
				return model.DiscardLogger
			},
		},
		newExperimentBuilderFn: nil,
		newTargetLoaderFn:      nil,
		newSubmitterFn: func(ctx context.Context) (model.Submitter, error) {
			subm := &mocks.Submitter{
				MockSubmit: func(ctx context.Context, m *model.Measurement) (string, error) {
					failedToSubmit++
					return "", errors.New("mocked error")
				},
			}
			return subm, nil
		},
		newSaverFn:          nil,
		newInputProcessorFn: nil,
	}
	if err := desc.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if failedToSubmit < 1 {
		t.Fatal("expected to see failure to submit")
	}
	if experimentShuffledInputs.Load() != shuffledInputsPrev+1 {
		t.Fatal("did not shuffle inputs")
	}
	if calledSetOptionsAny < 1 {
		t.Fatal("should have called SetOptionsAny")
	}
	if calledSetOptionsJSON < 1 {
		t.Fatal("should have called SetOptionsJSON")
	}
	if calledKibiBytesReceived < 1 {
		t.Fatal("did not call KibiBytesReceived")
	}
	if calledKibiBytesSent < 1 {
		t.Fatal("did not call KibiBytesSent")
	}
}

// This test ensures that we honour InitialOptions then ExtraOptions.
func TestExperimentSetOptions(t *testing.T) {

	// create the Experiment we're using for this test
	exp := &Experiment{
		ExtraOptions: map[string]any{
			"Message": "jarjarbinks",
		},
		InitialOptions: []byte(`{"Message": "foobar", "ReturnError": true}`),
		Name:           "example",

		// TODO(bassosimone): A zero-value session works here. The proper change
		// however would be to write a engine.NewExperimentBuilder factory that takes
		// as input an interface for the session. This would help testing.
		Session: &engine.Session{},
	}

	// create the experiment builder manually
	builder, err := exp.newExperimentBuilder(exp.Name)
	if err != nil {
		t.Fatal(err)
	}

	// invoke the method we're testing
	if err := exp.setOptions(builder); err != nil {
		t.Fatal(err)
	}

	// obtain the options
	options, err := builder.Options()
	if err != nil {
		t.Fatal(err)
	}

	// describe what we expect to happen
	//
	// we basically want ExtraOptions to override InitialOptions
	expect := map[string]model.ExperimentOptionInfo{
		"Message": {
			Doc:   "Message to emit at test completion",
			Type:  "string",
			Value: string("jarjarbinks"), // set by ExtraOptions
		},
		"ReturnError": {
			Doc:   "Toogle to return a mocked error",
			Type:  "bool",
			Value: bool(true), // set by InitialOptions
		},
		"SleepTime": {
			Doc:   "Amount of time to sleep for in nanosecond",
			Type:  "int64",
			Value: int64(1000000000), // still the default nonzero value
		},
	}

	// make sure the result equals expectation
	if diff := cmp.Diff(expect, options); diff != "" {
		t.Fatal(diff)
	}
}

func TestExperimentRun(t *testing.T) {
	errMocked := errors.New("mocked error")
	type fields struct {
		Annotations            map[string]string
		ExtraOptions           map[string]any
		Inputs                 []string
		InputFilePaths         []string
		MaxRuntime             int64
		Name                   string
		NoCollector            bool
		NoJSON                 bool
		Random                 bool
		ReportFile             string
		Session                Session
		newExperimentBuilderFn func(experimentName string) (model.ExperimentBuilder, error)
		newTargetLoaderFn      func(builder model.ExperimentBuilder) targetLoader
		newSubmitterFn         func(ctx context.Context) (model.Submitter, error)
		newSaverFn             func() (model.Saver, error)
		newInputProcessorFn    func(experiment model.Experiment,
			inputList []model.ExperimentTarget, saver model.Saver, submitter model.Submitter) inputProcessor
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		expectErr error
	}{{
		name: "cannot construct an experiment builder",
		fields: fields{
			newExperimentBuilderFn: func(experimentName string) (model.ExperimentBuilder, error) {
				return nil, errMocked
			},
		},
		args:      args{},
		expectErr: errMocked,
	}, {
		name: "cannot set InitialOptions",
		fields: fields{
			newExperimentBuilderFn: func(experimentName string) (model.ExperimentBuilder, error) {
				eb := &mocks.ExperimentBuilder{
					MockSetOptionsJSON: func(value json.RawMessage) error {
						return errMocked
					},
				}
				return eb, nil
			},
			newTargetLoaderFn: func(builder model.ExperimentBuilder) targetLoader {
				return &mocks.ExperimentTargetLoader{
					MockLoad: func(ctx context.Context) ([]model.ExperimentTarget, error) {
						return []model.ExperimentTarget{}, nil
					},
				}
			},
		},
		args:      args{},
		expectErr: errMocked,
	}, {
		name: "cannot set ExtraOptions",
		fields: fields{
			newExperimentBuilderFn: func(experimentName string) (model.ExperimentBuilder, error) {
				eb := &mocks.ExperimentBuilder{
					MockSetOptionsJSON: func(value json.RawMessage) error {
						return nil
					},
					MockSetOptionsAny: func(options map[string]any) error {
						return errMocked
					},
				}
				return eb, nil
			},
			newTargetLoaderFn: func(builder model.ExperimentBuilder) targetLoader {
				return &mocks.ExperimentTargetLoader{
					MockLoad: func(ctx context.Context) ([]model.ExperimentTarget, error) {
						return []model.ExperimentTarget{}, nil
					},
				}
			},
		},
		args:      args{},
		expectErr: errMocked,
	}, {
		name: "cannot load input",
		fields: fields{
			newExperimentBuilderFn: func(experimentName string) (model.ExperimentBuilder, error) {
				eb := &mocks.ExperimentBuilder{
					MockInputPolicy: func() model.InputPolicy {
						return model.InputOptional
					},
					MockSetOptionsJSON: func(value json.RawMessage) error {
						return nil
					},
					MockSetOptionsAny: func(options map[string]any) error {
						return nil
					},
				}
				return eb, nil
			},
			newTargetLoaderFn: func(builder model.ExperimentBuilder) targetLoader {
				return &mocks.ExperimentTargetLoader{
					MockLoad: func(ctx context.Context) ([]model.ExperimentTarget, error) {
						return nil, errMocked
					},
				}
			},
		},
		args:      args{},
		expectErr: errMocked,
	}, {
		name: "cannot create new submitter",
		fields: fields{
			Session: &mocks.Session{
				MockLogger: func() model.Logger {
					return model.DiscardLogger
				},
			},
			newExperimentBuilderFn: func(experimentName string) (model.ExperimentBuilder, error) {
				eb := &mocks.ExperimentBuilder{
					MockInputPolicy: func() model.InputPolicy {
						return model.InputOptional
					},
					MockSetOptionsJSON: func(value json.RawMessage) error {
						return nil
					},
					MockSetOptionsAny: func(options map[string]any) error {
						return nil
					},
					MockNewExperiment: func() model.Experiment {
						exp := &mocks.Experiment{
							MockKibiBytesReceived: func() float64 {
								return 0
							},
							MockKibiBytesSent: func() float64 {
								return 0
							},
						}
						return exp
					},
				}
				return eb, nil
			},
			newTargetLoaderFn: func(builder model.ExperimentBuilder) targetLoader {
				return &mocks.ExperimentTargetLoader{
					MockLoad: func(ctx context.Context) ([]model.ExperimentTarget, error) {
						return []model.ExperimentTarget{}, nil
					},
				}
			},
			newSubmitterFn: func(ctx context.Context) (model.Submitter, error) {
				return nil, errMocked
			},
		},
		args:      args{},
		expectErr: errMocked,
	}, {
		name: "cannot create new saver",
		fields: fields{
			Session: &mocks.Session{
				MockLogger: func() model.Logger {
					return model.DiscardLogger
				},
			},
			newExperimentBuilderFn: func(experimentName string) (model.ExperimentBuilder, error) {
				eb := &mocks.ExperimentBuilder{
					MockInputPolicy: func() model.InputPolicy {
						return model.InputOptional
					},
					MockSetOptionsJSON: func(value json.RawMessage) error {
						return nil
					},
					MockSetOptionsAny: func(options map[string]any) error {
						return nil
					},
					MockNewExperiment: func() model.Experiment {
						exp := &mocks.Experiment{
							MockKibiBytesReceived: func() float64 {
								return 0
							},
							MockKibiBytesSent: func() float64 {
								return 0
							},
						}
						return exp
					},
				}
				return eb, nil
			},
			newTargetLoaderFn: func(builder model.ExperimentBuilder) targetLoader {
				return &mocks.ExperimentTargetLoader{
					MockLoad: func(ctx context.Context) ([]model.ExperimentTarget, error) {
						return []model.ExperimentTarget{}, nil
					},
				}
			},
			newSubmitterFn: func(ctx context.Context) (model.Submitter, error) {
				return &mocks.Submitter{}, nil
			},
			newSaverFn: func() (model.Saver, error) {
				return nil, errMocked
			},
		},
		args:      args{},
		expectErr: errMocked,
	}, {
		name: "input processor fails",
		fields: fields{
			Session: &mocks.Session{
				MockLogger: func() model.Logger {
					return model.DiscardLogger
				},
			},
			newExperimentBuilderFn: func(experimentName string) (model.ExperimentBuilder, error) {
				eb := &mocks.ExperimentBuilder{
					MockInputPolicy: func() model.InputPolicy {
						return model.InputOptional
					},
					MockSetOptionsJSON: func(value json.RawMessage) error {
						return nil
					},
					MockSetOptionsAny: func(options map[string]any) error {
						return nil
					},
					MockNewExperiment: func() model.Experiment {
						exp := &mocks.Experiment{
							MockKibiBytesReceived: func() float64 {
								return 0
							},
							MockKibiBytesSent: func() float64 {
								return 0
							},
						}
						return exp
					},
				}
				return eb, nil
			},
			newTargetLoaderFn: func(builder model.ExperimentBuilder) targetLoader {
				return &mocks.ExperimentTargetLoader{
					MockLoad: func(ctx context.Context) ([]model.ExperimentTarget, error) {
						return []model.ExperimentTarget{}, nil
					},
				}
			},
			newSubmitterFn: func(ctx context.Context) (model.Submitter, error) {
				return &mocks.Submitter{}, nil
			},
			newSaverFn: func() (model.Saver, error) {
				return &mocks.Saver{}, nil
			},
			newInputProcessorFn: func(experiment model.Experiment, inputList []model.ExperimentTarget,
				saver model.Saver, submitter model.Submitter) inputProcessor {
				return &mocks.ExperimentInputProcessor{
					MockRun: func(ctx context.Context) error {
						return errMocked
					},
				}
			},
		},
		args:      args{},
		expectErr: errMocked,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ed := &Experiment{
				Annotations:            tt.fields.Annotations,
				ExtraOptions:           tt.fields.ExtraOptions,
				Inputs:                 tt.fields.Inputs,
				InputFilePaths:         tt.fields.InputFilePaths,
				MaxRuntime:             tt.fields.MaxRuntime,
				Name:                   tt.fields.Name,
				NoCollector:            tt.fields.NoCollector,
				NoJSON:                 tt.fields.NoJSON,
				Random:                 tt.fields.Random,
				ReportFile:             tt.fields.ReportFile,
				Session:                tt.fields.Session,
				newExperimentBuilderFn: tt.fields.newExperimentBuilderFn,
				newTargetLoaderFn:      tt.fields.newTargetLoaderFn,
				newSubmitterFn:         tt.fields.newSubmitterFn,
				newSaverFn:             tt.fields.newSaverFn,
				newInputProcessorFn:    tt.fields.newInputProcessorFn,
			}
			err := ed.Run(tt.args.ctx)
			if !errors.Is(err, tt.expectErr) {
				t.Fatalf("Experiment.Run() error = %v, expectErr %v", err, tt.expectErr)
			}
		})
	}
}
