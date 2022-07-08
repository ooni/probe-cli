package oonirun

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/kvstore"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// TODO(bassosimone): it would be cool to write unit tests. However, to do that
// we need to ~redesign the engine package for unit-testability.

func TestOONIRunV2LinkCommonCase(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		descriptor := &v2Descriptor{
			Name:        "",
			Description: "",
			Author:      "",
			Nettests: []v2Nettest{{
				Inputs: []string{},
				Options: map[string]any{
					"SleepTime": int64(10 * time.Millisecond),
				},
				TestName: "example",
			}},
		}
		data, err := json.Marshal(descriptor)
		runtimex.PanicOnError(err, "json.Marshal failed")
		w.Write(data)
	}))
	defer server.Close()
	ctx := context.Background()
	config := &LinkConfig{
		AcceptChanges: true, // avoid "oonirun: need to accept changes" error
		Annotations: map[string]string{
			"platform": "linux",
		},
		KVStore:     &kvstore.Memory{},
		MaxRuntime:  0,
		NoCollector: true,
		NoJSON:      true,
		Random:      false,
		ReportFile:  "",
		Session:     newSession(ctx, t),
	}
	r := NewLinkRunner(config, server.URL)
	if err := r.Run(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestOONIRunV2LinkCannotUpdateCache(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		descriptor := &v2Descriptor{
			Name:        "",
			Description: "",
			Author:      "",
			Nettests: []v2Nettest{{
				Inputs: []string{},
				Options: map[string]any{
					"SleepTime": int64(10 * time.Millisecond),
				},
				TestName: "example",
			}},
		}
		data, err := json.Marshal(descriptor)
		runtimex.PanicOnError(err, "json.Marshal failed")
		w.Write(data)
	}))
	defer server.Close()
	ctx := context.Background()
	expected := errors.New("mocked")
	config := &LinkConfig{
		AcceptChanges: true, // avoid "oonirun: need to accept changes" error
		Annotations: map[string]string{
			"platform": "linux",
		},
		KVStore: &mocks.KeyValueStore{
			MockGet: func(key string) ([]byte, error) {
				return []byte("{}"), nil
			},
			MockSet: func(key string, value []byte) error {
				return expected
			},
		},
		MaxRuntime:  0,
		NoCollector: true,
		NoJSON:      true,
		Random:      false,
		ReportFile:  "",
		Session:     newSession(ctx, t),
	}
	r := NewLinkRunner(config, server.URL)
	err := r.Run(ctx)
	if !errors.Is(err, expected) {
		t.Fatal("unexpected err", err)
	}
}

func TestOONIRunV2LinkWithoutAcceptChanges(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		descriptor := &v2Descriptor{
			Name:        "",
			Description: "",
			Author:      "",
			Nettests: []v2Nettest{{
				Inputs: []string{},
				Options: map[string]any{
					"SleepTime": int64(10 * time.Millisecond),
				},
				TestName: "example",
			}},
		}
		data, err := json.Marshal(descriptor)
		runtimex.PanicOnError(err, "json.Marshal failed")
		w.Write(data)
	}))
	defer server.Close()
	ctx := context.Background()
	config := &LinkConfig{
		AcceptChanges: false, // should see "oonirun: need to accept changes" error
		Annotations: map[string]string{
			"platform": "linux",
		},
		KVStore:     &kvstore.Memory{},
		MaxRuntime:  0,
		NoCollector: true,
		NoJSON:      true,
		Random:      false,
		ReportFile:  "",
		Session:     newSession(ctx, t),
	}
	r := NewLinkRunner(config, server.URL)
	err := r.Run(ctx)
	if !errors.Is(err, ErrNeedToAcceptChanges) {
		t.Fatal("unexpected err", err)
	}
}

func TestOONIRunV2LinkNilDescriptor(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("null"))
	}))
	defer server.Close()
	ctx := context.Background()
	config := &LinkConfig{
		AcceptChanges: true, // avoid "oonirun: need to accept changes" error
		Annotations: map[string]string{
			"platform": "linux",
		},
		KVStore:     &kvstore.Memory{},
		MaxRuntime:  0,
		NoCollector: true,
		NoJSON:      true,
		Random:      false,
		ReportFile:  "",
		Session:     newSession(ctx, t),
	}
	r := NewLinkRunner(config, server.URL)
	if err := r.Run(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestOONIRunV2LinkEmptyTestName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		descriptor := &v2Descriptor{
			Name:        "",
			Description: "",
			Author:      "",
			Nettests: []v2Nettest{{
				Inputs: []string{},
				Options: map[string]any{
					"SleepTime": int64(10 * time.Millisecond),
				},
				TestName: "", // empty!
			}},
		}
		data, err := json.Marshal(descriptor)
		runtimex.PanicOnError(err, "json.Marshal failed")
		w.Write(data)
	}))
	defer server.Close()
	ctx := context.Background()
	config := &LinkConfig{
		AcceptChanges: true, // avoid "oonirun: need to accept changes" error
		Annotations: map[string]string{
			"platform": "linux",
		},
		KVStore:     &kvstore.Memory{},
		MaxRuntime:  0,
		NoCollector: true,
		NoJSON:      true,
		Random:      false,
		ReportFile:  "",
		Session:     newSession(ctx, t),
	}
	r := NewLinkRunner(config, server.URL)
	if err := r.Run(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestV2MeasureDescriptor(t *testing.T) {
	t.Run("with nil descriptor", func(t *testing.T) {
		ctx := context.Background()
		config := &LinkConfig{}
		err := v2MeasureDescriptor(ctx, config, nil)
		if !errors.Is(err, ErrNilDescriptor) {
			t.Fatal("unexpected err", err)
		}
	})
}

func TestV2MeasureHTTPS(t *testing.T) {
	t.Run("when we cannot load from cache", func(t *testing.T) {
		expected := errors.New("mocked error")
		ctx := context.Background()
		config := &LinkConfig{
			AcceptChanges: false,
			Annotations:   map[string]string{},
			KVStore: &mocks.KeyValueStore{
				MockGet: func(key string) (value []byte, err error) {
					return nil, expected
				},
			},
			MaxRuntime:  0,
			NoCollector: false,
			NoJSON:      false,
			Random:      false,
			ReportFile:  "",
			Session:     newSession(ctx, t),
		}
		err := v2MeasureHTTPS(ctx, config, "")
		if !errors.Is(err, expected) {
			t.Fatal("unexpected err", err)
		}
	})

	t.Run("when we cannot pull changes", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // fail immediately
		config := &LinkConfig{
			AcceptChanges: false,
			Annotations:   map[string]string{},
			KVStore:       &kvstore.Memory{},
			MaxRuntime:    0,
			NoCollector:   false,
			NoJSON:        false,
			Random:        false,
			ReportFile:    "",
			Session:       newSession(ctx, t),
		}
		err := v2MeasureHTTPS(ctx, config, "https://example.com") // should not use URL
		if !errors.Is(err, context.Canceled) {
			t.Fatal("unexpected err", err)
		}
	})
}

func TestV2DescriptorCacheLoad(t *testing.T) {
	t.Run("cannot unmarshal cache content", func(t *testing.T) {
		fsstore := &kvstore.Memory{}
		if err := fsstore.Set(v2DescriptorCacheKey, []byte("{")); err != nil {
			t.Fatal(err)
		}
		cache, err := v2DescriptorCacheLoad(fsstore)
		if err == nil || err.Error() != "unexpected end of JSON input" {
			t.Fatal("unexpected err", err)
		}
		if cache != nil {
			t.Fatal("expected nil cache")
		}
	})
}
