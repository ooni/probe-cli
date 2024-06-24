package openvpn_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	vpntracex "github.com/ooni/minivpn/pkg/tracex"
	"github.com/ooni/probe-cli/v3/internal/experiment/openvpn"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
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
	if m.ExperimentVersion() != "0.1.3" {
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
				BootstrapTime:  1,
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
				BootstrapTime:  1,
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
			URL:     "openvpn://badprovider/?address=aa",
			Options: &openvpn.Config{},
		},
	}
	err := m.Run(ctx, args)
	if !errors.Is(err, openvpn.ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestVPNInput(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	// TODO(ainghazal): do a real test, get credentials etc.
}

func TestMeasurer_FetchProviderCredentials(t *testing.T) {
	t.Run("Measurer.FetchProviderCredentials calls method in session", func(t *testing.T) {
		m := openvpn.NewExperimentMeasurer().(openvpn.Measurer)

		sess := makeMockSession()
		_, err := m.FetchProviderCredentials(
			context.Background(),
			sess, "riseup")
		if err != nil {
			t.Fatal("expected no error")
		}
	})
	t.Run("Measurer.FetchProviderCredentials raises error if API calls fail", func(t *testing.T) {
		someError := errors.New("unexpected")

		m := openvpn.NewExperimentMeasurer().(openvpn.Measurer)

		sess := makeMockSession()
		sess.MockFetchOpenVPNConfig = func(context.Context, string, string) (*model.OOAPIVPNProviderConfig, error) {
			return nil, someError
		}
		_, err := m.FetchProviderCredentials(
			context.Background(),
			sess, "riseup")
		if !errors.Is(err, someError) {
			t.Fatalf("expected error %v, got %v", someError, err)
		}
	})
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
			URL:     "openvpn://riseupvpn.corp/?address=127.0.0.1:9989&transport=tcp",
			Options: &openvpn.Config{},
		},
	}
	err := m.Run(ctx, args)
	if err != nil {
		t.Fatal(err)
	}
}

func TestTimestampsFromHandshake(t *testing.T) {
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

}
