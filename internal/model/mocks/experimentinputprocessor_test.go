package mocks

import (
	"context"
	"errors"
	"testing"
)

func TestExperimentInputProcessor(t *testing.T) {
	t.Run("Run", func(t *testing.T) {
		expected := errors.New("mocked error")
		eip := &ExperimentInputProcessor{
			MockRun: func(ctx context.Context) error {
				return expected
			},
		}
		err := eip.Run(context.Background())
		if !errors.Is(err, expected) {
			t.Fatal("unexpected result")
		}
	})
}
