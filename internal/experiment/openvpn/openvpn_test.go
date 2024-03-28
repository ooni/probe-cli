package openvpn_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/experiment/example"
	"github.com/ooni/probe-cli/v3/internal/experiment/openvpn"
	"github.com/ooni/probe-cli/v3/internal/legacy/mockable"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"

	vpntracex "github.com/ooni/minivpn/pkg/tracex"
)

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
				StartTime:      time.Now(),
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
	if m.ExperimentName() != "openvpn" {
		t.Fatal("invalid ExperimentName")
	}
	if m.ExperimentVersion() != "0.1.0" {
		t.Fatal("invalid ExperimentVersion")
	}
	ctx := context.Background()
	sess := &mocks.Session{}
	callbacks := model.NewPrinterCallbacks(sess.Logger())
	measurement := new(model.Measurement)
	args := &model.ExperimentArgs{
		Callbacks:   callbacks,
		Measurement: measurement,
		Session:     sess,
	}
	err := m.Run(ctx, args)
	if err != nil {
		t.Fatal(err)
	}
}

func TestFailure(t *testing.T) {
	m := example.NewExperimentMeasurer(example.Config{
		SleepTime:   int64(2 * time.Millisecond),
		ReturnError: true,
	}, "example")
	ctx := context.Background()
	sess := &mockable.Session{MockableLogger: log.Log}
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
