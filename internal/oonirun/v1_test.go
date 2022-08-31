package oonirun

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/kvstore"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

func newMinimalFakeSession() *mocks.Session {
	return &mocks.Session{
		MockLogger: func() model.Logger {
			return model.DiscardLogger
		},
		MockNewExperimentBuilder: func(name string) (model.ExperimentBuilder, error) {
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
							out := make(chan *model.Measurement)
							go func() {
								defer close(out)
								ff := &testingx.FakeFiller{}
								var meas model.Measurement
								ff.Fill(&meas)
								out <- &meas
							}()
							return out, nil
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
		},
		MockDefaultHTTPClient: func() model.HTTPClient {
			return http.DefaultClient
		},
	}
}

func TestOONIRunV1Link(t *testing.T) {
	ctx := context.Background()
	config := &LinkConfig{
		AcceptChanges: false,
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
	r := NewLinkRunner(config, "https://run.ooni.io/nettest?tn=example&mv=1.2.0")
	if err := r.Run(ctx); err != nil {
		t.Fatal(err)
	}
	r = NewLinkRunner(config, "ooni://nettest?tn=example&mv=1.2.0")
	if err := r.Run(ctx); err != nil {
		t.Fatal(err)
	}
}

func TestV1MeasureInvalidURL(t *testing.T) {
	t.Run("URL does not parse", func(t *testing.T) {
		ctx := context.Background()
		config := &LinkConfig{
			Session: &mocks.Session{
				MockLogger: func() model.Logger {
					return model.DiscardLogger
				},
			},
		}
		URL := "\t"
		err := v1Measure(ctx, config, URL)
		if err == nil || !strings.HasSuffix(err.Error(), "invalid control character in URL") {
			t.Fatal("unexpected err", err)
		}
	})

	t.Run("with https:// URL and invalid hostname", func(t *testing.T) {
		ctx := context.Background()
		config := &LinkConfig{
			Session: &mocks.Session{
				MockLogger: func() model.Logger {
					return model.DiscardLogger
				},
			},
		}
		URL := "https://run.ooni.nu/nettest"
		err := v1Measure(ctx, config, URL)
		if !errors.Is(err, ErrInvalidV1URLHost) {
			t.Fatal("unexpected err", err)
		}
	})

	t.Run("with https:// URL and invalid path", func(t *testing.T) {
		ctx := context.Background()
		config := &LinkConfig{
			Session: &mocks.Session{
				MockLogger: func() model.Logger {
					return model.DiscardLogger
				},
			},
		}
		URL := "https://run.ooni.io/antani"
		err := v1Measure(ctx, config, URL)
		if !errors.Is(err, ErrInvalidV1URLPath) {
			t.Fatal("unexpected err", err)
		}
	})

	t.Run("with ooni:// URL and invalid host", func(t *testing.T) {
		ctx := context.Background()
		config := &LinkConfig{
			Session: &mocks.Session{
				MockLogger: func() model.Logger {
					return model.DiscardLogger
				},
			},
		}
		URL := "ooni://antani"
		err := v1Measure(ctx, config, URL)
		if !errors.Is(err, ErrInvalidV1URLHost) {
			t.Fatal("unexpected err", err)
		}
	})

	t.Run("with ooni:// URL and path", func(t *testing.T) {
		ctx := context.Background()
		config := &LinkConfig{
			Session: &mocks.Session{
				MockLogger: func() model.Logger {
					return model.DiscardLogger
				},
			},
		}
		URL := "ooni://nettest/x"
		err := v1Measure(ctx, config, URL)
		if !errors.Is(err, ErrInvalidV1URLPath) {
			t.Fatal("unexpected err", err)
		}
	})

	t.Run("with invalid URL scheme", func(t *testing.T) {
		ctx := context.Background()
		config := &LinkConfig{
			Session: &mocks.Session{
				MockLogger: func() model.Logger {
					return model.DiscardLogger
				},
			},
		}
		URL := "antani://nettest"
		err := v1Measure(ctx, config, URL)
		if !errors.Is(err, ErrInvalidV1URLScheme) {
			t.Fatal("unexpected err", err)
		}
	})

	t.Run("with empty test name", func(t *testing.T) {
		ctx := context.Background()
		config := &LinkConfig{
			Session: &mocks.Session{
				MockLogger: func() model.Logger {
					return model.DiscardLogger
				},
			},
		}
		URL := "ooni://nettest/"
		err := v1Measure(ctx, config, URL)
		if !errors.Is(err, ErrInvalidV1URLQueryArgument) {
			t.Fatal("unexpected err", err)
		}
	})

	t.Run("with invalid JSON and explicit / as path", func(t *testing.T) {
		ctx := context.Background()
		config := &LinkConfig{
			Session: &mocks.Session{
				MockLogger: func() model.Logger {
					return model.DiscardLogger
				},
			},
		}
		URL := "ooni://nettest/?tn=web_connectivity&ta=123x"
		err := v1Measure(ctx, config, URL)
		if !errors.Is(err, ErrInvalidV1URLQueryArgument) {
			t.Fatal("unexpected err", err)
		}
	})

	t.Run("with invalid JSON and empty path", func(t *testing.T) {
		ctx := context.Background()
		config := &LinkConfig{
			Session: &mocks.Session{
				MockLogger: func() model.Logger {
					return model.DiscardLogger
				},
			},
		}
		URL := "ooni://nettest?tn=web_connectivity&ta=123x"
		err := v1Measure(ctx, config, URL)
		if !errors.Is(err, ErrInvalidV1URLQueryArgument) {
			t.Fatal("unexpected err", err)
		}
	})

	t.Run("with missing minimum version", func(t *testing.T) {
		ctx := context.Background()
		config := &LinkConfig{
			Session: &mocks.Session{
				MockLogger: func() model.Logger {
					return model.DiscardLogger
				},
			},
		}
		URL := "ooni://nettest?tn=example"
		err := v1Measure(ctx, config, URL)
		if !errors.Is(err, ErrInvalidV1URLQueryArgument) {
			t.Fatal("unexpected err", err)
		}
	})
}

func TestV1ParseArguments(t *testing.T) {
	t.Run("with invalid test arguments", func(t *testing.T) {
		// "[QueryUnescape] returns an error if any % is not followed by two hexadecimal digits."
		out, err := v1ParseArguments("%KK")
		if !errors.Is(err, ErrInvalidV1URLQueryArgument) {
			t.Fatal("unexpected err", err)
		}
		if len(out) > 0 {
			t.Fatal("expected no output")
		}
	})

	t.Run("with valid arguments", func(t *testing.T) {
		out, err := v1ParseArguments("%7B%22urls%22%3A%5B%22https%3A%2F%2Fexample.com%2F%22%5D%7D")
		if err != nil {
			t.Fatal(err)
		}
		if len(out) != 1 || out[0] != "https://example.com/" {
			t.Fatal("unexpected out", out)
		}
	})
}
