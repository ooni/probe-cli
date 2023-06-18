package simplequicping

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/legacy/mockable"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/quic-go/quic-go"
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
		// we use the same empty DNS config for both client and servers
		dnsConfig := netem.NewDNSConfig()

		// configure [netemx.Environment]
		clientConf := &netemx.ClientConfig{DNSConfig: dnsConfig}
		serversConf := &netemx.ServersConfig{
			DNSConfig: dnsConfig,
			Servers: []netemx.ConfigServerStack{
				{
					ServerAddr: "8.8.8.8",
					HTTPServers: []netemx.ConfigHTTPServer{
						{
							Port: 443,
							QUIC: true,
						},
					},
				},
			},
		}
		// create a new test environment
		env := netemx.NewEnvironment(clientConf, serversConf)
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

// Start a server that echos all data on the first stream opened by the client.
//
// SPDX-License-Identifier: MIT
//
// See https://github.com/quic-go/quic-go/blob/v0.27.0/example/echo/echo.go#L34
func startEchoServer() (string, quic.Listener, error) {
	listener, err := quic.ListenAddr("127.0.0.1:0", generateTLSConfig(), nil)
	if err != nil {
		return "", nil, err
	}
	go echoWorkerMain(listener)
	URL := &url.URL{
		Scheme: "quichandshake",
		Host:   listener.Addr().String(),
		Path:   "/",
	}
	return URL.String(), listener, nil
}

// Worker used by startEchoServer to accept a quic connection.
//
// SPDX-License-Identifier: MIT
//
// See https://github.com/quic-go/quic-go/blob/v0.27.0/example/echo/echo.go#L34
func echoWorkerMain(listener quic.Listener) {
	for {
		conn, err := listener.Accept(context.Background())
		if err != nil {
			return
		}

// Setup a bare-bones TLS config for the server.
//
// SPDX-License-Identifier: MIT
//
// See https://github.com/quic-go/quic-go/blob/v0.27.0/example/echo/echo.go#L91
func generateTLSConfig() *tls.Config {
	key, err := rsa.GenerateKey(rand.Reader, 1024)
	if err != nil {
		panic(err)
	}
	template := x509.Certificate{SerialNumber: big.NewInt(1)}
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}
	keyPEM := pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)})
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		panic(err)
	}
	return &tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		NextProtos:   []string{"quic-echo-example"},
	}
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
