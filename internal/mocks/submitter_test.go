package mocks

import (
	"context"
	"errors"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestSubmitter(t *testing.T) {
	t.Run("Submit", func(t *testing.T) {
		expect := errors.New("mocked error")
		s := &Submitter{
			MockSubmit: func(ctx context.Context, m *model.Measurement) error {
				return expect
			},
		}
		err := s.Submit(context.Background(), &model.Measurement{})
		if !errors.Is(err, expect) {
			t.Fatal("unexpected err", err)
		}
	})
}
