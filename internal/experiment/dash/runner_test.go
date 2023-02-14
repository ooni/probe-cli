package dash

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/legacy/mockable"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/tracex"
)

func TestRunnerLoopLocateFailure(t *testing.T) {
	expected := errors.New("mocked error")

	r := runner{
		callbacks: model.NewPrinterCallbacks(log.Log),
		httpClient: &mocks.HTTPClient{
			MockDo: func(req *http.Request) (*http.Response, error) {
				switch {
				case req.URL.Hostname() == "locate.measurementlab.net":
					return nil, expected
				default:
					return nil, errors.New("unexpected HTTP request")
				}
			},
		},
		saver: &tracex.Saver{},
		sess: &mockable.Session{
			MockableLogger: log.Log,
		},
		tk: &TestKeys{},
	}

	err := r.loop(context.Background(), 1)
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
}

func TestRunnerLoopNegotiateFailure(t *testing.T) {
	expected := errors.New("mocked error")

	r := runner{
		callbacks: model.NewPrinterCallbacks(log.Log),

		httpClient: &mocks.HTTPClient{
			MockDo: func(req *http.Request) (*http.Response, error) {
				switch {
				case req.URL.Hostname() == "locate.measurementlab.net":
					resp := &http.Response{
						StatusCode: 200,
						Body: io.NopCloser(strings.NewReader(
							`{"fqdn": "ams01.measurementlab.net"}`,
						)),
					}
					return resp, nil
				case req.URL.Path == negotiatePath:
					return nil, expected
				default:
					return nil, errors.New("unexpected HTTP request")
				}
			},
		},

		saver: &tracex.Saver{},
		sess: &mockable.Session{
			MockableLogger: log.Log,
		},
		tk: &TestKeys{},
	}

	err := r.loop(context.Background(), 1)
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
}

func TestRunnerLoopMeasureFailure(t *testing.T) {
	expected := errors.New("mocked error")
	r := runner{
		callbacks: model.NewPrinterCallbacks(log.Log),

		httpClient: &mocks.HTTPClient{
			MockDo: func(req *http.Request) (*http.Response, error) {
				switch {
				case req.URL.Hostname() == "locate.measurementlab.net":
					resp := &http.Response{
						StatusCode: 200,
						Body: io.NopCloser(strings.NewReader(
							`{"fqdn": "ams01.measurementlab.net"}`,
						)),
					}
					return resp, nil
				case req.URL.Path == negotiatePath:
					resp := &http.Response{
						StatusCode: 200,
						Body: io.NopCloser(strings.NewReader(
							`{"authorization": "xx", "unchoked": 1}`,
						)),
					}
					return resp, nil
				case strings.HasPrefix(req.URL.Path, downloadPath):
					return nil, expected
				default:
					return nil, errors.New("unexpected HTTP request")
				}
			},
		},

		saver: &tracex.Saver{},
		sess: &mockable.Session{
			MockableLogger: log.Log,
		},
		tk: &TestKeys{},
	}
	err := r.loop(context.Background(), 1)
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
}

func TestRunnerLoopCollectFailure(t *testing.T) {
	expected := errors.New("mocked error")
	saver := new(tracex.Saver)
	saver.Write(&tracex.EventConnectOperation{V: &tracex.EventValue{Duration: 150 * time.Millisecond}})
	r := runner{
		callbacks: model.NewPrinterCallbacks(log.Log),

		httpClient: &mocks.HTTPClient{
			MockDo: func(req *http.Request) (*http.Response, error) {
				switch {
				case req.URL.Hostname() == "locate.measurementlab.net":
					resp := &http.Response{
						StatusCode: 200,
						Body: io.NopCloser(strings.NewReader(
							`{"fqdn": "ams01.measurementlab.net"}`,
						)),
					}
					return resp, nil
				case req.URL.Path == negotiatePath:
					resp := &http.Response{
						StatusCode: 200,
						Body: io.NopCloser(strings.NewReader(
							`{"authorization": "xx", "unchoked": 1}`,
						)),
					}
					return resp, nil
				case strings.HasPrefix(req.URL.Path, downloadPath):
					resp := &http.Response{
						StatusCode: 200,
						Body: io.NopCloser(strings.NewReader(
							`1234567`,
						)),
					}
					return resp, nil
				case req.URL.Path == collectPath:
					return nil, expected
				default:
					return nil, errors.New("unexpected HTTP request")
				}
			},
		},

		saver: saver,
		sess: &mockable.Session{
			MockableLogger: log.Log,
		},
		tk: &TestKeys{},
	}
	err := r.loop(context.Background(), 1)
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
}

func TestRunnerLoopSuccess(t *testing.T) {
	saver := &tracex.Saver{}
	saver.Write(&tracex.EventConnectOperation{V: &tracex.EventValue{Duration: 150 * time.Millisecond}})

	r := runner{
		callbacks: model.NewPrinterCallbacks(log.Log),

		httpClient: &mocks.HTTPClient{
			MockDo: func(req *http.Request) (*http.Response, error) {
				switch {
				case req.URL.Hostname() == "locate.measurementlab.net":
					resp := &http.Response{
						StatusCode: 200,
						Body: io.NopCloser(strings.NewReader(
							`{"fqdn": "ams01.measurementlab.net"}`,
						)),
					}
					return resp, nil
				case req.URL.Path == negotiatePath:
					resp := &http.Response{
						StatusCode: 200,
						Body: io.NopCloser(strings.NewReader(
							`{"authorization": "xx", "unchoked": 1}`,
						)),
					}
					return resp, nil
				case strings.HasPrefix(req.URL.Path, downloadPath):
					resp := &http.Response{
						StatusCode: 200,
						Body: io.NopCloser(strings.NewReader(
							`1234567`,
						)),
					}
					return resp, nil
				case req.URL.Path == collectPath:
					resp := &http.Response{
						StatusCode: 200,
						Body: io.NopCloser(strings.NewReader(
							`[]`,
						)),
					}
					return resp, nil
				default:
					return nil, errors.New("unexpected HTTP request")
				}
			},
		},

		saver: saver,
		sess: &mockable.Session{
			MockableLogger: log.Log,
		},
		tk: &TestKeys{},
	}
	err := r.loop(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
}
