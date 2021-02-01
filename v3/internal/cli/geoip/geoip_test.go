package geoip

import (
	"errors"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/ooni"
	"github.com/ooni/probe-cli/v3/internal/oonitest"
)

func TestNewProbeCLIFailed(t *testing.T) {
	fo := &oonitest.FakeOutput{}
	expected := errors.New("mocked error")
	err := dogeoip(dogeoipconfig{
		SectionTitle: fo.SectionTitle,
		NewProbeCLI: func() (ooni.ProbeCLI, error) {
			return nil, expected
		},
	})
	if !errors.Is(err, expected) {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if len(fo.FakeSectionTitle) != 1 {
		t.Fatal("invalid section title list size")
	}
	if fo.FakeSectionTitle[0] != "GeoIP lookup" {
		t.Fatal("unexpected string")
	}
}

func TestNewProbeEngineFailed(t *testing.T) {
	fo := &oonitest.FakeOutput{}
	expected := errors.New("mocked error")
	cli := &oonitest.FakeProbeCLI{
		FakeProbeEngineErr: expected,
	}
	err := dogeoip(dogeoipconfig{
		SectionTitle: fo.SectionTitle,
		NewProbeCLI: func() (ooni.ProbeCLI, error) {
			return cli, nil
		},
	})
	if !errors.Is(err, expected) {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if len(fo.FakeSectionTitle) != 1 {
		t.Fatal("invalid section title list size")
	}
	if fo.FakeSectionTitle[0] != "GeoIP lookup" {
		t.Fatal("unexpected string")
	}
}

func TestMaybeLookupLocationFailed(t *testing.T) {
	fo := &oonitest.FakeOutput{}
	expected := errors.New("mocked error")
	engine := &oonitest.FakeProbeEngine{
		FakeMaybeLookupLocation: expected,
	}
	cli := &oonitest.FakeProbeCLI{
		FakeProbeEnginePtr: engine,
	}
	err := dogeoip(dogeoipconfig{
		SectionTitle: fo.SectionTitle,
		NewProbeCLI: func() (ooni.ProbeCLI, error) {
			return cli, nil
		},
	})
	if !errors.Is(err, expected) {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if len(fo.FakeSectionTitle) != 1 {
		t.Fatal("invalid section title list size")
	}
	if fo.FakeSectionTitle[0] != "GeoIP lookup" {
		t.Fatal("unexpected string")
	}
}

func TestMaybeLookupLocationSuccess(t *testing.T) {
	fo := &oonitest.FakeOutput{}
	engine := &oonitest.FakeProbeEngine{
		FakeProbeASNString:   "AS30722",
		FakeProbeCC:          "IT",
		FakeProbeNetworkName: "Vodafone Italia S.p.A.",
		FakeProbeIP:          "130.25.90.216",
	}
	cli := &oonitest.FakeProbeCLI{
		FakeProbeEnginePtr: engine,
	}
	handler := &oonitest.FakeLoggerHandler{}
	err := dogeoip(dogeoipconfig{
		SectionTitle: fo.SectionTitle,
		NewProbeCLI: func() (ooni.ProbeCLI, error) {
			return cli, nil
		},
		Logger: &log.Logger{
			Handler: handler,
			Level:   log.DebugLevel,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(fo.FakeSectionTitle) != 1 {
		t.Fatal("invalid section title list size")
	}
	if fo.FakeSectionTitle[0] != "GeoIP lookup" {
		t.Fatal("unexpected string")
	}
	if len(handler.FakeEntries) != 1 {
		t.Fatal("invalid number of written entries")
	}
	entry := handler.FakeEntries[0]
	if entry.Level != log.InfoLevel {
		t.Fatal("invalid log level")
	}
	if entry.Message != "Looked up your location" {
		t.Fatal("invalid .Message")
	}
	if entry.Fields["asn"].(string) != "AS30722" {
		t.Fatal("invalid asn")
	}
	if entry.Fields["country_code"].(string) != "IT" {
		t.Fatal("invalid asn")
	}
	if entry.Fields["network_name"].(string) != "Vodafone Italia S.p.A." {
		t.Fatal("invalid asn")
	}
	if entry.Fields["ip"].(string) != "130.25.90.216" {
		t.Fatal("invalid asn")
	}
}
