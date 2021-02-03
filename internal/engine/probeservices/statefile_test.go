package probeservices_test

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/engine/kvstore"
	"github.com/ooni/probe-cli/v3/internal/engine/probeservices"
)

func TestStateAuth(t *testing.T) {
	t.Run("with no Token", func(t *testing.T) {
		state := probeservices.State{Expire: time.Now().Add(10 * time.Hour)}
		if state.Auth() != nil {
			t.Fatal("expected nil here")
		}
	})
	t.Run("with expired Token", func(t *testing.T) {
		state := probeservices.State{
			Expire: time.Now().Add(-1 * time.Hour),
			Token:  "xx-x-xxx-xx",
		}
		if state.Auth() != nil {
			t.Fatal("expected nil here")
		}
	})
	t.Run("with good Token", func(t *testing.T) {
		state := probeservices.State{
			Expire: time.Now().Add(10 * time.Hour),
			Token:  "xx-x-xxx-xx",
		}
		if state.Auth() == nil {
			t.Fatal("expected valid auth here")
		}
	})
}

func TestStateCredentials(t *testing.T) {
	t.Run("with no ClientID", func(t *testing.T) {
		state := probeservices.State{}
		if state.Credentials() != nil {
			t.Fatal("expected nil here")
		}
	})
	t.Run("with no Password", func(t *testing.T) {
		state := probeservices.State{
			ClientID: "xx-x-xxx-xx",
		}
		if state.Credentials() != nil {
			t.Fatal("expected nil here")
		}
	})
	t.Run("with all good", func(t *testing.T) {
		state := probeservices.State{
			ClientID: "xx-x-xxx-xx",
			Password: "xx",
		}
		if state.Credentials() == nil {
			t.Fatal("expected valid auth here")
		}
	})
}

func TestStateFileMemoryIntegration(t *testing.T) {
	// Does the StateFile have the property that we can write
	// values into it and then read again the same files?
	sf := probeservices.NewStateFile(kvstore.NewMemoryKeyValueStore())
	s := probeservices.State{
		Expire:   time.Now(),
		Password: "xy",
		Token:    "abc",
		ClientID: "xx",
	}
	if err := sf.Set(s); err != nil {
		t.Fatal(err)
	}
	os := sf.Get()
	diff := cmp.Diff(s, os)
	if diff != "" {
		t.Fatal(diff)
	}
}

func TestStateFileSetMarshalError(t *testing.T) {
	sf := probeservices.NewStateFile(kvstore.NewMemoryKeyValueStore())
	s := probeservices.State{
		Expire:   time.Now(),
		Password: "xy",
		Token:    "abc",
		ClientID: "xx",
	}
	expected := errors.New("mocked error")
	failingfunc := func(v interface{}) ([]byte, error) {
		return nil, expected
	}
	if err := sf.SetMockable(s, failingfunc); !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
}

func TestStateFileGetKVStoreGetError(t *testing.T) {
	sf := probeservices.NewStateFile(kvstore.NewMemoryKeyValueStore())
	expected := errors.New("mocked error")
	failingfunc := func(string) ([]byte, error) {
		return nil, expected
	}
	s, err := sf.GetMockable(failingfunc, json.Unmarshal)
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
	if s.ClientID != "" {
		t.Fatal("unexpected ClientID field")
	}
	if !s.Expire.IsZero() {
		t.Fatal("unexpected Expire field")
	}
	if s.Password != "" {
		t.Fatal("unexpected Password field")
	}
	if s.Token != "" {
		t.Fatal("unexpected Token field")
	}
}

func TestStateFileGetUnmarshalError(t *testing.T) {
	sf := probeservices.NewStateFile(kvstore.NewMemoryKeyValueStore())
	if err := sf.Set(probeservices.State{}); err != nil {
		t.Fatal(err)
	}
	expected := errors.New("mocked error")
	failingfunc := func([]byte, interface{}) error {
		return expected
	}
	s, err := sf.GetMockable(sf.Store.Get, failingfunc)
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
	if s.ClientID != "" {
		t.Fatal("unexpected ClientID field")
	}
	if !s.Expire.IsZero() {
		t.Fatal("unexpected Expire field")
	}
	if s.Password != "" {
		t.Fatal("unexpected Password field")
	}
	if s.Token != "" {
		t.Fatal("unexpected Token field")
	}
}
