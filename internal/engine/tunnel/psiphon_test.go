package tunnel

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/engine/internal/mockable"
	"github.com/ooni/psiphon/oopsi/github.com/Psiphon-Labs/psiphon-tunnel-core/ClientLibrary/clientlib"
)

func TestPsiphonFetchPsiphonConfigFailure(t *testing.T) {
	expected := errors.New("mocked error")
	sess := &mockable.Session{
		MockableFetchPsiphonConfigErr: expected,
	}
	tunnel, err := psiphonStart(context.Background(), sess, psiphonConfig{})
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
	if tunnel != nil {
		t.Fatal("expected nil tunnel here")
	}
}

func TestPsiphonMakeMkdirAllFailure(t *testing.T) {
	expected := errors.New("mocked error")
	dependencies := psiphonFakeDependencies{
		MkdirAllErr: expected,
	}
	sess := &mockable.Session{
		MockableFetchPsiphonConfigResult: []byte(`{}`),
	}
	tunnel, err := psiphonStart(context.Background(), sess, psiphonConfig{
		Dependencies: dependencies,
	})
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
	if tunnel != nil {
		t.Fatal("expected nil tunnel here")
	}
}

func TestPsiphonMakeRemoveAllFailure(t *testing.T) {
	expected := errors.New("mocked error")
	dependencies := psiphonFakeDependencies{
		RemoveAllErr: expected,
	}
	sess := &mockable.Session{
		MockableFetchPsiphonConfigResult: []byte(`{}`),
	}
	tunnel, err := psiphonStart(context.Background(), sess, psiphonConfig{
		Dependencies: dependencies,
	})
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
	if tunnel != nil {
		t.Fatal("expected nil tunnel here")
	}
}

func TestPsiphonMakeStartFailure(t *testing.T) {
	expected := errors.New("mocked error")
	dependencies := psiphonFakeDependencies{
		StartErr: expected,
	}
	sess := &mockable.Session{
		MockableFetchPsiphonConfigResult: []byte(`{}`),
	}
	tunnel, err := psiphonStart(context.Background(), sess, psiphonConfig{
		Dependencies: dependencies,
	})
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
	if tunnel != nil {
		t.Fatal("expected nil tunnel here")
	}
}

func TestPsiphonNilTunnel(t *testing.T) {
	var tunnel *psiphonTunnel
	if tunnel.BootstrapTime() != 0 {
		t.Fatal("expected zero bootstrap time")
	}
	if tunnel.SOCKS5ProxyURL() != nil {
		t.Fatal("expected nil SOCKS Proxy URL")
	}
	tunnel.Stop() // must not crash
}

type psiphonFakeDependencies struct {
	MkdirAllErr  error
	RemoveAllErr error
	StartErr     error
}

func (fd psiphonFakeDependencies) MkdirAll(path string, perm os.FileMode) error {
	return fd.MkdirAllErr
}

func (fd psiphonFakeDependencies) RemoveAll(path string) error {
	return fd.RemoveAllErr
}

func (fd psiphonFakeDependencies) Start(
	ctx context.Context, config []byte, workdir string) (*clientlib.PsiphonTunnel, error) {
	return nil, fd.StartErr
}
