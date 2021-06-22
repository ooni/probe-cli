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
	"github.com/montanaflynn/stats"
	"github.com/ooni/probe-cli/v3/internal/engine/mockable"
	"github.com/ooni/probe-cli/v3/internal/engine/model"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/errorx"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/trace"
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
		saver: new(trace.Saver),
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
		saver: new(trace.Saver),
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
		saver: new(trace.Saver),
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
	saver := new(trace.Saver)
	saver.Write(trace.Event{Name: errorx.ConnectOperation, Duration: 150 * time.Millisecond})
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
	saver := new(trace.Saver)
	saver.Write(trace.Event{Name: errorx.ConnectOperation, Duration: 150 * time.Millisecond})
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

func TestTestKeysAnalyzeWithNoData(t *testing.T) {
	tk := &TestKeys{}
	err := tk.analyze()
	if !errors.Is(err, stats.EmptyInputErr) {
		t.Fatal("expected an error here")
	}
}

func TestTestKeysAnalyzeMedian(t *testing.T) {
	tk := &TestKeys{
		ReceiverData: []clientResults{
			{
				Rate: 1,
			},
			{
				Rate: 2,
			},
			{
				Rate: 3,
			},
		},
	}
	err := tk.analyze()
	if err != nil {
		t.Fatal(err)
	}
	if tk.Simple.MedianBitrate != 2 {
		t.Fatal("unexpected median value")
	}
}

func TestTestKeysAnalyzeMinPlayoutDelay(t *testing.T) {
	tk := &TestKeys{
		ReceiverData: []clientResults{
			{
				ElapsedTarget: 2,
				Elapsed:       1.4,
			},
			{
				ElapsedTarget: 2,
				Elapsed:       3.0,
			},
			{
				ElapsedTarget: 2,
				Elapsed:       1.8,
			},
		},
	}
	err := tk.analyze()
	if err != nil {
		t.Fatal(err)
	}
	if tk.Simple.MinPlayoutDelay < 0.99 || tk.Simple.MinPlayoutDelay > 1.01 {
		t.Fatal("unexpected min-playout-delay value")
	}
}

func TestNewExperimentMeasurer(t *testing.T) {
	measurer := NewExperimentMeasurer(Config{})
	if measurer.ExperimentName() != "dash" {
		t.Fatal("unexpected name")
	}
	if measurer.ExperimentVersion() != "0.12.0" {
		t.Fatal("unexpected version")
	}
}

func TestMeasureWithCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cause failure
	measurement := new(model.Measurement)
	m := &Measurer{}
	err := m.Run(
		ctx,
		&mockable.Session{
			MockableHTTPClient: http.DefaultClient,
			MockableLogger:     log.Log,
		},
		measurement,
		model.NewPrinterCallbacks(log.Log),
	)
	if !errors.Is(err, context.Canceled) {
		t.Fatal("unexpected error value")
	}
	sk, err := m.GetSummaryKeys(measurement)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := sk.(SummaryKeys); !ok {
		t.Fatal("invalid type for summary keys")
	}
}

func TestSummaryKeysInvalidType(t *testing.T) {
	measurement := new(model.Measurement)
	m := &Measurer{}
	_, err := m.GetSummaryKeys(measurement)
	if err.Error() != "invalid test keys type" {
		t.Fatal("not the error we expected")
	}
}

func TestSummaryKeysGood(t *testing.T) {
	measurement := &model.Measurement{TestKeys: &TestKeys{Simple: Simple{
		ConnectLatency:  1234,
		MedianBitrate:   123,
		MinPlayoutDelay: 12,
	}}}
	m := &Measurer{}
	osk, err := m.GetSummaryKeys(measurement)
	if err != nil {
		t.Fatal(err)
	}
	sk := osk.(SummaryKeys)
	if sk.Latency != 1234 {
		t.Fatal("invalid latency")
	}
	if sk.Bitrate != 123 {
		t.Fatal("invalid bitrate")
	}
	if sk.Delay != 12 {
		t.Fatal("invalid delay")
	}
	if sk.IsAnomaly {
		t.Fatal("invalid isAnomaly")
	}
}
