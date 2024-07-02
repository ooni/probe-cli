package openvpn_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/google/go-cmp/cmp"
	vpntracex "github.com/ooni/minivpn/pkg/tracex"
	"github.com/ooni/probe-cli/v3/internal/experiment/openvpn"
	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/targetloading"
)

func makeMockSession() *mocks.Session {
	return &mocks.Session{
		MockLogger: func() model.Logger {
			return model.DiscardLogger
		},
		MockFetchOpenVPNConfig: func(context.Context, string, string) (*model.OOAPIVPNProviderConfig, error) {
			return &model.OOAPIVPNProviderConfig{
				Provider: "provider",
				Config: &model.OOAPIVPNConfig{
					CA:   "ca",
					Cert: "cert",
					Key:  "key",
				},
				Inputs:      []string{},
				DateUpdated: time.Now(),
			}, nil
		},
	}
}

func TestNewExperimentMeasurer(t *testing.T) {
	m := openvpn.NewExperimentMeasurer()
	if m.ExperimentName() != "openvpn" {
		t.Fatal("invalid ExperimentName")
	}
	if m.ExperimentVersion() != "0.1.4" {
		t.Fatal("invalid ExperimentVersion")
	}
}

func TestNewTestKeys(t *testing.T) {
	tk := openvpn.NewTestKeys()
	if tk.Success != false {
		t.Fatal("default success should be false")
	}
	if tk.NetworkEvents == nil {
		t.Fatal("NetworkEvents not initialized")
	}
	if tk.TCPConnect == nil {
		t.Fatal("TCPConnect not initialized")
	}
	if tk.OpenVPNHandshake == nil {
		t.Fatal("OpenVPNHandshake not initialized")
	}
}

func TestAddConnectionTestKeys(t *testing.T) {
	t.Run("append tcp connection result to empty keys", func(t *testing.T) {
		tk := openvpn.NewTestKeys()
		sc := &openvpn.SingleConnection{
			TCPConnect: &model.ArchivalTCPConnectResult{
				IP:   "1.1.1.1",
				Port: 1194,
				Status: model.ArchivalTCPConnectStatus{
					Blocked: new(bool),
					Failure: new(string),
					Success: false,
				},
				T0:            0.1,
				T:             0.9,
				Tags:          []string{},
				TransactionID: 1,
			},
			OpenVPNHandshake: &model.ArchivalOpenVPNHandshakeResult{
				HandshakeTime:  1,
				Endpoint:       "aa",
				Failure:        nil,
				IP:             "1.1.1.1",
				Port:           1194,
				Transport:      "tcp",
				Provider:       "unknown",
				OpenVPNOptions: model.ArchivalOpenVPNOptions{},
				T0:             0,
				T:              0,
				Tags:           []string{},
				TransactionID:  1,
			},
			NetworkEvents: []*vpntracex.Event{},
		}
		tk.AddConnectionTestKeys(sc)
		if diff := cmp.Diff(tk.TCPConnect[0], sc.TCPConnect); diff != "" {
			t.Fatal(diff)
		}
		if diff := cmp.Diff(tk.OpenVPNHandshake[0], sc.OpenVPNHandshake); diff != "" {
			t.Fatal(diff)
		}
		if diff := cmp.Diff(tk.NetworkEvents, sc.NetworkEvents); diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("append udp connection result to empty keys", func(t *testing.T) {
		tk := openvpn.NewTestKeys()
		sc := &openvpn.SingleConnection{
			TCPConnect: nil,
			OpenVPNHandshake: &model.ArchivalOpenVPNHandshakeResult{
				HandshakeTime:  1,
				Endpoint:       "aa",
				Failure:        nil,
				IP:             "1.1.1.1",
				Port:           1194,
				Transport:      "udp",
				Provider:       "unknown",
				OpenVPNOptions: model.ArchivalOpenVPNOptions{},
				T0:             0,
				T:              0,
				Tags:           []string{},
				TransactionID:  1,
			},
			NetworkEvents: []*vpntracex.Event{},
		}
		tk.AddConnectionTestKeys(sc)
		if len(tk.TCPConnect) != 0 {
			t.Fatal("expected empty tcpconnect")
		}
		if diff := cmp.Diff(tk.OpenVPNHandshake[0], sc.OpenVPNHandshake); diff != "" {
			t.Fatal(diff)
		}
		if diff := cmp.Diff(tk.NetworkEvents, sc.NetworkEvents); diff != "" {
			t.Fatal(diff)
		}
	})
}

func TestAllConnectionsSuccessful(t *testing.T) {
	t.Run("all success", func(t *testing.T) {
		tk := openvpn.NewTestKeys()
		tk.OpenVPNHandshake = []*model.ArchivalOpenVPNHandshakeResult{
			{Failure: nil},
			{Failure: nil},
			{Failure: nil},
		}
		if tk.AllConnectionsSuccessful() != true {
			t.Fatal("expected all connections successful")
		}
	})
	t.Run("one failure", func(t *testing.T) {
		fail := "uh"
		tk := openvpn.NewTestKeys()
		tk.OpenVPNHandshake = []*model.ArchivalOpenVPNHandshakeResult{
			{Failure: &fail},
			{Failure: nil},
			{Failure: nil},
		}
		if tk.AllConnectionsSuccessful() != false {
			t.Fatal("expected false")
		}
	})
	t.Run("all failures", func(t *testing.T) {
		fail := "uh"
		tk := openvpn.NewTestKeys()
		tk.OpenVPNHandshake = []*model.ArchivalOpenVPNHandshakeResult{
			{Failure: &fail},
			{Failure: &fail},
			{Failure: &fail},
		}
		if tk.AllConnectionsSuccessful() != false {
			t.Fatal("expected false")
		}
	})
}

func TestOpenVPNFailsWithInvalidInputType(t *testing.T) {
	measurer := openvpn.NewExperimentMeasurer()
	args := &model.ExperimentArgs{
		Callbacks:   model.NewPrinterCallbacks(log.Log),
		Measurement: new(model.Measurement),
		Session:     makeMockSession(),
		Target:      &model.OOAPIURLInfo{}, // not the input type we expect
	}
	err := measurer.Run(context.Background(), args)
	if !errors.Is(err, openvpn.ErrInvalidInputType) {
		t.Fatal("expected input error")
	}
}

func TestBadTargetURLFailure(t *testing.T) {
	m := openvpn.NewExperimentMeasurer()
	ctx := context.Background()
	sess := makeMockSession()
	callbacks := model.NewPrinterCallbacks(sess.Logger())
	measurement := new(model.Measurement)
	args := &model.ExperimentArgs{
		Callbacks:   callbacks,
		Measurement: measurement,
		Session:     sess,
		Target: &openvpn.Target{
			URL:    "openvpn://badprovider/?address=aa",
			Config: &openvpn.Config{},
		},
	}
	err := m.Run(ctx, args)
	if !errors.Is(err, targetloading.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestSuccess(t *testing.T) {
	m := openvpn.NewExperimentMeasurer()
	ctx := context.Background()
	sess := makeMockSession()
	callbacks := model.NewPrinterCallbacks(sess.Logger())
	measurement := new(model.Measurement)
	args := &model.ExperimentArgs{
		Callbacks:   callbacks,
		Measurement: measurement,
		Session:     sess,
		Target: &openvpn.Target{
			URL:    "openvpn://riseupvpn.corp/?address=127.0.0.1:9989&transport=tcp",
			Config: &openvpn.Config{},
		},
	}
	err := m.Run(ctx, args)
	if err != nil {
		t.Fatal(err)
	}
}

func TestTimestampsFromHandshake(t *testing.T) {
	t.Run("with more than a single event (common case)", func(t *testing.T) {
		events := []*vpntracex.Event{{AtTime: 0}, {AtTime: 1}, {AtTime: 2}}
		t0, tlast, duration := openvpn.TimestampsFromHandshake(events)
		if t0 != 0 {
			t.Fatal("expected t0 == 0")
		}
		if tlast != 2.0 {
			t.Fatal("expected t == 2")
		}
		if duration != 2 {
			t.Fatal("expected duration == 2")
		}
	})

	t.Run("with a single event", func(t *testing.T) {
		events := []*vpntracex.Event{{AtTime: 1}}
		t0, tlast, duration := openvpn.TimestampsFromHandshake(events)
		if t0 != 1.0 {
			t.Fatal("expected t0 == 1.0")
		}
		if tlast != 1.0 {
			t.Fatal("expected t == 1.0")
		}
		if duration != 0 {
			t.Fatal("expected duration == 0")
		}
	})

	t.Run("with no events", func(t *testing.T) {
		events := []*vpntracex.Event{}
		t0, tlast, duration := openvpn.TimestampsFromHandshake(events)
		if t0 != 0 {
			t.Fatal("expected t0 == 0")
		}
		if tlast != 0 {
			t.Fatal("expected t == 0")
		}
		if duration != 0 {
			t.Fatal("expected duration == 0")
		}
	})
}

func TestBootstrapTimeWithNoFailure(t *testing.T) {
	bootstrapTime := 1.2305
	tk := openvpn.NewTestKeys()
	sc := &openvpn.SingleConnection{
		BootstrapTime: bootstrapTime,
		TCPConnect: &model.ArchivalTCPConnectResult{
			IP:   "1.1.1.1",
			Port: 1194,
			Status: model.ArchivalTCPConnectStatus{
				Blocked: new(bool),
				Failure: new(string),
				Success: false,
			},
			T0:            0.1,
			T:             0.9,
			Tags:          []string{},
			TransactionID: 1,
		},
		OpenVPNHandshake: &model.ArchivalOpenVPNHandshakeResult{
			HandshakeTime:  1.20,
			Endpoint:       "aa",
			Failure:        nil,
			IP:             "1.1.1.1",
			Port:           1194,
			Transport:      "tcp",
			Provider:       "unknown",
			OpenVPNOptions: model.ArchivalOpenVPNOptions{},
			T0:             0.03,
			T:              1.23,
			Tags:           []string{},
			TransactionID:  1,
		},
		NetworkEvents: []*vpntracex.Event{},
	}
	tk.AddConnectionTestKeys(sc)

	if tk.Failure != nil {
		t.Fatal("expected nil failure")
	}
	if tk.BootstrapTime != bootstrapTime {
		t.Fatal("wrong bootstrap time")
	}
	if tk.Tunnel != "openvpn" {
		t.Fatal("tunnel should be openvpn")
	}
}

func TestBootstrapTimeWithFailure(t *testing.T) {
	bootstrapTime := 6.1

	handshakeError := errors.New("mocked error")
	handshakeFailure := measurexlite.NewFailure(handshakeError)

	tk := openvpn.NewTestKeys()
	sc := &openvpn.SingleConnection{
		BootstrapTime: bootstrapTime,
		TCPConnect: &model.ArchivalTCPConnectResult{
			IP:   "1.1.1.1",
			Port: 1194,
			Status: model.ArchivalTCPConnectStatus{
				Blocked: new(bool),
				Failure: new(string),
				Success: false,
			},
			T0:            0.1,
			T:             0.9,
			Tags:          []string{},
			TransactionID: 1,
		},
		OpenVPNHandshake: &model.ArchivalOpenVPNHandshakeResult{
			HandshakeTime:  1.20,
			Endpoint:       "aa",
			Failure:        handshakeFailure,
			IP:             "1.1.1.1",
			Port:           1194,
			Transport:      "tcp",
			Provider:       "unknown",
			OpenVPNOptions: model.ArchivalOpenVPNOptions{},
			T0:             0.03,
			T:              1.23,
			Tags:           []string{},
			TransactionID:  1,
		},
		NetworkEvents: []*vpntracex.Event{},
	}
	tk.AddConnectionTestKeys(sc)

	if tk.Failure != handshakeFailure {
		t.Fatalf("expected handshake failure, got %v", tk.Failure)
	}
	if tk.BootstrapTime != 0 {
		t.Fatalf("wrong bootstrap time: expected 0, got %v", tk.BootstrapTime)
	}
	if tk.Tunnel != "openvpn" {
		t.Fatal("tunnel should be openvpn")
	}
}

func TestVPNInput(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	// TODO(ainghazal): do a real test, get credentials etc.
}
