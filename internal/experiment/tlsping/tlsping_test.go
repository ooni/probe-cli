package tlsping

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/gopacket/layers"
	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/legacy/mockable"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netemx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
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

func TestMeasurerRun(t *testing.T) {

	// run is an helper function to run this set of tests.
	run := func(ctx context.Context, input string) (*model.Measurement, model.ExperimentMeasurer, error) {
		m := NewExperimentMeasurer(Config{
			ALPN:        "http/1.1",
			Delay:       1, // millisecond
			Repetitions: 4,
			SNI:         "blocked.com",
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
		_, _, err := run(context.Background(), "")
		if !errors.Is(err, errNoInputProvided) {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("with invalid URL", func(t *testing.T) {
		_, _, err := run(context.Background(), "\t") // \t causes the URL to be invalid
		if !errors.Is(err, errInputIsNotAnURL) {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("with invalid scheme", func(t *testing.T) {
		_, _, err := run(context.Background(), "https://8.8.8.8:443/") // we expect tlshandshake://
		if !errors.Is(err, errInvalidScheme) {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("with missing port", func(t *testing.T) {
		_, _, err := run(context.Background(), "tlshandshake://8.8.8.8") // missing port
		if !errors.Is(err, errMissingPort) {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("with netem: without DPI: expect success", func(t *testing.T) {
		dnsConfig := netem.NewDNSConfig()
		conf := netemx.Config{
			DNSConfig: dnsConfig,
			Servers: []netemx.ServerStack{
				{
					ServerAddr: "8.8.8.8",
					Listeners:  []netemx.Listener{{Port: 443}},
				},
			},
		}

		env := netemx.NewEnvironment(conf)
		defer env.Close()

		env.Do(func() {
			meas, m, err := run(context.Background(), "tlshandshake://8.8.8.8:443")
			if err != nil {
				t.Fatalf("Unexpected error: %s", err)
			}

			tk, _ := (meas.TestKeys).(*TestKeys)
			if len(tk.Pings) != 4 {
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

			for _, p := range tk.Pings {
				if p.TCPConnect == nil {
					t.Fatal("TCPConnect should not be nil")
				}
				if p.TLSHandshake == nil {
					t.Fatal("TLSHandshake should not be nil")
				}
				if p.TLSHandshake.Failure != nil {
					t.Fatal("unexpected error")
				}
				if len(p.NetworkEvents) < 1 {
					t.Fatal("unexpected number of network events")
				}
			}
		})
	})

	t.Run("with netem: with DPI that drops TCP segments to 8.8.8.8:443: expect failure", func(t *testing.T) {
		dnsConfig := netem.NewDNSConfig()
		conf := netemx.Config{
			DNSConfig: dnsConfig,
			Servers: []netemx.ServerStack{
				{
					ServerAddr: "8.8.8.8",
					Listeners:  []netemx.Listener{{Port: 443}},
				},
			},
		}

		env := netemx.NewEnvironment(conf)
		defer env.Close()

		dpi := env.DPIEngine()
		dpi.AddRule(&netem.DPIDropTrafficForServerEndpoint{
			Logger:          model.DiscardLogger,
			ServerIPAddress: "8.8.8.8",
			ServerPort:      443,
			ServerProtocol:  layers.IPProtocolTCP,
		})

		env.Do(func() {
			meas, m, err := run(context.Background(), "tlshandshake://8.8.8.8:443")
			if err != nil {
				t.Fatalf("Unexpected error: %s", err)
			}

			tk, _ := (meas.TestKeys).(*TestKeys)

			// note: this experiment does not set anomaly but we still want
			// to have a test here for when we possibly will
			ask, err := m.GetSummaryKeys(meas)
			if err != nil {
				t.Fatal("cannot obtain summary")
			}
			summary := ask.(SummaryKeys)
			if summary.IsAnomaly {
				t.Fatal("expected no anomaly")
			}

			for _, p := range tk.Pings {
				if p.TCPConnect == nil {
					t.Fatal("TCPConnect should not be nil")
				}
				if p.TCPConnect.Status.Failure == nil {
					t.Fatal("expected an error here")
				}
				if *p.TCPConnect.Status.Failure != netxlite.FailureGenericTimeoutError {
					t.Fatal("expected an error here")
				}

				if p.TLSHandshake != nil {
					t.Fatal("expected TLSHandshake to be nil")
				}

				// TODO(bassosimone): if we were using dslx here we would have an
				// event about connect, so we should eventually address this issue
				// and have one.
				if len(p.NetworkEvents) > 0 {
					t.Fatal("unexpected number of network events")
				}
			}
		})
	})

	t.Run("with netem: with DPI that resets TLS to SNI blocked.com: expect failure", func(t *testing.T) {
		dnsConfig := netem.NewDNSConfig()
		conf := netemx.Config{
			DNSConfig: dnsConfig,
			Servers: []netemx.ServerStack{
				{
					ServerAddr: "8.8.8.8",
					Listeners:  []netemx.Listener{{Port: 443}},
				},
			},
		}

		env := netemx.NewEnvironment(conf)
		defer env.Close()

		dpi := env.DPIEngine()
		dpi.AddRule(&netem.DPIResetTrafficForTLSSNI{
			Logger: model.DiscardLogger,
			SNI:    "blocked.com", // this is the SNI we set inside run()
		})

		env.Do(func() {
			meas, m, err := run(context.Background(), "tlshandshake://8.8.8.8:443")
			if err != nil {
				t.Fatalf("Unexpected error: %s", err)
			}
			tk, _ := (meas.TestKeys).(*TestKeys)

			// note: this experiment does not set anomaly but we still want
			// to have a test here for when we possibly will
			ask, err := m.GetSummaryKeys(meas)
			if err != nil {
				t.Fatal("cannot obtain summary")
			}
			summary := ask.(SummaryKeys)
			if summary.IsAnomaly {
				t.Fatal("expected no anomaly")
			}

			for _, p := range tk.Pings {
				if p.TCPConnect == nil {
					t.Fatal("TCPConnect should not be nil")
				}
				if p.TCPConnect.Status.Failure != nil {
					t.Fatal("did not expect an error here")
				}

				if p.TLSHandshake == nil {
					t.Fatal("unexpected nil TLSHandshake")
				}
				if p.TLSHandshake.Failure == nil {
					t.Fatal("expected an TLS Handshake failure here")
				}
				if *p.TLSHandshake.Failure != netxlite.FailureConnectionReset {
					t.Fatal("unexpected TLS failure type")
				}

				if len(p.NetworkEvents) <= 0 {
					t.Fatal("unexpected number of network events")
				}
			}
		})
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
