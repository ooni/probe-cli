package httphostheader

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine/mockable"
	"github.com/ooni/probe-cli/v3/internal/model"
)

const (
	softwareName    = "ooniprobe-example"
	softwareVersion = "0.0.1"
)

func TestNewExperimentMeasurer(t *testing.T) {
	measurer := NewExperimentMeasurer(Config{})
	if measurer.ExperimentName() != "http_host_header" {
		t.Fatal("unexpected name")
	}
	if measurer.ExperimentVersion() != "0.3.0" {
		t.Fatal("unexpected version")
	}
}

func TestMeasurerMeasureNoMeasurementInput(t *testing.T) {
	measurer := NewExperimentMeasurer(Config{
		TestHelperURL: "http://www.google.com",
	})
	args := &model.ExperimentArgs{
		Callbacks:   model.NewPrinterCallbacks(log.Log),
		Measurement: &model.Measurement{},
		Session:     newsession(),
	}
	err := measurer.Run(context.Background(), args)
	if err == nil || err.Error() != "experiment requires input" {
		t.Fatal("not the error we expected")
	}
}

func TestMeasurerMeasureNoTestHelper(t *testing.T) {
	measurer := NewExperimentMeasurer(Config{})
	measurement := &model.Measurement{Input: "x.org"}
	args := &model.ExperimentArgs{
		Callbacks:   model.NewPrinterCallbacks(log.Log),
		Measurement: measurement,
		Session:     newsession(),
	}
	err := measurer.Run(context.Background(), args)
	if err != nil {
		t.Fatal(err)
	}
	sk, err := measurer.GetSummaryKeys(measurement)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := sk.(SummaryKeys); !ok {
		t.Fatal("invalid type for summary keys")
	}
}

func TestRunnerHTTPSetHostHeader(t *testing.T) {
	var host string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host = r.Host
		w.WriteHeader(200)
	}))
	defer server.Close()
	measurer := NewExperimentMeasurer(Config{
		TestHelperURL: server.URL,
	})
	measurement := &model.Measurement{
		Input: "x.org",
	}
	args := &model.ExperimentArgs{
		Callbacks:   model.NewPrinterCallbacks(log.Log),
		Measurement: measurement,
		Session:     newsession(),
	}
	err := measurer.Run(context.Background(), args)
	if host != "x.org" {
		t.Fatal("not the host we expected")
	}
	if err != nil {
		t.Fatal(err)
	}
}

func newsession() model.ExperimentSession {
	return &mockable.Session{MockableLogger: log.Log}
}

func TestSummaryKeysGeneric(t *testing.T) {
	measurement := &model.Measurement{TestKeys: &TestKeys{}}
	m := &Measurer{}
	osk, err := m.GetSummaryKeys(measurement)
	if err != nil {
		t.Fatal(err)
	}
	sk := osk.(SummaryKeys)
	if sk.IsAnomaly {
		t.Fatal("invalid isAnomaly")
	}
}
