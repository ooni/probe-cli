package mocks

import (
	"errors"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestSaver(t *testing.T) {
	t.Run("SaveMeasurement", func(t *testing.T) {
		expected := errors.New("mocked error")
		s := &Saver{
			MockSaveMeasurement: func(m *model.Measurement) error {
				return expected
			},
		}
		err := s.SaveMeasurement(&model.Measurement{})
		if !errors.Is(err, expected) {
			t.Fatal("unexpected err", err)
		}
	})
}
