package mocks

import (
	"context"
	"errors"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestExperimentInputLoader(t *testing.T) {
	t.Run("Load", func(t *testing.T) {
		expected := errors.New("mocked error")
		eil := &ExperimentInputLoader{
			MockLoad: func(ctx context.Context) ([]model.OOAPIURLInfo, error) {
				return nil, expected
			},
		}
		out, err := eil.Load(context.Background())
		if !errors.Is(err, expected) {
			t.Fatal("unexpected err", err)
		}
		if len(out) > 0 {
			t.Fatal("unexpected length")
		}
	})
}
