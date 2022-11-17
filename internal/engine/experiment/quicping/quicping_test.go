package quicping

import (
	"context"
	"encoding/hex"
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine/mockable"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
)

func TestNewExperimentMeasurer(t *testing.T) {
	measurer := NewExperimentMeasurer(Config{})
	if measurer.ExperimentName() != "quicping" {
		t.Fatal("unexpected name")
	}
	if measurer.ExperimentVersion() != "0.1.0" {
		t.Fatal("unexpected version")
	}
}

func TestInvalidHost(t *testing.T) {
	measurer := NewExperimentMeasurer(Config{
		Port:        443,
		Repetitions: 1,
	})
	measurement := new(model.Measurement)
	measurement.Input = model.MeasurementTarget("a.a.a.a")
	sess := &mockable.Session{MockableLogger: log.Log}
	args := &model.ExperimentArgs{
		Callbacks:   model.NewPrinterCallbacks(log.Log),
		Measurement: measurement,
		Session:     sess,
	}
	err := measurer.Run(context.Background(), args)
	if err == nil {
		t.Fatal("expected an error here")
	}
	if _, ok := err.(*net.DNSError); !ok {
		t.Fatal("unexpected error type")
	}
}

func TestURLInput(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	measurer := NewExperimentMeasurer(Config{
		Repetitions: 1,
	})
	measurement := new(model.Measurement)
	measurement.Input = model.MeasurementTarget("https://google.com/")
	sess := &mockable.Session{MockableLogger: log.Log}
	args := &model.ExperimentArgs{
		Callbacks:   model.NewPrinterCallbacks(log.Log),
		Measurement: measurement,
		Session:     sess,
	}
	err := measurer.Run(context.Background(), args)
	if err != nil {
		t.Fatal("unexpected error")
	}
	tk := measurement.TestKeys.(*TestKeys)
	if tk.Domain != "google.com" {
		t.Fatal("unexpected domain")
	}

}

func TestSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	measurer := NewExperimentMeasurer(Config{})
	measurement := new(model.Measurement)
	measurement.Input = model.MeasurementTarget("google.com")
	sess := &mockable.Session{MockableLogger: log.Log}
	args := &model.ExperimentArgs{
		Callbacks:   model.NewPrinterCallbacks(log.Log),
		Measurement: measurement,
		Session:     sess,
	}
	err := measurer.Run(context.Background(), args)
	if err != nil {
		t.Fatal("did not expect an error here")
	}
	tk := measurement.TestKeys.(*TestKeys)
	if tk.Domain != "google.com" {
		t.Fatal("unexpected domain")
	}
	if tk.Repetitions != 10 {
		t.Fatal("unexpected number of repetitions, default is 10")
	}
	if tk.Pings == nil || len(tk.Pings) != 10 {
		t.Fatal("unexpected number of pings", len(tk.Pings))
	}
	for i, ping := range tk.Pings {
		if ping.Failure != nil {
			t.Fatal("ping failed unexpectedly", i, *ping.Failure)
		}
		for _, resp := range ping.Responses {
			if resp.Failure != nil {
				t.Fatal("unexepcted response failure")
			}
			if resp.SupportedVersions == nil || len(resp.SupportedVersions) == 0 {
				t.Fatal("server did not respond with supported versions")
			}
		}
	}
	sk, err := measurer.GetSummaryKeys(measurement)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := sk.(SummaryKeys); !ok {
		t.Fatal("invalid type for summary keys")
	}
}

func TestWithCancelledContext(t *testing.T) {
	measurer := NewExperimentMeasurer(Config{})
	measurement := new(model.Measurement)
	measurement.Input = model.MeasurementTarget("google.com")
	sess := &mockable.Session{MockableLogger: log.Log}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	args := &model.ExperimentArgs{
		Callbacks:   model.NewPrinterCallbacks(log.Log),
		Measurement: measurement,
		Session:     sess,
	}
	err := measurer.Run(ctx, args)
	if err != nil {
		t.Fatal("did not expect an error here")
	}
	tk := measurement.TestKeys.(*TestKeys)
	if len(tk.Pings) > 0 {
		t.Fatal("there should not be any measurements")
	}
}

func TestListenFails(t *testing.T) {
	expected := errors.New("expected")
	measurer := NewExperimentMeasurer(Config{
		netListenUDP: func(network string, laddr *net.UDPAddr) (model.UDPLikeConn, error) {
			return nil, expected
		},
	})
	measurement := new(model.Measurement)
	measurement.Input = model.MeasurementTarget("google.com")
	sess := &mockable.Session{MockableLogger: log.Log}
	args := &model.ExperimentArgs{
		Callbacks:   model.NewPrinterCallbacks(log.Log),
		Measurement: measurement,
		Session:     sess,
	}
	err := measurer.Run(context.Background(), args)
	if err == nil {
		t.Fatal("expected an error here")
	}
	if err != expected {
		t.Fatal("unexpected error type")
	}
}

func TestWriteFails(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	expected := errors.New("expected")
	setDeadlineCalled := false
	closeCalled := false
	pconn := &mocks.UDPLikeConn{
		MockReadFrom: func(p []byte) (int, net.Addr, error) {
			source := make([]byte, len(p))
			copy(p, source)
			return len(p), &mocks.Addr{}, nil
		},
		MockSetDeadline: func(t time.Time) error {
			setDeadlineCalled = true
			return nil
		},
		MockClose: func() error {
			closeCalled = true
			return nil
		},
		MockWriteTo: func(p []byte, addr net.Addr) (int, error) {
			return 0, expected
		},
	}
	measurer := NewExperimentMeasurer(Config{
		netListenUDP: func(network string, laddr *net.UDPAddr) (model.UDPLikeConn, error) {
			return pconn, nil
		},
		Repetitions: 1,
	})
	measurement := new(model.Measurement)
	measurement.Input = model.MeasurementTarget("google.com")
	sess := &mockable.Session{MockableLogger: log.Log}
	args := &model.ExperimentArgs{
		Callbacks:   model.NewPrinterCallbacks(log.Log),
		Measurement: measurement,
		Session:     sess,
	}
	err := measurer.Run(context.Background(), args)
	if err != nil {
		t.Fatal("unexpected error")
	}
	tk := measurement.TestKeys.(*TestKeys)
	if tk.Pings == nil || len(tk.Pings) != 1 {
		t.Fatal("unexpected number of pings", len(tk.Pings))
	}
	for i, ping := range tk.Pings {
		if ping.Failure == nil {
			t.Fatal("expected an error here, ping", i)
		}
		if !strings.Contains(*ping.Failure, "expected") {
			t.Fatal("ping: unexpected error type", i, *ping.Failure)
		}
	}
	if !setDeadlineCalled {
		t.Fatal("did not call set deadline")
	}
	if !closeCalled {
		t.Fatal("did not call close")
	}
}

func TestReadFails(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	setDeadlineCalled := false
	closeCalled := false
	expected := errors.New("expected")
	pconn := &mocks.UDPLikeConn{
		MockReadFrom: func(p []byte) (int, net.Addr, error) {
			return 0, nil, expected
		},
		MockSetDeadline: func(t time.Time) error {
			setDeadlineCalled = true
			return nil
		},
		MockClose: func() error {
			closeCalled = true
			return nil
		},
		MockWriteTo: func(p []byte, addr net.Addr) (int, error) {
			return len(p), nil
		},
	}
	measurer := NewExperimentMeasurer(Config{
		netListenUDP: func(network string, laddr *net.UDPAddr) (model.UDPLikeConn, error) {
			return pconn, nil
		},
		Repetitions: 1,
	})
	measurement := new(model.Measurement)
	measurement.Input = model.MeasurementTarget("google.com")
	sess := &mockable.Session{MockableLogger: log.Log}
	args := &model.ExperimentArgs{
		Callbacks:   model.NewPrinterCallbacks(log.Log),
		Measurement: measurement,
		Session:     sess,
	}
	err := measurer.Run(context.Background(), args)
	if err != nil {
		t.Fatal("unexpected error")
	}
	tk := measurement.TestKeys.(*TestKeys)
	if tk.Pings == nil || len(tk.Pings) != 1 {
		t.Fatal("unexpected number of pings", len(tk.Pings))
	}
	for i, ping := range tk.Pings {
		if ping.Failure == nil {
			t.Fatal("expected an error here, ping", i)
		}
	}
	if !setDeadlineCalled {
		t.Fatal("did not call set deadline")
	}
	if !closeCalled {
		t.Fatal("did not call close")
	}
}

func TestNoResponse(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	measurer := NewExperimentMeasurer(Config{
		Repetitions: 1,
	})
	measurement := new(model.Measurement)
	measurement.Input = model.MeasurementTarget("ooni.org")
	sess := &mockable.Session{MockableLogger: log.Log}
	args := &model.ExperimentArgs{
		Callbacks:   model.NewPrinterCallbacks(log.Log),
		Measurement: measurement,
		Session:     sess,
	}
	err := measurer.Run(context.Background(), args)
	if err != nil {
		t.Fatal("did not expect an error here")
	}
	tk := measurement.TestKeys.(*TestKeys)
	if tk.Pings == nil || len(tk.Pings) != 1 {
		t.Fatal("unexpected number of pings", len(tk.Pings))
	}
	if tk.Pings[0].Failure == nil {
		t.Fatal("expected an error here")
	}
	if *tk.Pings[0].Failure != "generic_timeout_error" {
		t.Fatal("unexpected error type")
	}
}

func TestDissect(t *testing.T) {
	// destID--srcID: 040b9649d3fd4c038ab6c073966f3921--44d064031288e97646451f
	versionNegotiationResponse, _ := hex.DecodeString("eb0000000010040b9649d3fd4c038ab6c073966f39210b44d064031288e97646451f00000001ff00001dff00001cff00001b")
	measurer := NewExperimentMeasurer(Config{})
	destID := "040b9649d3fd4c038ab6c073966f3921"
	_, dst, err := measurer.(*Measurer).dissectVersionNegotiation(versionNegotiationResponse)
	if err != nil {
		t.Fatal("unexpected error", err)
	}
	if hex.EncodeToString(dst) != destID {
		t.Fatal("unexpected destination connection ID")
	}

	versionNegotiationResponse[1] = byte(0xff)
	_, _, err = measurer.(*Measurer).dissectVersionNegotiation(versionNegotiationResponse)
	if err == nil {
		t.Fatal("expected an error here", err)
	}
	if !strings.HasSuffix(err.Error(), "unexpected Version Negotiation format") {
		t.Fatal("unexpected error type", err)
	}

	versionNegotiationResponse[0] = byte(0x01)
	_, _, err = measurer.(*Measurer).dissectVersionNegotiation(versionNegotiationResponse)
	if err == nil {
		t.Fatal("expected an error here", err)
	}
	if !strings.HasSuffix(err.Error(), "not a long header packet") {
		t.Fatal("unexpected error type", err)
	}
}
