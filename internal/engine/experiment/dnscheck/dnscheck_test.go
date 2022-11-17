package dnscheck

import (
	"context"
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine/mockable"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestHTTPHostWithOverride(t *testing.T) {
	m := Measurer{Config: Config{HTTPHost: "antani"}}
	result := m.httpHost("mascetti")
	if result != "antani" {
		t.Fatal("not the result we expected")
	}
}

func TestHTTPHostWithoutOverride(t *testing.T) {
	m := Measurer{Config: Config{}}
	result := m.httpHost("mascetti")
	if result != "mascetti" {
		t.Fatal("not the result we expected")
	}
}

func TestTLSServerNameWithOverride(t *testing.T) {
	m := Measurer{Config: Config{TLSServerName: "antani"}}
	result := m.tlsServerName("mascetti")
	if result != "antani" {
		t.Fatal("not the result we expected")
	}
}

func TestTLSServerNameWithoutOverride(t *testing.T) {
	m := Measurer{Config: Config{}}
	result := m.tlsServerName("mascetti")
	if result != "mascetti" {
		t.Fatal("not the result we expected")
	}
}

func TestExperimentNameAndVersion(t *testing.T) {
	measurer := NewExperimentMeasurer(Config{Domain: "example.com"})
	if measurer.ExperimentName() != "dnscheck" {
		t.Error("unexpected experiment name")
	}
	if measurer.ExperimentVersion() != "0.9.2" {
		t.Error("unexpected experiment version")
	}
}

func TestDNSCheckFailsWithoutInput(t *testing.T) {
	measurer := NewExperimentMeasurer(Config{Domain: "example.com"})
	args := &model.ExperimentArgs{
		Callbacks:   model.NewPrinterCallbacks(log.Log),
		Measurement: new(model.Measurement),
		Session:     newsession(),
	}
	err := measurer.Run(context.Background(), args)
	if !errors.Is(err, ErrInputRequired) {
		t.Fatal("expected no input error")
	}
}

func TestDNSCheckFailsWithInvalidURL(t *testing.T) {
	measurer := NewExperimentMeasurer(Config{})
	args := &model.ExperimentArgs{
		Callbacks:   model.NewPrinterCallbacks(log.Log),
		Measurement: &model.Measurement{Input: "Not a valid URL \x7f"},
		Session:     newsession(),
	}
	err := measurer.Run(context.Background(), args)
	if !errors.Is(err, ErrInvalidURL) {
		t.Fatal("expected invalid input error")
	}
}

func TestDNSCheckFailsWithUnsupportedProtocol(t *testing.T) {
	measurer := NewExperimentMeasurer(Config{})
	args := &model.ExperimentArgs{
		Callbacks:   model.NewPrinterCallbacks(log.Log),
		Measurement: &model.Measurement{Input: "file://1.1.1.1"},
		Session:     newsession(),
	}
	err := measurer.Run(context.Background(), args)
	if !errors.Is(err, ErrUnsupportedURLScheme) {
		t.Fatal("expected unsupported scheme error")
	}
}

func TestWithCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately cancel the context
	measurer := NewExperimentMeasurer(Config{
		DefaultAddrs: "1.1.1.1 1.0.0.1",
	})
	measurement := &model.Measurement{Input: "dot://one.one.one.one"}
	args := &model.ExperimentArgs{
		Callbacks:   model.NewPrinterCallbacks(log.Log),
		Measurement: measurement,
		Session:     newsession(),
	}
	err := measurer.Run(ctx, args)
	if err != nil {
		t.Fatal(err)
	}
	sk, err := measurer.GetSummaryKeys(measurement)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := sk.(SummaryKeys); !ok {
		t.Fatal("invalid type for summary keys")
	}
}

func TestMakeResolverURL(t *testing.T) {
	// test address substitution
	addr := "255.255.255.0"
	resolver := makeResolverURL(&url.URL{Host: "example.com"}, addr)
	resolverURL, err := url.Parse(resolver)
	if err != nil {
		t.Fatal(err)
	}
	if resolverURL.Host != addr {
		t.Fatal("expected address to be set as host")
	}

	// test IPv6 URLs are quoted
	addr = "2001:db8:85a3:8d3:1319:8a2e:370"
	resolver = makeResolverURL(&url.URL{Host: "example.com"}, addr)
	resolverURL, err = url.Parse(resolver)
	if err != nil {
		t.Fatal(err)
	}
	if resolverURL.Host != "["+addr+"]" {
		t.Fatal("expected URL host to be quoted")
	}
}

func TestDNSCheckValid(t *testing.T) {
	measurer := NewExperimentMeasurer(Config{
		DefaultAddrs: "1.1.1.1 1.0.0.1",
	})
	measurement := model.Measurement{Input: "dot://one.one.one.one:853"}
	args := &model.ExperimentArgs{
		Callbacks:   model.NewPrinterCallbacks(log.Log),
		Measurement: &measurement,
		Session:     newsession(),
	}
	err := measurer.Run(context.Background(), args)
	if err != nil {
		t.Fatalf("unexpected error: %s", err.Error())
	}
	tk := measurement.TestKeys.(*TestKeys)
	if tk.Domain != defaultDomain {
		t.Fatal("unexpected default value for domain")
	}
	if tk.Bootstrap == nil {
		t.Fatal("unexpected value for bootstrap")
	}
	if tk.BootstrapFailure != nil {
		t.Fatal("unexpected value for bootstrap_failure")
	}
	if len(tk.Lookups) <= 0 {
		t.Fatal("unexpected value for lookups")
	}
}

func newsession() model.ExperimentSession {
	return &mockable.Session{MockableLogger: log.Log}
}

func TestSummaryKeysGeneric(t *testing.T) {
	measurement := &model.Measurement{TestKeys: &TestKeys{}}
	m := &Measurer{}
	osk, err := m.GetSummaryKeys(measurement)
	if err != nil {
		t.Fatal(err)
	}
	sk := osk.(SummaryKeys)
	if sk.IsAnomaly {
		t.Fatal("invalid isAnomaly")
	}
}

func TestDNSCheckWait(t *testing.T) {
	endpoints := &Endpoints{
		WaitTime: 1 * time.Second,
	}
	measurer := &Measurer{Endpoints: endpoints}
	run := func(input string) {
		measurement := model.Measurement{Input: model.MeasurementTarget(input)}
		args := &model.ExperimentArgs{
			Callbacks:   model.NewPrinterCallbacks(log.Log),
			Measurement: &measurement,
			Session:     newsession(),
		}
		err := measurer.Run(context.Background(), args)
		if err != nil {
			t.Fatalf("unexpected error: %s", err.Error())
		}
		tk := measurement.TestKeys.(*TestKeys)
		if tk.Domain != defaultDomain {
			t.Fatal("unexpected default value for domain")
		}
		if tk.Bootstrap == nil {
			t.Fatalf("unexpected value for bootstrap: %+v", tk.Bootstrap)
		}
		if tk.BootstrapFailure != nil {
			t.Fatal("unexpected value for bootstrap_failure")
		}
		if len(tk.Lookups) <= 0 {
			t.Fatal("unexpected value for lookups")
		}
	}
	run("dot://one.one.one.one")
	run("dot://1dot1dot1dot1.cloudflare-dns.com")
	if endpoints.count.Load() < 1 {
		t.Fatal("did not sleep")
	}
}
