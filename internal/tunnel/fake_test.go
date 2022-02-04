package tunnel

import (
	"context"
	"errors"
	"net"
	"os"
	"testing"

	"github.com/armon/go-socks5"
)

func TestFakeWithCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately fail
	sess := &MockableSession{}
	tunnel, _, err := fakeStart(ctx, &Config{
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

func TestFakeWithEmptyTunnelDir(t *testing.T) {
	ctx := context.Background()
	sess := &MockableSession{}
	tunnel, _, err := fakeStart(ctx, &Config{
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

func TestFakeWithFailingMkdirAll(t *testing.T) {
	expected := errors.New("mocked error")
	ctx := context.Background()
	sess := &MockableSession{}
	tunnel, _, err := fakeStart(ctx, &Config{
		Session:   sess,
		TunnelDir: "testdata",
		testMkdirAll: func(dir string, mode os.FileMode) error {
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

func TestFakeSocks5NewFails(t *testing.T) {
	expected := errors.New("mocked error")
	ctx := context.Background()
	sess := &MockableSession{}
	tunnel, _, err := fakeStart(ctx, &Config{
		Session:   sess,
		TunnelDir: "testdata",
		testSocks5New: func(conf *socks5.Config) (*socks5.Server, error) {
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

func TestFakeNetListenFails(t *testing.T) {
	expected := errors.New("mocked error")
	ctx := context.Background()
	sess := &MockableSession{}
	tunnel, _, err := fakeStart(ctx, &Config{
		Session:   sess,
		TunnelDir: "testdata",
		testNetListen: func(network, address string) (net.Listener, error) {
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
