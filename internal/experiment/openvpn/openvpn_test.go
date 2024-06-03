package openvpn_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	vpnconfig "github.com/ooni/minivpn/pkg/config"
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

func TestMaybeGetCredentialsFromOptions(t *testing.T) {
	t.Run("cert auth returns false if cert, key and ca are not all provided", func(t *testing.T) {
		cfg := openvpn.Config{
			SafeCA:   "base64:Zm9v",
			SafeCert: "base64:Zm9v",
		}
		ok, err := openvpn.MaybeGetCredentialsFromOptions(cfg, &vpnconfig.OpenVPNOptions{}, openvpn.AuthCertificate)
		if err != nil {
			t.Fatal("should not raise error")
		}
		if ok {
			t.Fatal("expected false")
		}
	})
	t.Run("cert auth returns ok if cert, key and ca are all provided", func(t *testing.T) {
		cfg := openvpn.Config{
			SafeCA:   "base64:Zm9v",
			SafeCert: "base64:Zm9v",
			SafeKey:  "base64:Zm9v",
		}
		opts := &vpnconfig.OpenVPNOptions{}
		ok, err := openvpn.MaybeGetCredentialsFromOptions(cfg, opts, openvpn.AuthCertificate)
		if err != nil {
			t.Fatalf("expected err=nil, got %v", err)
		}
		if !ok {
			t.Fatal("expected true")
		}
		if diff := cmp.Diff(opts.CA, []byte("foo")); diff != "" {
			t.Fatal(diff)
		}
		if diff := cmp.Diff(opts.Cert, []byte("foo")); diff != "" {
			t.Fatal(diff)
		}
		if diff := cmp.Diff(opts.Key, []byte("foo")); diff != "" {
			t.Fatal(diff)
		}
	})
	t.Run("cert auth returns false and error if CA base64 is bad blob", func(t *testing.T) {
		cfg := openvpn.Config{
			SafeCA:   "base64:Zm9vaaa",
			SafeCert: "base64:Zm9v",
			SafeKey:  "base64:Zm9v",
		}
		opts := &vpnconfig.OpenVPNOptions{}
		ok, err := openvpn.MaybeGetCredentialsFromOptions(cfg, opts, openvpn.AuthCertificate)
		if ok {
			t.Fatal("expected false")
		}
		if !errors.Is(err, openvpn.ErrBadBase64Blob) {
			t.Fatalf("expected err=ErrBase64Blob, got %v", err)
		}
	})
	t.Run("cert auth returns false and error if key base64 is bad blob", func(t *testing.T) {
		cfg := openvpn.Config{
			SafeCA:   "base64:Zm9v",
			SafeCert: "base64:Zm9v",
			SafeKey:  "base64:Zm9vaaa",
		}
		opts := &vpnconfig.OpenVPNOptions{}
		ok, err := openvpn.MaybeGetCredentialsFromOptions(cfg, opts, openvpn.AuthCertificate)
		if ok {
			t.Fatal("expected false")
		}
		if !errors.Is(err, openvpn.ErrBadBase64Blob) {
			t.Fatalf("expected err=ErrBase64Blob, got %v", err)
		}
	})
	t.Run("cert auth returns false and error if cert base64 is bad blob", func(t *testing.T) {
		cfg := openvpn.Config{
			SafeCA:   "base64:Zm9v",
			SafeCert: "base64:Zm9vaaa",
			SafeKey:  "base64:Zm9v",
		}
		opts := &vpnconfig.OpenVPNOptions{}
		ok, err := openvpn.MaybeGetCredentialsFromOptions(cfg, opts, openvpn.AuthCertificate)
		if ok {
			t.Fatal("expected false")
		}
		if !errors.Is(err, openvpn.ErrBadBase64Blob) {
			t.Fatalf("expected err=ErrBase64Blob, got %v", err)
		}
	})
	t.Run("userpass auth returns error, not yet implemented", func(t *testing.T) {
		cfg := openvpn.Config{}
		ok, err := openvpn.MaybeGetCredentialsFromOptions(cfg, &vpnconfig.OpenVPNOptions{}, openvpn.AuthUserPass)
		if ok {
			t.Fatal("expected false")
		}
		if err != nil {
			t.Fatalf("expected err=nil, got %v", err)
		}
	})

}

func TestGetCredentialsFromOptionsOrAPI(t *testing.T) {
	t.Run("non-registered provider raises error", func(t *testing.T) {
		m := openvpn.NewExperimentMeasurer(openvpn.Config{}, "openvpn").(openvpn.Measurer)
		ctx := context.Background()
		sess := makeMockSession()
		opts, err := m.GetCredentialsFromOptionsOrAPI(ctx, sess, "nsa")
		if !errors.Is(err, openvpn.ErrInvalidInput) {
			t.Fatalf("expected err=ErrInvalidInput, got %v", err)
		}
		if opts != nil {
			t.Fatal("expected opts=nil")
		}
	})
	t.Run("providers with userpass auth method raise error, not yet implemented", func(t *testing.T) {
		m := openvpn.NewExperimentMeasurer(openvpn.Config{}, "openvpn").(openvpn.Measurer)
		ctx := context.Background()
		sess := makeMockSession()
		opts, err := m.GetCredentialsFromOptionsOrAPI(ctx, sess, "tunnelbear")
		if !errors.Is(err, openvpn.ErrInvalidInput) {
			t.Fatalf("expected err=ErrInvalidInput, got %v", err)
		}
		if opts != nil {
			t.Fatal("expected opts=nil")
		}
	})
	t.Run("known cert auth provider and creds in options is ok", func(t *testing.T) {
		config := openvpn.Config{
			SafeCA:   "base64:Zm9v",
			SafeCert: "base64:Zm9v",
			SafeKey:  "base64:Zm9v",
		}
		m := openvpn.NewExperimentMeasurer(config, "openvpn").(openvpn.Measurer)
		ctx := context.Background()
		sess := makeMockSession()
		opts, err := m.GetCredentialsFromOptionsOrAPI(ctx, sess, "riseup")
		if err != nil {
			t.Fatalf("expected err=nil, got %v", err)
		}
		if opts == nil {
			t.Fatal("expected non-nil options")
		}
	})
	t.Run("known cert auth provider and bad creds in options returns error", func(t *testing.T) {
		config := openvpn.Config{
			SafeCA:   "base64:Zm9v",
			SafeCert: "base64:Zm9v",
			SafeKey:  "base64:Zm9vaaa",
		}
		m := openvpn.NewExperimentMeasurer(config, "openvpn").(openvpn.Measurer)
		ctx := context.Background()
		sess := makeMockSession()
		opts, err := m.GetCredentialsFromOptionsOrAPI(ctx, sess, "riseup")
		if !errors.Is(err, openvpn.ErrBadBase64Blob) {
			t.Fatalf("expected err=ErrBadBase64, got %v", err)
		}
		if opts != nil {
			t.Fatal("expected nil opts")
		}
	})
	t.Run("known cert auth provider with null options hits the api", func(t *testing.T) {
		config := openvpn.Config{}
		m := openvpn.NewExperimentMeasurer(config, "openvpn").(openvpn.Measurer)
		ctx := context.Background()
		sess := makeMockSession()
		opts, err := m.GetCredentialsFromOptionsOrAPI(ctx, sess, "riseup")
		if err != nil {
			t.Fatalf("expected err=nil, got %v", err)
		}
		if opts == nil {
			t.Fatalf("expected not-nil options, got %v", opts)
		}
	})
	t.Run("known cert auth provider with null options hits the api and raises error if api fails", func(t *testing.T) {
		config := openvpn.Config{}
		m := openvpn.NewExperimentMeasurer(config, "openvpn").(openvpn.Measurer)
		ctx := context.Background()

		someError := errors.New("some error")
		sess := makeMockSession()
		sess.MockFetchOpenVPNConfig = func(context.Context, string, string) (*model.OOAPIVPNProviderConfig, error) {
			return nil, someError
		}

		opts, err := m.GetCredentialsFromOptionsOrAPI(ctx, sess, "riseup")
		if !errors.Is(err, someError) {
			t.Fatalf("expected err=someError, got %v", err)
		}
		if opts != nil {
			t.Fatalf("expected nil options, got %v", opts)
		}
	})
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
			OpenVPNHandshake: &model.ArchivalOpenVPNHandshakeResult{
				BootstrapTime:  1,
				Endpoint:       "aa",
				IP:             "1.1.1.1",
				Port:           1194,
				Transport:      "tcp",
				Provider:       "unknown",
				OpenVPNOptions: model.ArchivalOpenVPNOptions{},
				Status:         model.ArchivalOpenVPNConnectStatus{},
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
		tk.OpenVPNHandshake = []*model.ArchivalOpenVPNHandshakeResult{
			{Status: model.ArchivalOpenVPNConnectStatus{Success: true}},
			{Status: model.ArchivalOpenVPNConnectStatus{Success: true}},
			{Status: model.ArchivalOpenVPNConnectStatus{Success: true}},
		}
		if tk.AllConnectionsSuccessful() != true {
			t.Fatal("expected all connections successful")
		}
	})
	t.Run("one failure", func(t *testing.T) {
		tk := openvpn.NewTestKeys()
		tk.OpenVPNHandshake = []*model.ArchivalOpenVPNHandshakeResult{
			{Status: model.ArchivalOpenVPNConnectStatus{Success: false}},
			{Status: model.ArchivalOpenVPNConnectStatus{Success: true}},
			{Status: model.ArchivalOpenVPNConnectStatus{Success: true}},
		}
		if tk.AllConnectionsSuccessful() != false {
			t.Fatal("expected false")
		}
	})
	t.Run("all failures", func(t *testing.T) {
		tk := openvpn.NewTestKeys()
		tk.OpenVPNHandshake = []*model.ArchivalOpenVPNHandshakeResult{
			{Status: model.ArchivalOpenVPNConnectStatus{Success: false}},
			{Status: model.ArchivalOpenVPNConnectStatus{Success: false}},
			{Status: model.ArchivalOpenVPNConnectStatus{Success: false}},
		}
		if tk.AllConnectionsSuccessful() != false {
			t.Fatal("expected false")
		}
	})
}

func TestBadInputFailure(t *testing.T) {
	m := openvpn.NewExperimentMeasurer(openvpn.Config{}, "openvpn")
	ctx := context.Background()
	sess := makeMockSession()
	callbacks := model.NewPrinterCallbacks(sess.Logger())
	measurement := new(model.Measurement)
	measurement.Input = "openvpn://badprovider/?address=aa"
	args := &model.ExperimentArgs{
		Callbacks:   callbacks,
		Measurement: measurement,
		Session:     sess,
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
	// TODO -- do a real test, get credentials etc.
}

func TestMeasurer_FetchProviderCredentials(t *testing.T) {
	t.Run("Measurer.FetchProviderCredentials calls method in session", func(t *testing.T) {
		m := openvpn.NewExperimentMeasurer(
			openvpn.Config{},
			"openvpn").(openvpn.Measurer)

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

		m := openvpn.NewExperimentMeasurer(
			openvpn.Config{},
			"openvpn").(openvpn.Measurer)

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
	m := openvpn.NewExperimentMeasurer(openvpn.Config{
		Provider: "riseup",
		SafeCA:   "base64:Zm9v",
		SafeKey:  "base64:Zm9v",
		SafeCert: "base64:Zm9v",
	}, "openvpn")
	ctx := context.Background()
	sess := makeMockSession()
	callbacks := model.NewPrinterCallbacks(sess.Logger())
	measurement := new(model.Measurement)
	measurement.Input = "openvpn://riseupvpn.corp/?address=127.0.0.1:9989&transport=tcp"
	args := &model.ExperimentArgs{
		Callbacks:   callbacks,
		Measurement: measurement,
		Session:     sess,
	}
	if sess.Logger() == nil {
		t.Fatal("logger should not be nil")
	}
	fmt.Println(ctx, args, m)

	err := m.Run(ctx, args)
	if err != nil {
		t.Fatal(err)
	}
}
