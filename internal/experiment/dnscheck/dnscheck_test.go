package dnscheck

import (
	"context"
	"errors"
	"net/url"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestHTTPHostWithOverride(t *testing.T) {
	c := &Config{HTTPHost: "antani"}
	if result := c.httpHost("mascetti"); result != "antani" {
		t.Fatal("not the result we expected")
	}
}

func TestHTTPHostWithoutOverride(t *testing.T) {
	c := &Config{}
	if result := c.httpHost("mascetti"); result != "mascetti" {
		t.Fatal("not the result we expected")
	}
}

func TestTLSServerNameWithOverride(t *testing.T) {
	c := &Config{TLSServerName: "antani"}
	if result := c.tlsServerName("mascetti"); result != "antani" {
		t.Fatal("not the result we expected")
	}
}

func TestTLSServerNameWithoutOverride(t *testing.T) {
	c := &Config{}
	if result := c.tlsServerName("mascetti"); result != "mascetti" {
		t.Fatal("not the result we expected")
	}
}

func TestExperimentNameAndVersion(t *testing.T) {
	measurer := NewExperimentMeasurer()
	if measurer.ExperimentName() != "dnscheck" {
		t.Error("unexpected experiment name")
	}
	if measurer.ExperimentVersion() != "0.9.2" {
		t.Error("unexpected experiment version")
	}
}

func TestDNSCheckFailsWithInvalidInputType(t *testing.T) {
	measurer := NewExperimentMeasurer()
	args := &model.ExperimentArgs{
		Callbacks:   model.NewPrinterCallbacks(log.Log),
		Measurement: new(model.Measurement),
		Session:     newsession(),
		Target:      &model.OOAPIURLInfo{}, // not the expected input type
	}
	err := measurer.Run(context.Background(), args)
	if !errors.Is(err, ErrInvalidInputType) {
		t.Fatal("expected invalid-input-type error")
	}
}

func TestDNSCheckFailsWithoutInput(t *testing.T) {
	measurer := NewExperimentMeasurer()
	args := &model.ExperimentArgs{
		Callbacks:   model.NewPrinterCallbacks(log.Log),
		Measurement: new(model.Measurement),
		Session:     newsession(),
		Target: &Target{
			URL: "", // explicitly empty
			Options: &Config{
				Domain: "example.com",
			},
		},
	}
	err := measurer.Run(context.Background(), args)
	if !errors.Is(err, ErrInputRequired) {
		t.Fatal("expected no input error")
	}
}

func TestDNSCheckFailsWithInvalidURL(t *testing.T) {
	measurer := NewExperimentMeasurer()
	args := &model.ExperimentArgs{
		Callbacks:   model.NewPrinterCallbacks(log.Log),
		Measurement: &model.Measurement{Input: "Not a valid URL \x7f"},
		Session:     newsession(),
		Target: &Target{
			URL:     "Not a valid URL \x7f",
			Options: &Config{},
		},
	}
	err := measurer.Run(context.Background(), args)
	if !errors.Is(err, ErrInvalidURL) {
		t.Fatal("expected invalid input error")
	}
}

func TestDNSCheckFailsWithUnsupportedProtocol(t *testing.T) {
	measurer := NewExperimentMeasurer()
	args := &model.ExperimentArgs{
		Callbacks:   model.NewPrinterCallbacks(log.Log),
		Measurement: &model.Measurement{Input: "file://1.1.1.1"},
		Session:     newsession(),
		Target: &Target{
			URL:     "file://1.1.1.1",
			Options: &Config{},
		},
	}
	err := measurer.Run(context.Background(), args)
	if !errors.Is(err, ErrUnsupportedURLScheme) {
		t.Fatal("expected unsupported scheme error")
	}
}

func TestWithCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately cancel the context
	measurer := NewExperimentMeasurer()
	measurement := &model.Measurement{Input: "dot://one.one.one.one"}
	args := &model.ExperimentArgs{
		Callbacks:   model.NewPrinterCallbacks(log.Log),
		Measurement: measurement,
		Session:     newsession(),
		Target: &Target{
			URL: "dot://one.one.one.one",
			Options: &Config{
				DefaultAddrs: "1.1.1.1 1.0.0.1",
			},
		},
	}
	err := measurer.Run(ctx, args)
	if err != nil {
		t.Fatal(err)
	}
}

func TestDNSCheckFailsWithNilTarget(t *testing.T) {
	measurer := NewExperimentMeasurer()
	measurement := &model.Measurement{Input: "dot://one.one.one.one"}
	args := &model.ExperimentArgs{
		Callbacks:   model.NewPrinterCallbacks(log.Log),
		Measurement: measurement,
		Session:     newsession(),
		Target:      nil, // explicitly nil
	}
	err := measurer.Run(context.Background(), args)
	if !errors.Is(err, ErrInputRequired) {
		t.Fatal("unexpected err", err)
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
	if testing.Short() {
		t.Skip("skip test in short mode")
	}

	measurer := NewExperimentMeasurer()
	measurement := model.Measurement{Input: "dot://one.one.one.one:853"}
	args := &model.ExperimentArgs{
		Callbacks:   model.NewPrinterCallbacks(log.Log),
		Measurement: &measurement,
		Session:     newsession(),
		Target: &Target{
			URL: "dot://one.one.one.one:853",
			Options: &Config{
				DefaultAddrs: "1.1.1.1 1.0.0.1",
			},
		},
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
	return &mocks.Session{
		MockLogger: func() model.Logger {
			return log.Log
		},
	}
}

func TestDNSCheckWait(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}

	endpoints := &Endpoints{
		WaitTime: 1 * time.Second,
	}
	measurer := &Measurer{Endpoints: endpoints}
	run := func(input string) {
		measurement := model.Measurement{Input: model.MeasurementInput(input)}
		args := &model.ExperimentArgs{
			Callbacks:   model.NewPrinterCallbacks(log.Log),
			Measurement: &measurement,
			Session:     newsession(),
			Target: &Target{
				URL:     input,
				Options: &Config{},
			},
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
