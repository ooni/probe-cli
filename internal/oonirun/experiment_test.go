package oonirun

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

func TestExperimentRunWithFailureToSubmitAndShuffle(t *testing.T) {
	shuffledInputsPrev := experimentShuffledInputs.Load()
	var calledSetOptionsAny int
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
		Inputs: []string{
			"a", "b", "c",
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
					MockSetOptionsAny: func(options map[string]any) error {
						calledSetOptionsAny++
						return nil
					},
					MockNewRicherInputExperiment: func() model.RicherInputExperiment {
						exp := &mocks.RicherInputExperiment{
							MockMeasure: func(ctx context.Context, input model.RicherInput) (*model.Measurement, error) {
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
					MockBuildRicherInput: func(annotations map[string]string, flatInputs []string) (output []model.RicherInput) {
						for _, input := range flatInputs {
							output = append(output, model.RicherInput{
								Annotations: annotations,
								Input:       input,
							})
						}
						return
					},
				}
				return eb, nil
			},
			MockLogger: func() model.Logger {
				return model.DiscardLogger
			},
		},
		newExperimentBuilderFn: nil,
		newInputLoaderFn:       nil,
		newSubmitterFn: func(ctx context.Context) (model.Submitter, error) {
			subm := &mocks.Submitter{
				MockSubmit: func(ctx context.Context, m *model.Measurement) error {
					failedToSubmit++
					return errors.New("mocked error")
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
	if calledKibiBytesReceived < 1 {
		t.Fatal("did not call KibiBytesReceived")
	}
	if calledKibiBytesSent < 1 {
		t.Fatal("did not call KibiBytesSent")
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
		newInputLoaderFn       func(inputPolicy model.InputPolicy) inputLoader
		newSubmitterFn         func(ctx context.Context) (model.Submitter, error)
		newSaverFn             func(experiment model.RicherInputExperiment) (model.Saver, error)
		newInputProcessorFn    func(experiment model.RicherInputExperiment, inputList []model.RicherInput, saver model.Saver, submitter model.Submitter) inputProcessor
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
		name: "cannot load input",
		fields: fields{
			newExperimentBuilderFn: func(experimentName string) (model.ExperimentBuilder, error) {
				eb := &mocks.ExperimentBuilder{
					MockInputPolicy: func() model.InputPolicy {
						return model.InputOptional
					},
				}
				return eb, nil
			},
			newInputLoaderFn: func(inputPolicy model.InputPolicy) inputLoader {
				return &mocks.ExperimentInputLoader{
					MockLoad: func(ctx context.Context) ([]model.OOAPIURLInfo, error) {
						return nil, errMocked
					},
				}
			},
		},
		args:      args{},
		expectErr: errMocked,
	}, {
		name: "cannot set options",
		fields: fields{
			newExperimentBuilderFn: func(experimentName string) (model.ExperimentBuilder, error) {
				eb := &mocks.ExperimentBuilder{
					MockInputPolicy: func() model.InputPolicy {
						return model.InputOptional
					},
					MockSetOptionsAny: func(options map[string]any) error {
						return errMocked
					},
				}
				return eb, nil
			},
			newInputLoaderFn: func(inputPolicy model.InputPolicy) inputLoader {
				return &mocks.ExperimentInputLoader{
					MockLoad: func(ctx context.Context) ([]model.OOAPIURLInfo, error) {
						return []model.OOAPIURLInfo{}, nil
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
			newInputLoaderFn: func(inputPolicy model.InputPolicy) inputLoader {
				return &mocks.ExperimentInputLoader{
					MockLoad: func(ctx context.Context) ([]model.OOAPIURLInfo, error) {
						return []model.OOAPIURLInfo{}, nil
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
			newInputLoaderFn: func(inputPolicy model.InputPolicy) inputLoader {
				return &mocks.ExperimentInputLoader{
					MockLoad: func(ctx context.Context) ([]model.OOAPIURLInfo, error) {
						return []model.OOAPIURLInfo{}, nil
					},
				}
			},
			newSubmitterFn: func(ctx context.Context) (model.Submitter, error) {
				return &mocks.Submitter{}, nil
			},
			newSaverFn: func(experiment model.RicherInputExperiment) (model.Saver, error) {
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
			newInputLoaderFn: func(inputPolicy model.InputPolicy) inputLoader {
				return &mocks.ExperimentInputLoader{
					MockLoad: func(ctx context.Context) ([]model.OOAPIURLInfo, error) {
						return []model.OOAPIURLInfo{}, nil
					},
				}
			},
			newSubmitterFn: func(ctx context.Context) (model.Submitter, error) {
				return &mocks.Submitter{}, nil
			},
			newSaverFn: func(experiment model.RicherInputExperiment) (model.Saver, error) {
				return &mocks.Saver{}, nil
			},
			newInputProcessorFn: func(experiment model.RicherInputExperiment, inputList []model.RicherInput,
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
				newInputLoaderFn:       tt.fields.newInputLoaderFn,
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
