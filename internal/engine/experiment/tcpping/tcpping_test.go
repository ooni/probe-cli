package tcpping

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/mockable"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestWorksWithLocalListener(t *testing.T) {
	srvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	}))
	URL, err := url.Parse(srvr.URL)
	if err != nil {
		t.Fatal(err)
	}
	URL.Scheme = "tcpconnect"
	const expectedPings = 4
	m := NewExperimentMeasurer(Config{
		Delay:       1,
		Repetitions: expectedPings,
	})
	if m.ExperimentName() != "tcpping" {
		t.Fatal("invalid experiment name")
	}
	if m.ExperimentVersion() != "0.1.0" {
		t.Fatal("invalid experiment version")
	}
	ctx := context.Background()
	meas := &model.Measurement{
		Input: model.MeasurementTarget(URL.String()),
	}
	sess := &mockable.Session{}
	callbacks := model.NewPrinterCallbacks(model.DiscardLogger)
	err = m.Run(ctx, sess, meas, callbacks)
	if err != nil {
		t.Fatal(err)
	}
	tk := meas.TestKeys.(*TestKeys)
	if len(tk.Pings) != expectedPings {
		t.Fatal("unexpected number of pings")
	}
	ask, err := m.GetSummaryKeys(meas)
	if err != nil {
		t.Fatal("cannot obtain summary")
	}
	summary := ask.(SummaryKeys)
	if summary.IsAnomaly {
		t.Fatal("expected no anomaly")
	}
}

func TestConfig_repetitions(t *testing.T) {
	c := Config{}
	if c.repetitions() != 10 {
		t.Fatal("invalid default number of repetitions")
	}
}

func TestConfig_delay(t *testing.T) {
	c := Config{}
	if c.delay() != time.Second {
		t.Fatal("invalid default delay")
	}
}

func TestMeasurer_run(t *testing.T) {
	// expectedPings is the expected number of pings
	const expectedPings = 4

	// runHelper is an helper function to run this set of tests.
	runHelper := func(input string) (*model.Measurement, model.ExperimentMeasurer, error) {
		m := NewExperimentMeasurer(Config{
			Delay:       1,
			Repetitions: expectedPings,
		})
		if m.ExperimentName() != "tcpping" {
			t.Fatal("invalid experiment name")
		}
		if m.ExperimentVersion() != "0.1.0" {
			t.Fatal("invalid experiment version")
		}
		if m.ExperimentName() != "tcpping" {
			t.Fatal("invalid experiment name")
		}
		if m.ExperimentVersion() != "0.1.0" {
			t.Fatal("invalid experiment version")
		}
		ctx := context.Background()
		meas := &model.Measurement{
			Input: model.MeasurementTarget(input),
		}
		sess := &mockable.Session{}
		callbacks := model.NewPrinterCallbacks(model.DiscardLogger)
		err := m.Run(ctx, sess, meas, callbacks)
		return meas, m, err
	}

	t.Run("with empty input", func(t *testing.T) {
		_, _, err := runHelper("")
		if !errors.Is(err, errNoInputProvided) {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("with invalid URL", func(t *testing.T) {
		_, _, err := runHelper("\t")
		if !errors.Is(err, errInputIsNotAnURL) {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("with invalid scheme", func(t *testing.T) {
		_, _, err := runHelper("https://8.8.8.8:443/")
		if !errors.Is(err, errInvalidScheme) {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("with invalid port", func(t *testing.T) {
		_, _, err := runHelper("tcpconnect://8.8.8.8")
		if !errors.Is(err, errMissingPort) {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("with local listener", func(t *testing.T) {
		srvr := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
		}))
		URL, err := url.Parse(srvr.URL)
		if err != nil {
			t.Fatal(err)
		}
		URL.Scheme = "tcpconnect"
		meas, m, err := runHelper(URL.String())
		if err != nil {
			t.Fatal(err)
		}
		tk := meas.TestKeys.(*TestKeys)
		if len(tk.Pings) != expectedPings {
			t.Fatal("unexpected number of pings")
		}
		ask, err := m.GetSummaryKeys(meas)
		if err != nil {
			t.Fatal("cannot obtain summary")
		}
		summary := ask.(SummaryKeys)
		if summary.IsAnomaly {
			t.Fatal("expected no anomaly")
		}
	})
}
