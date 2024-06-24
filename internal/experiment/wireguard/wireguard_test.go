package wireguard_test

import (
	"context"
	"errors"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/experiment/example"
	"github.com/ooni/probe-cli/v3/internal/experiment/wireguard"
	"github.com/ooni/probe-cli/v3/internal/legacy/mockable"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestSuccess(t *testing.T) {
	m := wireguard.NewExperimentMeasurer()
	if m.ExperimentName() != "wireguard" {
		t.Fatal("invalid ExperimentName")
	}
	if m.ExperimentVersion() != "0.1.1" {
		t.Fatal("invalid ExperimentVersion")
	}
	ctx := context.Background()
	sess := &mockable.Session{MockableLogger: log.Log}
	callbacks := model.NewPrinterCallbacks(sess.Logger())
	measurement := new(model.Measurement)
	args := &model.ExperimentArgs{
		Callbacks:   callbacks,
		Measurement: measurement,
		Session:     sess,
	}
	err := m.Run(ctx, args)
	if err != nil {
		t.Fatal(err)
	}
}

func TestFailure(t *testing.T) {
	m := wireguard.NewExperimentMeasurer()
	ctx := context.Background()
	sess := &mockable.Session{MockableLogger: log.Log}
	callbacks := model.NewPrinterCallbacks(sess.Logger())
	args := &model.ExperimentArgs{
		Callbacks:   callbacks,
		Measurement: new(model.Measurement),
		Session:     sess,
	}
	err := m.Run(ctx, args)
	if !errors.Is(err, example.ErrFailure) {
		t.Fatal("expected an error here")
	}
}
