package quicping_test

import (
	"context"
	"encoding/hex"
	"errors"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/quicping"
	"github.com/ooni/probe-cli/v3/internal/engine/mockable"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
)

// FailStdLib is a failing model.UnderlyingNetworkLibrary.
type FailStdLib struct {
	conn      model.UDPLikeConn
	err       error
	writeErr  error
	readErr   error
}

// ListenUDP implements UnderlyingNetworkLibrary.ListenUDP.
func (f *FailStdLib) ListenUDP(network string, laddr *net.UDPAddr) (model.UDPLikeConn, error) {
	conn, _ := net.ListenUDP(network, laddr)
	f.conn = model.UDPLikeConn(conn)
	if f.err != nil {
		return nil, f.err
	}
	if f.writeErr != nil {
		return &mocks.UDPLikeConn{
			MockWriteTo: func(p []byte, addr net.Addr) (int, error) {
				return 0, f.writeErr
			},
			MockReadFrom: func(p []byte) (int, net.Addr, error) {
				return f.conn.ReadFrom(p)
			},
			MockSetReadDeadline: func(t time.Time) error {
				return f.conn.SetReadDeadline(t)
			},
			MockClose: func() error {
				return f.conn.Close()
			},
		}, nil
	}
	if f.readErr != nil {
		return &mocks.UDPLikeConn{
			MockWriteTo: func(p []byte, addr net.Addr) (int, error) {
				return f.conn.WriteTo(p, addr)
			},
			MockReadFrom: func(p []byte) (int, net.Addr, error) {
				return 0, nil, f.readErr
			},
			MockSetReadDeadline: func(t time.Time) error {
				return f.conn.SetReadDeadline(t)
			},
			MockClose: func() error {
				return f.conn.Close()
			},
		}, nil 
	}
	return &mocks.UDPLikeConn{}, nil
}

// LookupHost implements UnderlyingNetworkLibrary.LookupHost.
func (f *FailStdLib) LookupHost(ctx context.Context, domain string) ([]string, error) {
	return nil, f.err
}

// NewSimpleDialer implements UnderlyingNetworkLibrary.NewSimpleDialer.
func (f *FailStdLib) NewSimpleDialer(timeout time.Duration) model.SimpleDialer {
	return nil
}

func TestNewExperimentMeasurer(t *testing.T) {
	measurer := quicping.NewExperimentMeasurer(quicping.Config{})
	if measurer.ExperimentName() != "quicping" {
		t.Fatal("unexpected name")
	}
	if measurer.ExperimentVersion() != "0.1.0" {
		t.Fatal("unexpected version")
	}
}

func TestInvalidHost(t *testing.T) {
	measurer := quicping.NewExperimentMeasurer(quicping.Config{
		Port: int64(443),
	})
	measurement := new(model.Measurement)
	measurement.Input = model.MeasurementTarget("a.a.a.a")
	sess := &mockable.Session{MockableLogger: log.Log}
	err := measurer.Run(context.Background(), sess, measurement,
		model.NewPrinterCallbacks(log.Log))
	if err == nil {
		t.Fatal("expected an error here")
	}
	if _, ok := err.(*net.DNSError); !ok {
		t.Fatal("unexpected error type")
	}
}

func TestReadTimeout(t *testing.T) {
	measurer := quicping.NewExperimentMeasurer(quicping.Config{
		Port:        int64(443),
		Timeout:     int64(10),
		Repetitions: int64(2),
	})
	measurement := new(model.Measurement)
	measurement.Input = model.MeasurementTarget("google.com")
	sess := &mockable.Session{MockableLogger: log.Log}
	err := measurer.Run(context.Background(), sess, measurement,
		model.NewPrinterCallbacks(log.Log))
	if err != nil {
		t.Fatal("unexpected error", err)
	}
	tk := measurement.TestKeys.(*quicping.TestKeys)
	for i, ping := range tk.Pings {
		if ping.Failure == nil {
			t.Fatal("ping should have failed", i)
		}
		if !strings.Contains(*ping.Failure, "timeout") {
			t.Fatal("ping: unexpected error type", i, *ping.Failure)
		}
	}
}

var mock = mocks.UDPLikeConn{}

func TestListenFails(t *testing.T) {
	expected := errors.New("expected")
	measurer := quicping.NewExperimentMeasurer(quicping.Config{
		NetworkLibrary: &FailStdLib{err: expected, readErr: nil, writeErr: nil},
	})
	measurement := new(model.Measurement)
	measurement.Input = model.MeasurementTarget("google.com")
	sess := &mockable.Session{MockableLogger: log.Log}
	err := measurer.Run(context.Background(), sess, measurement,
		model.NewPrinterCallbacks(log.Log))
	if err == nil {
		t.Fatal("expected an error here")
	}
	if err != expected {
		t.Fatal("unexpected error type")
	}
}

func TestWriteFails(t *testing.T) {
	expected := errors.New("expected")
	measurer := quicping.NewExperimentMeasurer(quicping.Config{
		NetworkLibrary: &FailStdLib{err: nil, readErr: nil, writeErr: expected},
		Repetitions:    1,
	})
	measurement := new(model.Measurement)
	measurement.Input = model.MeasurementTarget("google.com")
	sess := &mockable.Session{MockableLogger: log.Log}
	err := measurer.Run(context.Background(), sess, measurement,
		model.NewPrinterCallbacks(log.Log))
	if err != nil {
		t.Fatal("unexpected error")
	}
	tk := measurement.TestKeys.(*quicping.TestKeys)
	if tk.Pings == nil || len(tk.Pings) != 1 {
		t.Fatal("not enough pings")
	}
	for i, ping := range tk.Pings {
		if ping.Failure == nil {
			t.Fatal("expected an error here, ping", i)
		}
		if !strings.Contains(*ping.Failure, "expected") {
			t.Fatal("ping: unexpected error type", i, *ping.Failure)
		}
	}
}

func TestReadFails(t *testing.T) {
	expected := errors.New("expected")
	measurer := quicping.NewExperimentMeasurer(quicping.Config{
		NetworkLibrary: &FailStdLib{err: nil, readErr: expected, writeErr: nil},
		Repetitions:    2,
	})
	measurement := new(model.Measurement)
	measurement.Input = model.MeasurementTarget("google.com")
	sess := &mockable.Session{MockableLogger: log.Log}
	err := measurer.Run(context.Background(), sess, measurement,
		model.NewPrinterCallbacks(log.Log))
	if err != nil {
		t.Fatal("unexpected error")
	}
	tk := measurement.TestKeys.(*quicping.TestKeys)
	if tk.Pings == nil || len(tk.Pings) != 2 {
		t.Fatal("not enough pings")
	}
	for i, ping := range tk.Pings {
		if ping.Failure == nil {
			t.Fatal("expected an error here, ping", i)
		}
	}
}

func TestSucess(t *testing.T) {
	measurer := quicping.NewExperimentMeasurer(quicping.Config{})
	measurement := new(model.Measurement)
	measurement.Input = model.MeasurementTarget("google.com")
	sess := &mockable.Session{MockableLogger: log.Log}
	err := measurer.Run(context.Background(), sess, measurement,
		model.NewPrinterCallbacks(log.Log))
	if err != nil {
		t.Fatal("did not expect an error here")
	}
	tk := measurement.TestKeys.(*quicping.TestKeys)
	if tk.Domain != "google.com" {
		t.Fatal("unexpected domain")
	}
	if tk.Repetitions != 10 {
		t.Fatal("unexpected number of repetitions, default is 10")
	}
	if tk.Pings == nil || len(tk.Pings) != 10 {
		t.Fatal("not enough pings")
	}
	for i, ping := range tk.Pings {
		if ping.Failure != nil {
			t.Fatal("ping failed unexpectedly", i, *ping.Failure)
		}
		if ping.SupportedVersions == nil || len(ping.SupportedVersions) == 0 {
			t.Fatal("server did not respond with supported versions")
		}
	}
	sk, err := measurer.GetSummaryKeys(measurement)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := sk.(quicping.SummaryKeys); !ok {
		t.Fatal("invalid type for summary keys")
	}
}

func TestDissect(t *testing.T) {
	//                             destID--srcID: 040b9649d3fd4c038ab6c073966f3921--44d064031288e97646451f
	versionNegotiationResponse, _ := hex.DecodeString("eb0000000010040b9649d3fd4c038ab6c073966f39210b44d064031288e97646451f00000001ff00001dff00001cff00001b")
	measurer := quicping.NewExperimentMeasurer(quicping.Config{})
	destID := "040b9649d3fd4c038ab6c073966f3921"
	_, dst, err := measurer.(*quicping.Measurer).DissectVersionNegotiation(versionNegotiationResponse)
	if err != nil {
		t.Fatal("unexpected error", err)
	}
	if dst != destID {
		t.Fatal("unexpected destination connection ID")
	}

	versionNegotiationResponse[1] = byte(0xff)
	_, _, err = measurer.(*quicping.Measurer).DissectVersionNegotiation(versionNegotiationResponse)
	if err == nil {
		t.Fatal("expected an error here", err)
	}
	if !strings.HasSuffix(err.Error(), "unexpected Version Negotiation format") {
		t.Fatal("unexpected error type", err)
	}

	versionNegotiationResponse[0] = byte(0x01)
	_, _, err = measurer.(*quicping.Measurer).DissectVersionNegotiation(versionNegotiationResponse)
	if err == nil {
		t.Fatal("expected an error here", err)
	}
	if !strings.HasSuffix(err.Error(), "not a long header packet") {
		t.Fatal("unexpected error type", err)
	}
}
