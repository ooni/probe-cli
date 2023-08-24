package webconnectivityqa

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netemx"
)

func TestRunTestCase(t *testing.T) {
	t.Run("we detect an unexpected error", func(t *testing.T) {
		tc := &TestCase{
			Name:           "",
			Input:          "",
			Configure:      nil,
			ExpectErr:      false,
			ExpectTestKeys: &testKeys{},
		}
		measurer := &mocks.ExperimentMeasurer{
			MockExperimentName: func() string {
				return "antani"
			},
			MockExperimentVersion: func() string {
				return "0.1.0"
			},
			MockRun: func(ctx context.Context, args *model.ExperimentArgs) error {
				return errors.New("mocked error")
			},
		}
		err := RunTestCase(measurer, tc)
		if err == nil || !strings.HasPrefix(err.Error(), "expected to see no error but got") {
			t.Fatal("unexpected error:", err)
		}
	})

	t.Run("we detect an unexpected success", func(t *testing.T) {
		tc := &TestCase{
			Name:           "",
			Input:          "",
			Configure:      nil,
			ExpectErr:      true,
			ExpectTestKeys: &testKeys{},
		}
		measurer := &mocks.ExperimentMeasurer{
			MockExperimentName: func() string {
				return "antani"
			},
			MockExperimentVersion: func() string {
				return "0.1.0"
			},
			MockRun: func(ctx context.Context, args *model.ExperimentArgs) error {
				return nil
			},
		}
		err := RunTestCase(measurer, tc)
		if err == nil || err.Error() != "expected to see an error but got <nil>" {
			t.Fatal("unexpected error:", err)
		}
	})

	t.Run("we detect a test keys mismatch", func(t *testing.T) {
		tc := &TestCase{
			Name:      "",
			Input:     "",
			Configure: nil,
			ExpectErr: false,
			ExpectTestKeys: &testKeys{
				Accessible: true,
				Blocking:   nil,
			},
		}
		measurer := &mocks.ExperimentMeasurer{
			MockExperimentName: func() string {
				return "antani"
			},
			MockExperimentVersion: func() string {
				return "0.1.0"
			},
			MockRun: func(ctx context.Context, args *model.ExperimentArgs) error {
				args.Measurement.TestKeys = &testKeys{}
				return nil
			},
		}
		err := RunTestCase(measurer, tc)
		if err == nil || !strings.HasPrefix(err.Error(), "test keys mismatch:") {
			t.Fatal("unexpected error:", err)
		}
	})

	t.Run("tc.Configure is called when it is not nil", func(t *testing.T) {
		var called bool
		tc := &TestCase{
			Name:  "",
			Input: "",
			Configure: func(env *netemx.QAEnv) {
				called = true
			},
			ExpectErr: false,
			ExpectTestKeys: &testKeys{
				Accessible: true,
				Blocking:   nil,
			},
		}
		measurer := &mocks.ExperimentMeasurer{
			MockExperimentName: func() string {
				return "antani"
			},
			MockExperimentVersion: func() string {
				return "0.1.0"
			},
			MockRun: func(ctx context.Context, args *model.ExperimentArgs) error {
				args.Measurement.TestKeys = &testKeys{
					Accessible: true,
					Blocking:   nil,
				}
				return nil
			},
		}
		err := RunTestCase(measurer, tc)
		if err != nil {
			t.Fatal(err)
		}
		if !called {
			t.Fatal("did not call tc.Configure")
		}
	})
}
