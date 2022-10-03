package portfiltering

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestMeasurerExperimentNameVersion(t *testing.T) {
	measurer := NewExperimentMeasurer(Config{})
	if measurer.ExperimentName() != "portfiltering" {
		t.Fatal("unexpected ExperimentName")
	}
	if measurer.ExperimentVersion() != "0.1.0" {
		t.Fatal("unexpected ExperimentVersion")
	}
}

func TestMeasurer_run(t *testing.T) {
	runHelper := func(ctx context.Context, input string, url string) (*model.Measurement, model.ExperimentMeasurer, error) {
		m := NewExperimentMeasurer(Config{
			TestHelper: url,
		})
		meas := &model.Measurement{
			Input: model.MeasurementTarget(input),
		}
		sess := &mocks.Session{
			MockLogger: func() model.Logger {
				return model.DiscardLogger
			},
		}
		callbacks := model.NewPrinterCallbacks(model.DiscardLogger)
		err := m.Run(ctx, sess, meas, callbacks)
		return meas, m, err
	}

	t.Run("with no input", func(t *testing.T) {
		ctx := context.Background()
		_, _, err := runHelper(ctx, "", "")
		if err == nil || err != errInputRequired {
			t.Fatal("unexpected error")
		}
	})

	t.Run("with invalid input", func(t *testing.T) {
		t.Run("with negative port number", func(t *testing.T) {
			ctx := context.Background()
			_, _, err := runHelper(ctx, "-1", "")
			if err == nil || err != errInvalidInput {
				t.Fatal(err)
			}
		})

		t.Run("with large invalid port number", func(t *testing.T) {
			ctx := context.Background()
			_, _, err := runHelper(ctx, "70000", "")
			if err == nil || err != errInvalidInput {
				t.Fatal(err)
			}
		})

		t.Run("with non-integer port number", func(t *testing.T) {
			ctx := context.Background()
			_, _, err := runHelper(ctx, "\t", "")
			if err == nil || err != errInvalidInput {
				t.Fatal(err)
			}
		})
	})

	// TODO(DecFox): Add a test that checks ports on the OONI API
	t.Run("with API testhelper", func(t *testing.T) {
	})

	t.Run("with local listener and successful outcome", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
		}))
		defer server.Close()
		URL, err := url.Parse(server.URL)
		if err != nil {
			t.Fatal(err)
		}
		ctx := context.Background()
		meas, m, err := runHelper(ctx, URL.Port(), URL.String())
		if err != nil {
			t.Fatal(err)
		}
		ask, err := m.GetSummaryKeys(meas)
		if err != nil {
			t.Fatal("cannot obtain summary")
		}
		summary := ask.(SummaryKeys)
		if summary.IsAnomaly {
			t.Fatal("expected no anomaly")
		}

		t.Run("testkeys", func(t *testing.T) {
			tk := meas.TestKeys.(*TestKeys)
			port, _ := strconv.Atoi(URL.Port())
			if tk.TCPConnect.IP != URL.Hostname() {
				t.Fatal("unexpected target IP")
			}
			if tk.TCPConnect.Port != port {
				t.Fatal("unexpected port")
			}
			if tk.TCPConnect.Status.Failure != nil {
				t.Fatal("unexpected error")
			}
		})
	})

	t.Run("with local listener and cancel", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
		}))
		defer server.Close()
		URL, err := url.Parse(server.URL)
		if err != nil {
			t.Fatal(err)
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		meas, m, err := runHelper(ctx, URL.Port(), URL.String())
		if err != nil {
			t.Fatal(err)
		}
		ask, err := m.GetSummaryKeys(meas)
		if err != nil {
			t.Fatal("cannot obtain summary")
		}
		summary := ask.(SummaryKeys)
		if summary.IsAnomaly {
			t.Fatal("expected no anomaly")
		}

		t.Run("testkeys", func(t *testing.T) {
			tk := meas.TestKeys.(*TestKeys)
			port, _ := strconv.Atoi(URL.Port())
			if tk.TCPConnect.IP != URL.Hostname() {
				t.Fatal("unexpected target IP")
			}
			if tk.TCPConnect.Port != port {
				t.Fatal("unexpected port")
			}
			if *tk.TCPConnect.Status.Failure != netxlite.FailureInterrupted {
				t.Fatal("unexpected error")
			}
		})
	})
}
