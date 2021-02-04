package psiphonx_test

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/apex/log"
	engine "github.com/ooni/probe-cli/v3/internal/engine"
	"github.com/ooni/probe-cli/v3/internal/engine/internal/mockable"
	"github.com/ooni/probe-cli/v3/internal/engine/internal/psiphonx"
	"github.com/ooni/psiphon/oopsi/github.com/Psiphon-Labs/psiphon-tunnel-core/ClientLibrary/clientlib"
)

func TestStartWithCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	sess, err := engine.NewSession(engine.SessionConfig{
		AssetsDir:       "../../testdata",
		Logger:          log.Log,
		SoftwareName:    "ooniprobe-engine",
		SoftwareVersion: "0.0.1",
	})
	if err != nil {
		t.Fatal(err)
	}
	tunnel, err := psiphonx.Start(ctx, sess, psiphonx.Config{})
	if !errors.Is(err, context.Canceled) {
		t.Fatal("not the error we expected")
	}
	if tunnel != nil {
		t.Fatal("expected nil tunnel here")
	}
}

func TestStartStop(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	sess, err := engine.NewSession(engine.SessionConfig{
		AssetsDir:       "../../testdata",
		Logger:          log.Log,
		SoftwareName:    "ooniprobe-engine",
		SoftwareVersion: "0.0.1",
	})
	if err != nil {
		t.Fatal(err)
	}
	tunnel, err := psiphonx.Start(context.Background(), sess, psiphonx.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if tunnel.SOCKS5ProxyURL() == nil {
		t.Fatal("expected non nil URL here")
	}
	if tunnel.BootstrapTime() <= 0 {
		t.Fatal("expected positive bootstrap time here")
	}
	tunnel.Stop()
}

func TestNewOrchestraClientFailure(t *testing.T) {
	expected := errors.New("mocked error")
	sess := &mockable.Session{
		MockableOrchestraClientError: expected,
	}
	tunnel, err := psiphonx.Start(context.Background(), sess, psiphonx.Config{})
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
	if tunnel != nil {
		t.Fatal("expected nil tunnel here")
	}
}

func TestFetchPsiphonConfigFailure(t *testing.T) {
	expected := errors.New("mocked error")
	clnt := mockable.ExperimentOrchestraClient{
		MockableFetchPsiphonConfigErr: expected,
	}
	sess := &mockable.Session{
		MockableOrchestraClient: clnt,
	}
	tunnel, err := psiphonx.Start(context.Background(), sess, psiphonx.Config{})
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
	if tunnel != nil {
		t.Fatal("expected nil tunnel here")
	}
}

func TestMakeMkdirAllFailure(t *testing.T) {
	expected := errors.New("mocked error")
	dependencies := FakeDependencies{
		MkdirAllErr: expected,
	}
	clnt := mockable.ExperimentOrchestraClient{
		MockableFetchPsiphonConfigResult: []byte(`{}`),
	}
	sess := &mockable.Session{
		MockableOrchestraClient: clnt,
	}
	tunnel, err := psiphonx.Start(context.Background(), sess, psiphonx.Config{
		Dependencies: dependencies,
	})
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
	if tunnel != nil {
		t.Fatal("expected nil tunnel here")
	}
}

func TestMakeRemoveAllFailure(t *testing.T) {
	expected := errors.New("mocked error")
	dependencies := FakeDependencies{
		RemoveAllErr: expected,
	}
	clnt := mockable.ExperimentOrchestraClient{
		MockableFetchPsiphonConfigResult: []byte(`{}`),
	}
	sess := &mockable.Session{
		MockableOrchestraClient: clnt,
	}
	tunnel, err := psiphonx.Start(context.Background(), sess, psiphonx.Config{
		Dependencies: dependencies,
	})
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
	if tunnel != nil {
		t.Fatal("expected nil tunnel here")
	}
}

func TestMakeStartFailure(t *testing.T) {
	expected := errors.New("mocked error")
	dependencies := FakeDependencies{
		StartErr: expected,
	}
	clnt := mockable.ExperimentOrchestraClient{
		MockableFetchPsiphonConfigResult: []byte(`{}`),
	}
	sess := &mockable.Session{
		MockableOrchestraClient: clnt,
	}
	tunnel, err := psiphonx.Start(context.Background(), sess, psiphonx.Config{
		Dependencies: dependencies,
	})
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
	if tunnel != nil {
		t.Fatal("expected nil tunnel here")
	}
}

func TestNilTunnel(t *testing.T) {
	var tunnel *psiphonx.Tunnel
	if tunnel.BootstrapTime() != 0 {
		t.Fatal("expected zero bootstrap time")
	}
	if tunnel.SOCKS5ProxyURL() != nil {
		t.Fatal("expected nil SOCKS Proxy URL")
	}
	tunnel.Stop() // must not crash
}

type FakeDependencies struct {
	MkdirAllErr  error
	RemoveAllErr error
	StartErr     error
}

func (fd FakeDependencies) MkdirAll(path string, perm os.FileMode) error {
	return fd.MkdirAllErr
}

func (fd FakeDependencies) RemoveAll(path string) error {
	return fd.RemoveAllErr
}

func (fd FakeDependencies) Start(
	ctx context.Context, config []byte, workdir string) (*clientlib.PsiphonTunnel, error) {
	return nil, fd.StartErr
}
