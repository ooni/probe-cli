package dnsping

import (
	"context"
	"errors"
	"log"
	"net"
	"net/url"
	"testing"
	"time"

	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/engine/mockable"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func TestConfig_domains(t *testing.T) {
	c := Config{}
	if c.domains() != "edge-chat.instagram.com example.com" {
		t.Fatal("invalid default domains list")
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
			Domains:     "example.com",
			Delay:       1, // millisecond
			Repetitions: expectedPings,
		})
		if m.ExperimentName() != "dnsping" {
			t.Fatal("invalid experiment name")
		}
		if m.ExperimentVersion() != "0.1.0" {
			t.Fatal("invalid experiment version")
		}
		ctx := context.Background()
		meas := &model.Measurement{
			Input: model.MeasurementTarget(input),
		}
		sess := &mockable.Session{
			MockableLogger: model.DiscardLogger,
		}
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

	t.Run("with missing port", func(t *testing.T) {
		_, _, err := runHelper("udp://8.8.8.8")
		if !errors.Is(err, errMissingPort) {
			t.Fatal("unexpected error", err)
		}
	})

	t.Run("with local listener", func(t *testing.T) {
		srvrURL, dnsListener, err := startDNSServer()
		if err != nil {
			log.Fatal(err)
		}
		defer dnsListener.Close()
		meas, m, err := runHelper(srvrURL)
		if err != nil {
			t.Fatal(err)
		}
		tk := meas.TestKeys.(*TestKeys)
		if len(tk.Pings) != expectedPings*2 { // account for A & AAAA pings
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

// startDNSServer starts a local DNS server.
func startDNSServer() (string, net.PacketConn, error) {
	dnsListener, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		return "", nil, err
	}
	go runDNSServer(dnsListener)
	URL := &url.URL{
		Scheme: "udp",
		Host:   dnsListener.LocalAddr().String(),
		Path:   "/",
	}
	return URL.String(), dnsListener, nil
}

// runDNSServer runs the DNS server.
func runDNSServer(dnsListener net.PacketConn) {
	ds := &dns.Server{
		Handler:    &dnsHandler{},
		Net:        "udp",
		PacketConn: dnsListener,
	}
	err := ds.ActivateAndServe()
	if !errors.Is(err, net.ErrClosed) {
		runtimex.PanicOnError(err, "ActivateAndServe failed")
	}
}

// dnsHandler handles DNS requests.
type dnsHandler struct{}

// ServeDNS serves a DNS request
func (h *dnsHandler) ServeDNS(rw dns.ResponseWriter, req *dns.Msg) {
	m := new(dns.Msg)
	m.Compress = true
	m.MsgHdr.RecursionAvailable = true
	m.SetRcode(req, dns.RcodeServerFailure)
	rw.WriteMsg(m)
}
