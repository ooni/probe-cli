package tlsmiddlebox

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/netxlite/filtering"
)

func TestMeasurerExperimentNameVersion(t *testing.T) {
	measurer := NewExperimentMeasurer(Config{})
	if measurer.ExperimentName() != "tlsmiddlebox" {
		t.Fatal("unexpected ExperimentName")
	}
	if measurer.ExperimentVersion() != "0.1.0" {
		t.Fatal("unexpected ExperimentVersion")
	}
}

func TestMeasurer_input_failure(t *testing.T) {
	runHelper := func(ctx context.Context, input string, th string, sniControl string) (*model.Measurement, model.ExperimentMeasurer, error) {
		m := NewExperimentMeasurer(Config{
			TestHelper: th,
			SNIControl: sniControl,
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
		args := &model.ExperimentArgs{
			Callbacks:   callbacks,
			Measurement: meas,
			Session:     sess,
		}
		err := m.Run(ctx, args)
		return meas, m, err
	}

	t.Run("with empty input", func(t *testing.T) {
		_, _, err := runHelper(context.Background(), "", "", "")
		if !errors.Is(err, errNoInputProvided) {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("with invalid URL", func(t *testing.T) {
		_, _, err := runHelper(context.Background(), "\t", "", "")
		if !errors.Is(err, errInputIsNotAnURL) {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("with invalid scheme", func(t *testing.T) {
		_, _, err := runHelper(context.Background(), "http://8.8.8.8:443/", "", "")
		if !errors.Is(err, errInvalidInputScheme) {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("with invalid testhelper", func(t *testing.T) {
		_, _, err := runHelper(context.Background(), "tlstrace://example.com", "\t", "")
		if !errors.Is(err, errInvalidTestHelper) {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("with invalid TH scheme", func(t *testing.T) {
		_, _, err := runHelper(context.Background(), "tlstrace://example.com", "http://google.com", "")
		if !errors.Is(err, errInvalidTHScheme) {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("with local listener and successful outcome", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skip test in short mode")
		}
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
		}))
		defer server.Close()
		URL, err := url.Parse(server.URL)
		if err != nil {
			t.Fatal(err)
		}
		URL.Scheme = "tlshandshake"
		meas, m, err := runHelper(context.Background(), "tlstrace://google.com", URL.String(), "")
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
			tr := tk.IterativeTrace
			if len(tr) != 1 {
				t.Fatal("unexpected number of trace")
			}
			trace := tr[0]
			if trace.Address != URL.Host {
				t.Fatal("unexpected trace address")
			}

			t.Run("control trace", func(t *testing.T) {
				if trace.ControlTrace == nil || trace.ControlTrace.SNI != "example.com" {
					t.Fatal("unexpected control trace for url")
				}
				if len(trace.ControlTrace.Iterations) != 1 {
					t.Fatal("unexpected number of iterations")
				}
			})

			t.Run("target trace", func(t *testing.T) {
				if trace.TargetTrace == nil || trace.TargetTrace.SNI != "google.com" {
					t.Fatal("unexpected target trace for url")
				}
				if len(trace.TargetTrace.Iterations) != 1 {
					t.Fatal("unexpected number of iterations")
				}
			})
		})
	})

	t.Run("with local listener and timeout", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skip test in short mode")
		}
		server := filtering.NewTLSServer(filtering.TLSActionTimeout)
		defer server.Close()
		th := "tlshandshake://" + server.Endpoint()
		URL, err := url.Parse(th)
		if err != nil {
			t.Fatal(err)
		}
		meas, m, err := runHelper(context.Background(), "tlstrace://google.com", URL.String(), "")
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
			tr := tk.IterativeTrace
			if len(tr) != 1 {
				t.Fatal("unexpected number of trace")
			}
			trace := tr[0]
			if trace.Address != URL.Host {
				t.Fatal("unexpected trace address")
			}

			t.Run("control trace", func(t *testing.T) {
				if trace.ControlTrace == nil || trace.ControlTrace.SNI != "example.com" {
					t.Fatal("unexpected control trace for url")
				}
				if len(trace.ControlTrace.Iterations) != 20 {
					t.Fatal("unexpected number of iterations")
				}
			})

			t.Run("target trace", func(t *testing.T) {
				if trace.TargetTrace == nil || trace.TargetTrace.SNI != "google.com" {
					t.Fatal("unexpected target trace for url")
				}
				if len(trace.TargetTrace.Iterations) != 20 {
					t.Fatal("unexpected number of iterations")
				}
			})
		})
	})

	t.Run("with local listener and connect issues", func(t *testing.T) {
		if testing.Short() {
			t.Skip("skip test in short mode")
		}
		server := filtering.NewTLSServer(filtering.TLSActionReset)
		defer server.Close()
		th := "tlshandshake://" + server.Endpoint()
		URL, err := url.Parse(th)
		if err != nil {
			t.Fatal(err)
		}
		meas, m, err := runHelper(context.Background(), "tlstrace://google.com", URL.String(), "")
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
			tr := tk.IterativeTrace
			if len(tr) != 1 {
				t.Fatal("unexpected number of trace")
			}
			trace := tr[0]
			if trace.Address != URL.Host {
				t.Fatal("unexpected trace address")
			}

			t.Run("control trace", func(t *testing.T) {
				if trace.ControlTrace == nil || trace.ControlTrace.SNI != "example.com" {
					t.Fatal("unexpected control trace for url")
				}
				if len(trace.ControlTrace.Iterations) != 1 {
					t.Fatal("unexpected number of iterations")
				}
			})

			t.Run("target trace", func(t *testing.T) {
				if trace.TargetTrace == nil || trace.TargetTrace.SNI != "google.com" {
					t.Fatal("unexpected target trace for url")
				}
				if len(trace.TargetTrace.Iterations) != 1 {
					t.Fatal("unexpected number of iterations")
				}
			})
		})
	})
}
