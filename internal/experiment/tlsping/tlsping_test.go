package tlsping

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/legacy/mockable"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestConfig_alpn(t *testing.T) {
	c := Config{}
	if c.alpn() != "h2 http/1.1" {
		t.Fatal("invalid default alpn list")
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
	runHelper := func(ctx context.Context, input string) (*model.Measurement, model.ExperimentMeasurer, error) {
		m := NewExperimentMeasurer(Config{
			ALPN:        "http/1.1",
			Delay:       1, // millisecond
			Repetitions: expectedPings,
		})
		if m.ExperimentName() != "tlsping" {
			t.Fatal("invalid experiment name")
		}
		if m.ExperimentVersion() != "0.2.1" {
			t.Fatal("invalid experiment version")
		}
		meas := &model.Measurement{
			Input: model.MeasurementTarget(input),
		}
		sess := &mockable.Session{
			MockableLogger: model.DiscardLogger,
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
		_, _, err := runHelper(context.Background(), "")
		if !errors.Is(err, errNoInputProvided) {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("with invalid URL", func(t *testing.T) {
		_, _, err := runHelper(context.Background(), "\t")
		if !errors.Is(err, errInputIsNotAnURL) {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("with invalid scheme", func(t *testing.T) {
		_, _, err := runHelper(context.Background(), "https://8.8.8.8:443/")
		if !errors.Is(err, errInvalidScheme) {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("with missing port", func(t *testing.T) {
		_, _, err := runHelper(context.Background(), "tlshandshake://8.8.8.8")
		if !errors.Is(err, errMissingPort) {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("with local listener and successful outcome", func(t *testing.T) {
		srvr := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
		}))
		defer srvr.Close()
		URL, err := url.Parse(srvr.URL)
		if err != nil {
			t.Fatal(err)
		}
		URL.Scheme = "tlshandshake"
		meas, m, err := runHelper(context.Background(), URL.String())
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

	t.Run("with local listener and connect issues", func(t *testing.T) {
		srvr := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
		}))
		defer srvr.Close()
		URL, err := url.Parse(srvr.URL)
		if err != nil {
			t.Fatal(err)
		}
		URL.Scheme = "tlshandshake"
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // so we cannot dial any connection
		meas, m, err := runHelper(ctx, URL.String())
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

func TestConfig_sni(t *testing.T) {
	type fields struct {
		SNI string
	}
	type args struct {
		address string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
	}{{
		name: "with config.SNI being set",
		fields: fields{
			SNI: "x.org",
		},
		args: args{
			address: "google.com:443",
		},
		want: "x.org",
	}, {
		name:   "with invalid endpoint",
		fields: fields{},
		args: args{
			address: "google.com",
		},
		want: "",
	}, {
		name:   "with valid endpoint",
		fields: fields{},
		args: args{
			address: "google.com:443",
		},
		want: "google.com",
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Config{
				SNI: tt.fields.SNI,
			}
			if got := c.sni(tt.args.address); got != tt.want {
				t.Fatalf("Config.sni() = %v, want %v", got, tt.want)
			}
		})
	}
}
