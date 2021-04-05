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
	tunnel, err := psiphonStart(context.Background(), &Config{
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
	sess := &mockable.Session{
		MockableFetchPsiphonConfigResult: []byte(`{}`),
	}
	tunnel, err := psiphonStart(context.Background(), &Config{
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
	sess := &mockable.Session{
		MockableFetchPsiphonConfigResult: []byte(`{}`),
	}
	tunnel, err := psiphonStart(context.Background(), &Config{
		Session:   sess,
		TunnelDir: "testdata",
		testStartPsiphon: func(ctx context.Context, config []byte,
			workdir string) (*clientlib.PsiphonTunnel, error) {
			return nil, expected
		},
	})
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
	if tunnel != nil {
		t.Fatal("expected nil tunnel here")
	}
}
