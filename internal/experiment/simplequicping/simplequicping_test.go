package simplequicping

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
	if c.alpn() != "h3" {
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
	// run is an helper function to run this set of tests.
	run := func(input string) (*model.Measurement, model.ExperimentMeasurer, error) {
		m := NewExperimentMeasurer(Config{
			ALPN:        "h3",
			Delay:       1, // millisecond
			Repetitions: NPINGS,
			SNI:         SNI,
		})

		if m.ExperimentName() != "simplequicping" {
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
		args := &model.ExperimentArgs{
			Callbacks:   model.NewPrinterCallbacks(model.DiscardLogger),
			Measurement: meas,
			Session:     sess,
		}

		err := m.Run(context.Background(), args)

		return meas, m, err
	}

	t.Run("with empty input", func(t *testing.T) {
		_, _, err := run("")
		if !errors.Is(err, errNoInputProvided) {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("with invalid URL", func(t *testing.T) {
		_, _, err := run("\t")
		if !errors.Is(err, errInputIsNotAnURL) {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("with invalid scheme", func(t *testing.T) {
		_, _, err := run("https://8.8.8.8:443/")
		if !errors.Is(err, errInvalidScheme) {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("with missing port", func(t *testing.T) {
		_, _, err := run("quichandshake://8.8.8.8")
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
					Listeners: []netemx.Listener{
						{
							Port: 443,
							QUIC: true,
						},
					},
				},
			},
		}

		env := netemx.NewEnvironment(conf)
		defer env.Close()

		env.Do(func() {
			meas, _, err := run("quichandshake://8.8.8.8:443")
			if err != nil {
				t.Fatalf("Unexpected error: %s", err)
			}

			tk, _ := (meas.TestKeys).(*TestKeys)
			if len(tk.Pings) != NPINGS {
				t.Fatal("unexpected number of pings")
			}

			for _, p := range tk.Pings {
				if p.QUICHandshake.Failure != nil {
					t.Fatal("unexpected error", *p.QUICHandshake.Failure)
				}
				if len(p.NetworkEvents) < 1 {
					t.Fatal("unexpected number of network events")
				}
			}
		})
	})

	t.Run("with netem: with DPI that drops UDP datagrams to 8.8.8.8:443: expect failure", func(t *testing.T) {
		dnsConfig := netem.NewDNSConfig()
		conf := netemx.Config{
			DNSConfig: dnsConfig,
			Servers: []netemx.ServerStack{
				{
					ServerAddr: "8.8.8.8",
					Listeners: []netemx.Listener{
						{
							Port: 443,
							QUIC: true,
						},
					},
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
			ServerProtocol:  layers.IPProtocolUDP,
		})

		env.Do(func() {
			meas, _, err := run("quichandshake://8.8.8.8:443")
			if err != nil {
				t.Fatalf("Unexpected error: %s", err)
			}

			tk, _ := (meas.TestKeys).(*TestKeys)

			for _, p := range tk.Pings {
				if p.QUICHandshake.Failure == nil {
					t.Fatal("expected an error here")
				}
				if *p.QUICHandshake.Failure != netxlite.FailureGenericTimeoutError {
					t.Fatal("unexpected error type")
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
