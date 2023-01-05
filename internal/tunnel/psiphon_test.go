package tunnel

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/Psiphon-Labs/psiphon-tunnel-core/ClientLibrary/clientlib"
)

func TestPsiphonWithCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately fail
	sess := &MockableSession{}
	tunnel, _, err := psiphonStart(ctx, &Config{
		Session:   sess,
		TunnelDir: "testdata",
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatal("not the error we expected")
	}
	if tunnel != nil {
		t.Fatal("expected nil tunnel here")
	}
}

func TestPsiphonWithEmptyTunnelDir(t *testing.T) {
	ctx := context.Background()
	sess := &MockableSession{}
	tunnel, _, err := psiphonStart(ctx, &Config{
		Session:   sess,
		TunnelDir: "",
	})
	if !errors.Is(err, ErrEmptyTunnelDir) {
		t.Fatal("not the error we expected")
	}
	if tunnel != nil {
		t.Fatal("expected nil tunnel here")
	}
}

func TestPsiphonFetchPsiphonConfigFailure(t *testing.T) {
	expected := errors.New("mocked error")
	sess := &MockableSession{
		Err: expected,
	}
	tunnel, _, err := psiphonStart(context.Background(), &Config{
		Session:   sess,
		TunnelDir: "testdata",
	})
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
	if tunnel != nil {
		t.Fatal("expected nil tunnel here")
	}
}

func TestPsiphonMkdirAllFailure(t *testing.T) {
	expected := errors.New("mocked error")
	sess := &MockableSession{
		Result: []byte(`{}`),
	}
	tunnel, _, err := psiphonStart(context.Background(), &Config{
		Session:   sess,
		TunnelDir: "testdata",
		testMkdirAll: func(path string, perm os.FileMode) error {
			return expected
		},
	})
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
	if tunnel != nil {
		t.Fatal("expected nil tunnel here")
	}
}

func TestPsiphonStartFailure(t *testing.T) {
	expected := errors.New("mocked error")
	sess := &MockableSession{
		Result: []byte(`{}`),
	}
	oldStartPsiphon := mockableStartPsiphon
	defer func() {
		mockableStartPsiphon = oldStartPsiphon
	}()
	mockableStartPsiphon = func(ctx context.Context, config []byte,
		workdir string) (*clientlib.PsiphonTunnel, error) {
		return nil, expected
	}
	tunnel, _, err := psiphonStart(context.Background(), &Config{
		Session:   sess,
		TunnelDir: "testdata",
	})
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
	if tunnel != nil {
		t.Fatal("expected nil tunnel here")
	}
}
