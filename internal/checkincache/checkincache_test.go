package checkincache

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/kvstore"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestStore(t *testing.T) {
	t.Run("when we can successfully store", func(t *testing.T) {
		memstore := &kvstore.Memory{}
		expectmap := map[string]bool{
			"foobar": true,
		}
		result := &model.OOAPICheckInResult{
			Conf: model.OOAPICheckInResultConfig{
				Features: expectmap,
			},
		}
		err := Store(memstore, result)
		if err != nil {
			t.Fatal(err)
		}
		var wrapper FeatureFlagsWrapper
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

	t.Run("when there's a failure trying to store", func(t *testing.T) {
		expected := errors.New("mocked error")
		memstore := &mocks.KeyValueStore{
			MockSet: func(key string, value []byte) (err error) {
				return expected
			},
		}
		err := Store(memstore, &model.OOAPICheckInResult{})
		if !errors.Is(err, expected) {
			t.Fatal(err)
		}
	})
}

func TestGetFeatureFlag(t *testing.T) {
	t.Run("when we cannot get from the store", func(t *testing.T) {
		expectedErr := errors.New("mocked error")
		memstore := &mocks.KeyValueStore{
			MockGet: func(key string) (value []byte, err error) {
				return nil, expectedErr
			},
		}
		if GetFeatureFlag(memstore, "antani") {
			t.Fatal("expected to see false here")
		}
	})

	t.Run("when we cannot unmarshal", func(t *testing.T) {
		memstore := &mocks.KeyValueStore{
			MockGet: func(key string) (value []byte, err error) {
				return []byte(`{`), nil
			},
		}
		if GetFeatureFlag(memstore, "antani") {
			t.Fatal("expected to see false here")
		}
	})

	t.Run("if the record was cached too much time ago", func(t *testing.T) {
		response := `{}` // zero struct means the expiry time is long ago in the past
		memstore := &mocks.KeyValueStore{
			MockGet: func(key string) (value []byte, err error) {
				return []byte(response), nil
			},
		}
		if GetFeatureFlag(memstore, "antani") {
			t.Fatal("expected to see false here")
		}
	})

	t.Run("in case of success", func(t *testing.T) {
		response := &FeatureFlagsWrapper{
			Expire: time.Now().Add(time.Hour),
			Flags: map[string]bool{
				"antani": true,
			},
		}
		data, err := json.Marshal(response)
		if err != nil {
			t.Fatal(err)
		}
		memstore := &mocks.KeyValueStore{
			MockGet: func(key string) (value []byte, err error) {
				return data, nil
			},
		}
		if !GetFeatureFlag(memstore, "antani") {
			t.Fatal("expected to see true here")
		}
	})

	t.Run("in case of success with a nil map", func(t *testing.T) {
		response := &FeatureFlagsWrapper{
			Expire: time.Now().Add(time.Hour),
			Flags:  nil, // here the map is explicitly nil
		}
		data, err := json.Marshal(response)
		if err != nil {
			t.Fatal(err)
		}
		memstore := &mocks.KeyValueStore{
			MockGet: func(key string) (value []byte, err error) {
				return data, nil
			},
		}
		if GetFeatureFlag(memstore, "antani") {
			t.Fatal("expected to see false here")
		}
	})
}
