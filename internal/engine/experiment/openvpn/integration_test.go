package openvpn_test

import (
	"context"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/openvpn"
	"github.com/ooni/probe-cli/v3/internal/engine/mockable"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestRun(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	/*
		if err != nil {
			// TODO skip if no config
			t.Skip("there is no tor executable installed")
		}
	*/
	t.Log("using this config", "")
	m := openvpn.NewExperimentMeasurer(openvpn.Config{})
	ctx := context.Background()
	measurement := &model.Measurement{}
	callbacks := model.NewPrinterCallbacks(log.Log)
	sess := &mockable.Session{
		MockableLogger: log.Log,
	}
	if err := m.Run(ctx, sess, measurement, callbacks); err != nil {
		t.Fatal(err)
	}
}
