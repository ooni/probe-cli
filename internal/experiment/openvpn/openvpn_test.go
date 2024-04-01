package openvpn_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/experiment/example"
	"github.com/ooni/probe-cli/v3/internal/experiment/openvpn"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"

	vpntracex "github.com/ooni/minivpn/pkg/tracex"
)

func makeMockSession() *mocks.Session {
	return &mocks.Session{
		MockLogger: func() model.Logger {
			return model.DiscardLogger
		},
		MockFetchOpenVPNConfig: func(context.Context, string, string) (*model.OOAPIVPNProviderConfig, error) {
			return &model.OOAPIVPNProviderConfig{
				Provider: "provider",
				Config: &struct {
					CA       string "json:\"ca\""
					Cert     string "json:\"cert,omitempty\""
					Key      string "json:\"key,omitempty\""
					Username string "json:\"username,omitempty\""
					Password string "json:\"password,omitempty\""
				}{
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
	m := openvpn.NewExperimentMeasurer(openvpn.Config{}, "openvpn")
	if m.ExperimentName() != "openvpn" {
		t.Fatal("invalid ExperimentName")
	}
	if m.ExperimentVersion() != "0.1.1" {
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

// TODO refactoring tests -----------------------------------------------

func TestAddConnectionTestKeys(t *testing.T) {
	t.Run("append connection result to empty keys", func(t *testing.T) {
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
			OpenVPNHandshake: &openvpn.ArchivalOpenVPNHandshakeResult{
				BootstrapTime:  1,
				Endpoint:       "aa",
				IP:             "1.1.1.1",
				Port:           1194,
				Transport:      "tcp",
				Provider:       "unknown",
				OpenVPNOptions: openvpn.OpenVPNOptions{},
				Status:         openvpn.ArchivalOpenVPNConnectStatus{},
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
}

func TestAllConnectionsSuccessful(t *testing.T) {
	t.Run("all success", func(t *testing.T) {
		tk := openvpn.NewTestKeys()
		tk.OpenVPNHandshake = []*openvpn.ArchivalOpenVPNHandshakeResult{
			{Status: openvpn.ArchivalOpenVPNConnectStatus{Success: true}},
			{Status: openvpn.ArchivalOpenVPNConnectStatus{Success: true}},
			{Status: openvpn.ArchivalOpenVPNConnectStatus{Success: true}},
		}
		if tk.AllConnectionsSuccessful() != true {
			t.Fatal("expected all connections successful")
		}
	})
	t.Run("one failure", func(t *testing.T) {
		tk := openvpn.NewTestKeys()
		tk.OpenVPNHandshake = []*openvpn.ArchivalOpenVPNHandshakeResult{
			{Status: openvpn.ArchivalOpenVPNConnectStatus{Success: false}},
			{Status: openvpn.ArchivalOpenVPNConnectStatus{Success: true}},
			{Status: openvpn.ArchivalOpenVPNConnectStatus{Success: true}},
		}
		if tk.AllConnectionsSuccessful() != false {
			t.Fatal("expected false")
		}
	})
	t.Run("all failures", func(t *testing.T) {
		tk := openvpn.NewTestKeys()
		tk.OpenVPNHandshake = []*openvpn.ArchivalOpenVPNHandshakeResult{
			{Status: openvpn.ArchivalOpenVPNConnectStatus{Success: false}},
			{Status: openvpn.ArchivalOpenVPNConnectStatus{Success: false}},
			{Status: openvpn.ArchivalOpenVPNConnectStatus{Success: false}},
		}
		if tk.AllConnectionsSuccessful() != false {
			t.Fatal("expected false")
		}
	})

}

func TestSuccess(t *testing.T) {
	m := openvpn.NewExperimentMeasurer(openvpn.Config{}, "openvpn")
	ctx := context.Background()
	sess := makeMockSession()
	callbacks := model.NewPrinterCallbacks(sess.Logger())
	measurement := new(model.Measurement)
	args := &model.ExperimentArgs{
		Callbacks:   callbacks,
		Measurement: measurement,
		Session:     sess,
	}
	// TODO: mock runner
	err := m.Run(ctx, args)
	if err != nil {
		t.Fatal(err)
	}
}

// TODO -- test incorrect certs failure.
func TestBadInputFailure(t *testing.T) {
	m := openvpn.NewExperimentMeasurer(openvpn.Config{}, "openvpn")
	ctx := context.Background()
	sess := &mocks.Session{
		MockLogger: func() model.Logger {
			return model.DiscardLogger
		},
	}
	callbacks := model.NewPrinterCallbacks(sess.Logger())
	args := &model.ExperimentArgs{
		Callbacks:   callbacks,
		Measurement: new(model.Measurement),
		Session:     sess,
	}
	err := m.Run(ctx, args)
	if !errors.Is(err, example.ErrFailure) {
		t.Fatal("expected an error here")
	}
}

func TestVPNInput(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	// TODO -- do a real test
}
