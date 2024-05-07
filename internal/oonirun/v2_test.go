package oonirun

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/httpclientx"
	"github.com/ooni/probe-cli/v3/internal/kvstore"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

func TestOONIRunV2LinkCommonCase(t *testing.T) {
	// make a local server that returns a reasonable descriptor for the example experiment
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
		NoCollector: true, // disable collector so we don't submit
		NoJSON:      true,
		Random:      false,
		ReportFile:  "",
		Session:     newMinimalFakeSession(),
	}

	// create a link runner for the local server URL
	r := NewLinkRunner(config, server.URL)

	// run and verify that we could run without getting errors
	if err := r.Run(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestOONIRunV2LinkCannotUpdateCache(t *testing.T) {
	// make a server that returns a minimal descriptor for the example experiment
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

	// create with a key value store that returns an empty cache and fails to update
	// the cache afterwards such that we can see if we detect such an error
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

	// create new runner for the local server URL
	r := NewLinkRunner(config, server.URL)

	// attempt to run the link
	err := r.Run(ctx)

	// make sure we exactly got the cache updating error
	if !errors.Is(err, expected) {
		t.Fatal("unexpected err", err)
	}
}

func TestOONIRunV2LinkWithoutAcceptChanges(t *testing.T) {
	// make a local server that would return a reasonable descriptor
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

	// create a minimal link configuration
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

	// create a new runner for the local server URL
	r := NewLinkRunner(config, server.URL)

	// attempt to run the link
	err := r.Run(ctx)

	// make sure the error indicates we need to accept changes
	if !errors.Is(err, ErrNeedToAcceptChanges) {
		t.Fatal("unexpected err", err)
	}
}

func TestOONIRunV2LinkNilDescriptor(t *testing.T) {
	// create a local server that returns a literal "null" as the JSON descriptor
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("null"))
	}))

	defer server.Close()
	ctx := context.Background()

	// create a minimal link configuration
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

	// attempt to run the link at the local server
	r := NewLinkRunner(config, server.URL)

	// make sure we correctly handled an invalid "null" descriptor
	if err := r.Run(ctx); !errors.Is(err, httpclientx.ErrIsNil) {
		t.Fatal("unexpected error", err)
	}
}

func TestOONIRunV2LinkEmptyTestName(t *testing.T) {
	// load the count of the number of cases where the test name was empty so we can
	// later on check whether this count has increased due to running this test
	emptyTestNamesPrev := v2CountEmptyNettestNames.Load()

	// create a local server that will respond with a minimal descriptor that
	// actually contains an empty test name, which is what we want to test
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

	// create a minimal link configuration
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

	// construct a link runner relative to the local server URL
	r := NewLinkRunner(config, server.URL)

	// attempt to run and verify there's no error (the code only emits a warning in this case)
	if err := r.Run(ctx); err != nil {
		t.Fatal(err)
	}

	// make sure the loop for running nettests continued where we expected it to do so
	if v2CountEmptyNettestNames.Load() != emptyTestNamesPrev+1 {
		t.Fatal("expected to see 1 more instance of empty nettest names")
	}
}

func TestOONIRunV2LinkConnectionResetByPeer(t *testing.T) {
	// create a local server that will reset the connection immediately.
	// actually contains an empty test name, which is what we want to test
	server := testingx.MustNewHTTPServer(testingx.HTTPHandlerReset())

	defer server.Close()
	ctx := context.Background()

	// create a minimal link configuration
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

	// construct a link runner relative to the local server URL
	r := NewLinkRunner(config, server.URL)

	// attempt to run and verify we got ECONNRESET
	if err := r.Run(ctx); !errors.Is(err, netxlite.ECONNRESET) {
		t.Fatal("unexpected error", err)
	}
}

func TestOONIRunV2LinkNonParseableJSON(t *testing.T) {
	// create a local server that will respond with a non-parseable JSON.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{`))
	}))

	defer server.Close()
	ctx := context.Background()

	// create a minimal link configuration
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

	// construct a link runner relative to the local server URL
	r := NewLinkRunner(config, server.URL)

	// attempt to run and verify there's a JSON parsing error
	if err := r.Run(ctx); err == nil || err.Error() != "unexpected end of JSON input" {
		t.Fatal("unexpected error", err)
	}
}

func TestV2MeasureDescriptor(t *testing.T) {

	t.Run("with nil descriptor", func(t *testing.T) {
		ctx := context.Background()
		config := &LinkConfig{}

		// invoke the function with a nil descriptor and make sure the code
		// is correctly handling this specific case by returnning error
		err := V2MeasureDescriptor(ctx, config, nil)

		if !errors.Is(err, ErrNilDescriptor) {
			t.Fatal("unexpected err", err)
		}
	})

	t.Run("with failing experiment", func(t *testing.T) {
		// load the previous count of failed experiments so we can check that it increased later
		previousFailedExperiments := v2CountFailedExperiments.Load()

		expected := errors.New("mocked error")

		ctx := context.Background()
		sess := newMinimalFakeSession()

		// create a mocked submitter that will panic in case we try to submit, such that
		// this test fails with a panic if we go as far as attempting to submit
		//
		// Note: the convention is that we do not submit experiment results when the
		// experiment measurement function returns a non-nil error, since such an error
		// represents a fundamental failure in setting up the experiment
		sess.MockNewSubmitter = func(ctx context.Context) (model.Submitter, error) {
			subm := &mocks.Submitter{
				MockSubmit: func(ctx context.Context, m *model.Measurement) error {
					panic("should not be called")
				},
			}
			return subm, nil
		}

		// mock an experiment builder where we have the measurement function fail by returning
		// an error, which has the meaning indicated in the previous comment
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

		// create a mostly empty config referring to the session
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

		// create a mostly empty descriptor referring to the example experiment
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

		// attempt to measure this descriptor
		err := V2MeasureDescriptor(ctx, config, descr)

		// here we do not expect to see an error because the implementation continues
		// until it has run all experiments and just emits warning messages
		if err != nil {
			t.Fatal(err)
		}

		// however there's also a count of the number of times we failed to load
		// an experiment and we use that to make sure the code failed where we expected
		if v2CountFailedExperiments.Load() != previousFailedExperiments+1 {
			t.Fatal("expected to see a failed experiment")
		}
	})
}

func TestV2MeasureHTTPS(t *testing.T) {

	t.Run("when we cannot load from cache", func(t *testing.T) {
		expected := errors.New("mocked error")
		ctx := context.Background()

		// construct the link configuration with a key-value store that fails
		// with a well-know error when attempting to load.
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

		// attempt to measure with the given config (there's no need to pass an URL
		// here because we should fail to load from the cache first)
		err := v2MeasureHTTPS(ctx, config, "")

		// verify that we've actually got the expected error
		if !errors.Is(err, expected) {
			t.Fatal("unexpected err", err)
		}
	})

	t.Run("when we cannot pull changes", func(t *testing.T) {
		// create and immediately cancel a context so that HTTP would fail
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

		// attempt to measure with a random URL (which is fine since we shouldn't use it)
		err := v2MeasureHTTPS(ctx, config, "https://example.com")

		// make sure that we've actually go the expected error
		if !errors.Is(err, context.Canceled) {
			t.Fatal("unexpected err", err)
		}
	})

}

func TestV2DescriptorCacheLoad(t *testing.T) {

	t.Run("handle the case where we cannot unmarshal the cache content", func(t *testing.T) {
		// write an invalid serialized JSON into the cache
		fsstore := &kvstore.Memory{}
		if err := fsstore.Set(v2DescriptorCacheKey, []byte("{")); err != nil {
			t.Fatal(err)
		}

		// attempt to load descriptors
		cache, err := v2DescriptorCacheLoad(fsstore)

		// make sure we cannot unmarshal
		if err == nil || err.Error() != "unexpected end of JSON input" {
			t.Fatal("unexpected err", err)
		}

		// make sure the returned cache is nil
		if cache != nil {
			t.Fatal("expected nil cache")
		}
	})

}
