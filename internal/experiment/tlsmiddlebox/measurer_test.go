package tlsmiddlebox

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"slices"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

func TestChooseIPv4(t *testing.T) {
	t.Run("returns IPv4", func(t *testing.T) {
		var addrs = []string{"1.2.3.4", "5.6.7.8", "2001:db8:7a3f:91c2:4e8b:2d10:6f5a:8c31", "2001:db8:4c71:a9e2:8b36:f105:7d2a:9e64"}
		result := chooseIPv4(addrs)

		if len(result) != 2 {
			t.Fatalf("expected 2 IPv4 addresses, got %d: %v", len(result), result)
		}

		if !slices.Contains(result, "1.2.3.4") {
			t.Errorf("missing IPv4 address 1.2.3.4: %v", result)
		}

		if !slices.Contains(result, "5.6.7.8") {
			t.Errorf("missing IPv4 address 5.6.7.8: %v", result)
		}
	})

	t.Run("returns IPv6", func(t *testing.T) {
		var addrs = []string{"2001:db8:7a3f:91c2:4e8b:2d10:6f5a:8c31", "2001:db8:4c71:a9e2:8b36:f105:7d2a:9e64"}
		result := chooseIPv4(addrs)

		if len(result) != 2 {
			t.Fatalf("expected 2 IPv6 addresses, got %d: %v", len(result), result)
		}

		if !slices.Contains(result, "2001:db8:7a3f:91c2:4e8b:2d10:6f5a:8c31") {
			t.Errorf("missing IPv6 address 2001:db8:7a3f:91c2:4e8b:2d10:6f5a:8c31: %v", result)
		}

		if !slices.Contains(result, "2001:db8:4c71:a9e2:8b36:f105:7d2a:9e64") {
			t.Errorf("missing IPv6 address 2001:db8:4c71:a9e2:8b36:f105:7d2a:9e64: %v", result)
		}
	})
}

func TestMeasurerExperimentNameVersion(t *testing.T) {
	measurer := NewExperimentMeasurer()
	if measurer.ExperimentName() != "tlsmiddlebox" {
		t.Fatal("unexpected ExperimentName")
	}
	if measurer.ExperimentVersion() != "0.1.5" {
		t.Fatal("unexpected ExperimentVersion")
	}
}

func TestMeasurer_input_failure(t *testing.T) {
	runHelper := func(ctx context.Context, input string, th string, sniControl string) (*model.Measurement, model.ExperimentMeasurer, error) {
		m := NewExperimentMeasurer()
		meas := &model.Measurement{
			Input: model.MeasurementInput(input),
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
			Target: &Target{
				Config: &Config{
					TestHelper: th,
					SNIControl: sniControl,
				},
				URL: input,
			},
		}
		err := m.Run(ctx, args)
		return meas, m, err
	}

	t.Run("with empty input", func(t *testing.T) {
		_, _, err := runHelper(context.Background(), "", "", "")
		if !errors.Is(err, ErrNoInput) {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("with invalid URL", func(t *testing.T) {
		_, _, err := runHelper(context.Background(), "\t", "", "")
		if !errors.Is(err, ErrInputIsNotAnURL) {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("with invalid scheme", func(t *testing.T) {
		_, _, err := runHelper(context.Background(), "http://8.8.8.8:443/", "", "")
		if !errors.Is(err, ErrUnsupportedInput) {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("with invalid testhelper", func(t *testing.T) {
		_, _, err := runHelper(context.Background(), "https://example.com", "\t", "")
		if !errors.Is(err, errInvalidTestHelper) {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("with invalid TH scheme", func(t *testing.T) {
		_, _, err := runHelper(context.Background(), "https://example.com", "http://google.com", "")
		if !errors.Is(err, errInvalidTHScheme) {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("with invalid ClientId", func(t *testing.T) {
		m := NewExperimentMeasurer()
		meas := &model.Measurement{
			Input: model.MeasurementInput("https://example.com"),
		}
		sess := &mocks.Session{
			MockLogger: func() model.Logger {
				return model.DiscardLogger
			},
		}
		args := &model.ExperimentArgs{
			Callbacks:   model.NewPrinterCallbacks(model.DiscardLogger),
			Measurement: meas,
			Session:     sess,
			Target: &Target{
				Config: &Config{
					ClientId: 5, // we only know fingerprints between 1 and 4
				},
				URL: "https://example.com",
			},
		}
		err := m.Run(context.Background(), args)
		if !errors.Is(err, errInvalidClientId) {
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
		meas, _, err := runHelper(context.Background(), "https://google.com", URL.String(), "")
		if err != nil {
			t.Fatal(err)
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
		server := testingx.MustNewTLSServer(testingx.TLSHandlerTimeout())
		defer server.Close()
		th := "tlshandshake://" + server.Endpoint()
		URL, err := url.Parse(th)
		if err != nil {
			t.Fatal(err)
		}
		meas, _, err := runHelper(context.Background(), "https://google.com", URL.String(), "")
		if err != nil {
			t.Fatal(err)
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
		server := testingx.MustNewTLSServer(testingx.TLSHandlerReset())
		defer server.Close()
		th := "tlshandshake://" + server.Endpoint()
		URL, err := url.Parse(th)
		if err != nil {
			t.Fatal(err)
		}
		meas, _, err := runHelper(context.Background(), "https://google.com", URL.String(), "")
		if err != nil {
			t.Fatal(err)
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
