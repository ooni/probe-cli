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
	"github.com/ooni/probe-cli/v3/internal/tracex"
)

func TestRunnerLoopLocateFailure(t *testing.T) {
	expected := errors.New("mocked error")
	r := runner{
		callbacks: model.NewPrinterCallbacks(log.Log),
		httpClient: &http.Client{
			Transport: FakeHTTPTransport{
				err: expected,
			},
		},
		saver: new(tracex.Saver),
		sess: &mockable.Session{
			MockableLogger: log.Log,
		},
		tk: new(TestKeys),
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
		httpClient: &http.Client{
			Transport: &FakeHTTPTransportStack{
				all: []FakeHTTPTransport{
					{
						resp: &http.Response{
							Body: io.NopCloser(strings.NewReader(
								`{"fqdn": "ams01.measurementlab.net"}`)),
							StatusCode: 200,
						},
					},
					{err: expected},
				},
			},
		},
		saver: new(tracex.Saver),
		sess: &mockable.Session{
			MockableLogger: log.Log,
		},
		tk: new(TestKeys),
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
		httpClient: &http.Client{
			Transport: &FakeHTTPTransportStack{
				all: []FakeHTTPTransport{
					{
						resp: &http.Response{
							Body: io.NopCloser(strings.NewReader(
								`{"fqdn": "ams01.measurementlab.net"}`)),
							StatusCode: 200,
						},
					},
					{
						resp: &http.Response{
							Body: io.NopCloser(strings.NewReader(
								`{"authorization": "xx", "unchoked": 1}`)),
							StatusCode: 200,
						},
					},
					{err: expected},
				},
			},
		},
		saver: new(tracex.Saver),
		sess: &mockable.Session{
			MockableLogger: log.Log,
		},
		tk: new(TestKeys),
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
		httpClient: &http.Client{
			Transport: &FakeHTTPTransportStack{
				all: []FakeHTTPTransport{
					{
						resp: &http.Response{
							Body: io.NopCloser(strings.NewReader(
								`{"fqdn": "ams01.measurementlab.net"}`)),
							StatusCode: 200,
						},
					},
					{
						resp: &http.Response{
							Body: io.NopCloser(strings.NewReader(
								`{"authorization": "xx", "unchoked": 1}`)),
							StatusCode: 200,
						},
					},
					{
						resp: &http.Response{
							Body:       io.NopCloser(strings.NewReader(`1234567`)),
							StatusCode: 200,
						},
					},
					{err: expected},
				},
			},
		},
		saver: saver,
		sess: &mockable.Session{
			MockableLogger: log.Log,
		},
		tk: new(TestKeys),
	}
	err := r.loop(context.Background(), 1)
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
}

func TestRunnerLoopSuccess(t *testing.T) {
	saver := new(tracex.Saver)
	saver.Write(&tracex.EventConnectOperation{V: &tracex.EventValue{Duration: 150 * time.Millisecond}})
	r := runner{
		callbacks: model.NewPrinterCallbacks(log.Log),
		httpClient: &http.Client{
			Transport: &FakeHTTPTransportStack{
				all: []FakeHTTPTransport{
					{
						resp: &http.Response{
							Body: io.NopCloser(strings.NewReader(
								`{"fqdn": "ams01.measurementlab.net"}`)),
							StatusCode: 200,
						},
					},
					{
						resp: &http.Response{
							Body: io.NopCloser(strings.NewReader(
								`{"authorization": "xx", "unchoked": 1}`)),
							StatusCode: 200,
						},
					},
					{
						resp: &http.Response{
							Body:       io.NopCloser(strings.NewReader(`1234567`)),
							StatusCode: 200,
						},
					},
					{
						resp: &http.Response{
							Body:       io.NopCloser(strings.NewReader(`[]`)),
							StatusCode: 200,
						},
					},
				},
			},
		},
		saver: saver,
		sess: &mockable.Session{
			MockableLogger: log.Log,
		},
		tk: new(TestKeys),
	}
	err := r.loop(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
}
