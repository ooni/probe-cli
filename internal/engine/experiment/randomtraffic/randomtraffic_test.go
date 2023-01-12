package randomtraffic_test

import (
	"context"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/randomtraffic"
	"github.com/ooni/probe-cli/v3/internal/engine/mockable"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestNewExperimentMeasurer(t *testing.T) {
	measurer := randomtraffic.NewExperimentMeasurer(randomtraffic.Config{})
	if measurer.ExperimentName() != "randomtraffic" {
		t.Fatal("unexpected name")
	}
	if measurer.ExperimentVersion() != "0.1.0" {
		t.Fatal("unexpected version")
	}
}

// Tests successful connection
func TestSuccess(t *testing.T) {
	m := randomtraffic.NewExperimentMeasurer(randomtraffic.Config{
		Target: "18.67.76.44:443", // Replace with Edgenet IP when ready
	})
	ctx := context.Background()
	sess := &mockable.Session{MockableLogger: log.Log}
	callbacks := model.NewPrinterCallbacks(sess.Logger())
	measurement := new(model.Measurement)
	err := m.Run(ctx, sess, measurement, callbacks)
	if err != nil {
		t.Fatal(err)
	}
	sk, err := m.GetSummaryKeys(measurement)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := sk.(randomtraffic.SummaryKeys); !ok {
		t.Fatal("invalid type for summary keys")
	}
	tk := measurement.TestKeys.(*randomtraffic.TestKeys)
	if tk.Success != true {
		t.Fatal("invalid Failure")
	}
	if tk.Censorship != false {
		t.Fatal("invalid Censorship")
	}
	t.Logf("%d", tk.ConnectionCount)
}

// Tests connection timeout
func TestTimeout(t *testing.T) {
	m := randomtraffic.NewExperimentMeasurer(randomtraffic.Config{
		Target: "8.8.4.4:1",
	})
	ctx := context.Background()
	sess := &mockable.Session{MockableLogger: log.Log}
	callbacks := model.NewPrinterCallbacks(sess.Logger())
	measurement := new(model.Measurement)
	err := m.Run(ctx, sess, measurement, callbacks)
	if err == nil {
		t.Fatal(err)
	}
	sk, err := m.GetSummaryKeys(measurement)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := sk.(randomtraffic.SummaryKeys); !ok {
		t.Fatal("invalid type for summary keys")
	}
	tk := measurement.TestKeys.(*randomtraffic.TestKeys)
	if tk.Success != false {
		t.Fatal("invalid Failure")
	}
	if tk.Censorship != false {
		t.Fatal("invalid Censorship")
	}
}

// Tests unrelated connection failure
func TestFailure(t *testing.T) {
	m := randomtraffic.NewExperimentMeasurer(randomtraffic.Config{
		Target: "127.0.0.1:443",
	})
	ctx := context.Background()
	sess := &mockable.Session{MockableLogger: log.Log}
	callbacks := model.NewPrinterCallbacks(sess.Logger())
	measurement := new(model.Measurement)
	err := m.Run(ctx, sess, measurement, callbacks)
	if err == nil {
		t.Fatal(err)
	}
	sk, err := m.GetSummaryKeys(measurement)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := sk.(randomtraffic.SummaryKeys); !ok {
		t.Fatal("invalid type for summary keys")
	}
	tk := measurement.TestKeys.(*randomtraffic.TestKeys)
	if tk.Success != false {
		t.Fatal("invalid Failure")
	}
	if tk.Censorship != false {
		t.Fatal("invalid Censorship")
	}
}

func TestSummaryKeysGeneric(t *testing.T) {
	measurement := &model.Measurement{TestKeys: &randomtraffic.TestKeys{}}
	m := &randomtraffic.Measurer{}
	osk, err := m.GetSummaryKeys(measurement)
	if err != nil {
		t.Fatal(err)
	}
	sk := osk.(randomtraffic.SummaryKeys)
	if sk.IsAnomaly {
		t.Fatal("invalid isAnomaly")
	}
}
