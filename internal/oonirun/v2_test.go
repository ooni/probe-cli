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
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func TestOONIRunV2LinkCommonCase(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		descriptor := &V2Descriptor{
			Name:        "",
			Description: "",
			Author:      "",
			Nettests: []V2Nettest{{
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
		Session:     newMinimalFakeSession(),
	}
	r := NewLinkRunner(config, server.URL)
	if err := r.Run(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestOONIRunV2LinkCannotUpdateCache(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		descriptor := &V2Descriptor{
			Name:        "",
			Description: "",
			Author:      "",
			Nettests: []V2Nettest{{
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
		Session:     newMinimalFakeSession(),
	}
	r := NewLinkRunner(config, server.URL)
	err := r.Run(ctx)
	if !errors.Is(err, expected) {
		t.Fatal("unexpected err", err)
	}
}

func TestOONIRunV2LinkWithoutAcceptChanges(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		descriptor := &V2Descriptor{
			Name:        "",
			Description: "",
			Author:      "",
			Nettests: []V2Nettest{{
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
		Session:     newMinimalFakeSession(),
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
		Session:     newMinimalFakeSession(),
	}
	r := NewLinkRunner(config, server.URL)
	if err := r.Run(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestOONIRunV2LinkEmptyTestName(t *testing.T) {
	emptyTestNamesPrev := v2CountEmptyNettestNames.Load()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		descriptor := &V2Descriptor{
			Name:        "",
			Description: "",
			Author:      "",
			Nettests: []V2Nettest{{
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
		Session:     newMinimalFakeSession(),
	}
	r := NewLinkRunner(config, server.URL)
	if err := r.Run(ctx); err != nil {
		t.Fatal(err)
	}
	if v2CountEmptyNettestNames.Load() != emptyTestNamesPrev+1 {
		t.Fatal("expected to see 1 more instance of empty nettest names")
	}
}

func TestV2MeasureDescriptor(t *testing.T) {
	t.Run("with nil descriptor", func(t *testing.T) {
		ctx := context.Background()
		config := &LinkConfig{}
		err := V2MeasureDescriptor(ctx, config, nil)
		if !errors.Is(err, ErrNilDescriptor) {
			t.Fatal("unexpected err", err)
		}
	})

	t.Run("with failing experiment", func(t *testing.T) {
		previousFailedExperiments := v2CountFailedExperiments.Load()
		expected := errors.New("mocked error")
		ctx := context.Background()
		sess := newMinimalFakeSession()
		sess.MockNewSubmitter = func(ctx context.Context) (model.Submitter, error) {
			subm := &mocks.Submitter{
				MockSubmit: func(ctx context.Context, m *model.Measurement) error {
					panic("should not be called")
				},
			}
			return subm, nil
		}
		sess.MockNewExperimentBuilder = func(name string) (model.ExperimentBuilder, error) {
			eb := &mocks.ExperimentBuilder{
				MockInputPolicy: func() model.InputPolicy {
					return model.InputNone
				},
				MockSetOptionsAny: func(options map[string]any) error {
					return nil
				},
				MockNewExperiment: func() model.Experiment {
					exp := &mocks.Experiment{
						MockMeasureAsync: func(ctx context.Context, input string) (<-chan *model.Measurement, error) {
							return nil, expected
						},
						MockKibiBytesReceived: func() float64 {
							return 1.1
						},
						MockKibiBytesSent: func() float64 {
							return 0.1
						},
					}
					return exp
				},
			}
			return eb, nil
		}
		config := &LinkConfig{
			AcceptChanges: false,
			Annotations:   map[string]string{},
			KVStore:       nil,
			MaxRuntime:    0,
			NoCollector:   false,
			NoJSON:        false,
			Random:        false,
			ReportFile:    "",
			Session:       sess,
		}
		descr := &V2Descriptor{
			Name:        "",
			Description: "",
			Author:      "",
			Nettests: []V2Nettest{{
				Inputs:   []string{},
				Options:  map[string]any{},
				TestName: "example",
			}},
		}
		err := V2MeasureDescriptor(ctx, config, descr)
		if err != nil {
			t.Fatal(err)
		}
		if v2CountFailedExperiments.Load() != previousFailedExperiments+1 {
			t.Fatal("expected to see a failed experiment")
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
			Session:     newMinimalFakeSession(),
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
			Session:       newMinimalFakeSession(),
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
