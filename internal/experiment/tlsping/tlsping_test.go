package tlsping

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/gopacket/layers"
	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/mocks"
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

const (
	NPINGS = 4
	SNI    = "blocked.com"
)

func TestMeasurerRun(t *testing.T) {
	// runHelper is an helper function to run this set of tests.
	runHelper := func(ctx context.Context, input string) (*model.Measurement, model.ExperimentMeasurer, error) {
		m := NewExperimentMeasurer(Config{
			ALPN:        "http/1.1",
			Delay:       1, // millisecond
			Repetitions: NPINGS,
			SNI:         SNI,
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
		sess := &mocks.Session{
			MockLogger: func() model.Logger { return model.DiscardLogger },
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
		_, _, err := runHelper(context.Background(), "\t") // \t causes the URL to be invalid
		if !errors.Is(err, errInputIsNotAnURL) {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("with invalid scheme", func(t *testing.T) {
		_, _, err := runHelper(context.Background(), "https://8.8.8.8:443/") // we expect tlshandshake://
		if !errors.Is(err, errInvalidScheme) {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("with missing port", func(t *testing.T) {
		_, _, err := runHelper(context.Background(), "tlshandshake://8.8.8.8") // missing port
		if !errors.Is(err, errMissingPort) {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("with netem: without DPI: expect success", func(t *testing.T) {
		// create a new test environment
		env := netemx.MustNewQAEnv(netemx.QAEnvOptionNetStack(
			"8.8.8.8",
			&netemx.HTTPSecureServerFactory{
				Factory:          netemx.ExampleWebPageHandlerFactory(),
				Ports:            []int{443},
				ServerNameMain:   SNI,
				ServerNameExtras: []string{},
			},
		))
		defer env.Close()

		env.Do(func() {
			meas, _, err := runHelper(context.Background(), "tlshandshake://8.8.8.8:443")
			if err != nil {
				t.Fatalf("Unexpected error: %s", err)
			}

			tk, _ := (meas.TestKeys).(*TestKeys)
			if len(tk.Pings) != NPINGS {
				t.Fatal("unexpected number of pings")
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
		// create a new test environment
		env := netemx.MustNewQAEnv(netemx.QAEnvOptionNetStack(
			"8.8.8.8",
			&netemx.HTTPSecureServerFactory{
				Factory:          netemx.ExampleWebPageHandlerFactory(),
				Ports:            []int{443},
				ServerNameMain:   SNI,
				ServerNameExtras: []string{},
			},
		))
		defer env.Close()

		// add DPI engine to emulate the censorship condition
		dpi := env.DPIEngine()
		dpi.AddRule(&netem.DPIDropTrafficForServerEndpoint{
			Logger:          model.DiscardLogger,
			ServerIPAddress: "8.8.8.8",
			ServerPort:      443,
			ServerProtocol:  layers.IPProtocolTCP,
		})

		env.Do(func() {
			meas, _, err := runHelper(context.Background(), "tlshandshake://8.8.8.8:443")
			if err != nil {
				t.Fatalf("Unexpected error: %s", err)
			}

			tk, _ := (meas.TestKeys).(*TestKeys)

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
		// create a new test environment
		env := netemx.MustNewQAEnv(netemx.QAEnvOptionNetStack(
			"8.8.8.8",
			&netemx.HTTPSecureServerFactory{
				Factory:          netemx.ExampleWebPageHandlerFactory(),
				Ports:            []int{443},
				ServerNameMain:   SNI,
				ServerNameExtras: []string{},
			},
		))
		defer env.Close()

		// add DPI engine to emulate the censorship condition
		dpi := env.DPIEngine()
		dpi.AddRule(&netem.DPIResetTrafficForTLSSNI{
			Logger: model.DiscardLogger,
			SNI:    SNI, // this is the SNI we set inside runHelper()
		})

		env.Do(func() {
			meas, _, err := runHelper(context.Background(), "tlshandshake://8.8.8.8:443")
			if err != nil {
				t.Fatalf("Unexpected error: %s", err)
			}
			tk, _ := (meas.TestKeys).(*TestKeys)

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
					t.Fatal("expected a TLS Handshake failure here")
				}
				if *p.TLSHandshake.Failure != netxlite.FailureConnectionReset {
					t.Fatal("unexpected TLS failure type")
				}

				// TODO(bassosimone): if we were using dslx here we would have an
				// event about connect, so we should eventually address this issue
				// and have one.
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
