package mocks

import (
	"context"
	"errors"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestExperimentMeasurer(t *testing.T) {
	t.Run("ExperimentName", func(t *testing.T) {
		expect := "antani"
		em := &ExperimentMeasurer{
			MockExperimentName: func() string {
				return expect
			},
		}
		if em.ExperimentName() != expect {
			t.Fatal("unexpected experiment name")
		}
	})

	t.Run("ExperimentVersion", func(t *testing.T) {
		expect := "0.1.0"
		em := &ExperimentMeasurer{
			MockExperimentVersion: func() string {
				return expect
			},
		}
		if em.ExperimentVersion() != expect {
			t.Fatal("unexpected experiment version")
		}
	})

	t.Run("GetSummaryKeys", func(t *testing.T) {
		expect := errors.New("mocked error")
		em := &ExperimentMeasurer{
			MockGetSummaryKeys: func(*model.Measurement) (any, error) {
				return nil, expect
			},
		}
		sk, err := em.GetSummaryKeys(&model.Measurement{})
		if !errors.Is(err, expect) {
			t.Fatal("unexpected error", err)
		}
		if sk != nil {
			t.Fatal("expected nil summary keys")
		}
	})

	t.Run("Run", func(t *testing.T) {
		expect := errors.New("mocked error")
		em := &ExperimentMeasurer{
			MockRun: func(ctx context.Context, args *model.ExperimentArgs) error {
				return expect
			},
		}
		err := em.Run(context.Background(), &model.ExperimentArgs{})
		if !errors.Is(err, expect) {
			t.Fatal("unexpected error", err)
		}
	})
}
