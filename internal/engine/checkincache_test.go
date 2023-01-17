package engine

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/kvstore"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
)

func TestSessionUpdateCheckInFlagsState(t *testing.T) {
	t.Run("when we can successfully store", func(t *testing.T) {
		memstore := &kvstore.Memory{}
		s := &Session{
			kvStore: memstore,
		}
		expectmap := map[string]bool{
			"foobar": true,
		}
		result := &model.OOAPICheckInResult{
			Conf: model.OOAPICheckInResultConfig{
				Features: expectmap,
			},
		}
		err := s.updateCheckInFlagsState(result)
		if err != nil {
			t.Fatal(err)
		}
		var wrapper checkInFlagsWrapper
		data, err := memstore.Get(checkInFlagsState)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(data, &wrapper); err != nil {
			t.Fatal(err)
		}
		if diff := cmp.Diff(expectmap, wrapper.Flags); diff != "" {
			t.Fatal(diff)
		}
		if wrapper.Expire.Before(time.Now().Add(23 * time.Hour)) {
			t.Fatal("unexpected expire value")
		}
	})

	t.Run("when we there's a failure trying to store", func(t *testing.T) {
		expected := errors.New("mocked error")
		s := &Session{
			kvStore: &mocks.KeyValueStore{
				MockSet: func(key string, value []byte) (err error) {
					return expected
				},
			},
		}
		err := s.updateCheckInFlagsState(&model.OOAPICheckInResult{})
		if !errors.Is(err, expected) {
			t.Fatal(err)
		}
	})
}

func TestSessionGetCheckInFlagValue(t *testing.T) {
	t.Run("when we cannot get from the store", func(t *testing.T) {
		expected := errors.New("mocked error")
		s := &Session{
			kvStore: &mocks.KeyValueStore{
				MockGet: func(key string) (value []byte, err error) {
					return nil, expected
				},
			},
		}
		if s.getCheckInFlagValue("antani") {
			t.Fatal("expected to see false here")
		}
	})

	t.Run("when we cannot unmarshal", func(t *testing.T) {
		s := &Session{
			kvStore: &mocks.KeyValueStore{
				MockGet: func(key string) (value []byte, err error) {
					return []byte(`{`), nil
				},
			},
		}
		if s.getCheckInFlagValue("antani") {
			t.Fatal("expected to see false here")
		}
	})

	t.Run("if the record was cached too much time ago", func(t *testing.T) {
		response := `{}`
		s := &Session{
			kvStore: &mocks.KeyValueStore{
				MockGet: func(key string) (value []byte, err error) {
					return []byte(response), nil
				},
			},
		}
		if s.getCheckInFlagValue("antani") {
			t.Fatal("expected to see false here")
		}
	})

	t.Run("in case of success", func(t *testing.T) {
		response := &checkInFlagsWrapper{
			Expire: time.Now().Add(time.Hour),
			Flags: map[string]bool{
				"antani": true,
			},
		}
		data, err := json.Marshal(response)
		if err != nil {
			t.Fatal(err)
		}
		s := &Session{
			kvStore: &mocks.KeyValueStore{
				MockGet: func(key string) (value []byte, err error) {
					return data, nil
				},
			},
		}
		if !s.getCheckInFlagValue("antani") {
			t.Fatal("expected to see true here")
		}
	})

	t.Run("in case of success with a nil map", func(t *testing.T) {
		response := &checkInFlagsWrapper{
			Expire: time.Now().Add(time.Hour),
			Flags:  nil,
		}
		data, err := json.Marshal(response)
		if err != nil {
			t.Fatal(err)
		}
		s := &Session{
			kvStore: &mocks.KeyValueStore{
				MockGet: func(key string) (value []byte, err error) {
					return data, nil
				},
			},
		}
		if s.getCheckInFlagValue("antani") {
			t.Fatal("expected to see false here")
		}
	})
}
