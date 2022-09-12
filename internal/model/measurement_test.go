package model

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
)

func TestMeasurementTargetMarshalJSON(t *testing.T) {
	var mt MeasurementTarget
	data, err := json.Marshal(mt)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "null" {
		t.Fatal("unexpected serialization")
	}
	mt = "xx"
	data, err = json.Marshal(mt)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != `"xx"` {
		t.Fatal("unexpected serialization")
	}
}

type fakeTestKeys struct {
	ClientResolver string `json:"client_resolver"`
	Body           string `json:"body"`
}

func TestAddAnnotations(t *testing.T) {
	m := &Measurement{}
	m.AddAnnotations(map[string]string{
		"foo": "bar",
		"f":   "b",
	})
	m.AddAnnotations(map[string]string{
		"foobar": "bar",
		"f":      "b",
	})
	if len(m.Annotations) != 3 {
		t.Fatal("unexpected number of annotations")
	}
	if m.Annotations["foo"] != "bar" {
		t.Fatal("unexpected annotation")
	}
	if m.Annotations["f"] != "b" {
		t.Fatal("unexpected annotation")
	}
	if m.Annotations["foobar"] != "bar" {
		t.Fatal("unexpected annotation")
	}
}

type makeMeasurementConfig struct {
	ProbeIP             string
	ProbeASN            string
	ProbeNetworkName    string
	ProbeCC             string
	ResolverIP          string
	ResolverNetworkName string
	ResolverASN         string
}

func makeMeasurement(config makeMeasurementConfig) *Measurement {
	return &Measurement{
		DataFormatVersion:    "0.3.0",
		ID:                   "bdd20d7a-bba5-40dd-a111-9863d7908572",
		MeasurementStartTime: "2018-11-01 15:33:20",
		ProbeIP:              config.ProbeIP,
		ProbeASN:             config.ProbeASN,
		ProbeNetworkName:     config.ProbeNetworkName,
		ProbeCC:              config.ProbeCC,
		ReportID:             "",
		ResolverIP:           config.ResolverIP,
		ResolverNetworkName:  config.ResolverNetworkName,
		ResolverASN:          config.ResolverASN,
		SoftwareName:         "probe-engine",
		SoftwareVersion:      "0.1.0",
		TestKeys: &fakeTestKeys{
			ClientResolver: "91.80.37.104",
			Body: fmt.Sprintf(`
				<HTML><HEAD><TITLE>Your IP is %s</TITLE></HEAD>
				<BODY><P>Hey you, I see your IP and it's %s!</P></BODY>
			`, config.ProbeIP, config.ProbeIP),
		},
		TestName:           "dummy",
		MeasurementRuntime: 5.0565230846405,
		TestStartTime:      "2018-11-01 15:33:17",
		TestVersion:        "0.1.0",
	}
}

func TestScrubMeasurementWeAreScrubbing(t *testing.T) {
	config := makeMeasurementConfig{
		ProbeIP:             "130.192.91.211",
		ProbeASN:            "AS137",
		ProbeCC:             "IT",
		ProbeNetworkName:    "Vodafone Italia S.p.A.",
		ResolverIP:          "8.8.8.8",
		ResolverNetworkName: "Google LLC",
		ResolverASN:         "AS12345",
	}
	m := makeMeasurement(config)
	if err := ScrubMeasurement(config.ProbeIP, &m); err != nil {
		t.Fatal(err)
	}
	if m.ProbeASN != config.ProbeASN {
		t.Fatal("ProbeASN has been scrubbed")
	}
	if m.ProbeCC != config.ProbeCC {
		t.Fatal("ProbeCC has been scrubbed")
	}
	if m.ProbeIP != DefaultProbeIP {
		t.Fatal("ProbeIP HAS NOT been scrubbed")
	}
	if m.ProbeNetworkName != config.ProbeNetworkName {
		t.Fatal("ProbeNetworkName has been scrubbed")
	}
	if m.ResolverIP != config.ResolverIP {
		t.Fatal("ResolverIP has been scrubbed")
	}
	if m.ResolverNetworkName != config.ResolverNetworkName {
		t.Fatal("ResolverNetworkName has been scrubbed")
	}
	if m.ResolverASN != config.ResolverASN {
		t.Fatal("ResolverASN has been scrubbed")
	}
	data, err := json.Marshal(m)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Count(data, []byte(config.ProbeIP)) != 0 {
		t.Fatalf("ProbeIP not fully redacted: %s", string(data))
	}
}

func TestScrubInvalidIP(t *testing.T) {
	m := &Measurement{
		ProbeASN: "AS1234",
		ProbeCC:  "IT",
	}
	err := ScrubMeasurement("", &m) // invalid IP
	if !errors.Is(err, ErrInvalidProbeIP) {
		t.Fatal("not the error we expected")
	}
}
