package ndt7

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

func defaultCallbackJSON(data []byte) error {
	return nil
}

func defaultCallbackPerformance(elapsed time.Duration, count int64) {
}

func TestDownloadSetReadDeadlineFailure(t *testing.T) {
	expected := errors.New("mocked error")
	mgr := newDownloadManager(
		&mockableWSConn{
			ReadDeadlineErr: expected,
		},
		defaultCallbackPerformance,
		defaultCallbackJSON,
	)
	err := mgr.run(context.Background())
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
}

func TestDownloadNextReaderFailure(t *testing.T) {
	expected := errors.New("mocked error")
	mgr := newDownloadManager(
		&mockableWSConn{
			NextReaderErr: expected,
		},
		defaultCallbackPerformance,
		defaultCallbackJSON,
	)
	err := mgr.run(context.Background())
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
}

func TestDownloadTextMessageReadAllFailure(t *testing.T) {
	expected := errors.New("mocked error")
	mgr := newDownloadManager(
		&mockableWSConn{
			NextReaderMsgType: websocket.TextMessage,
			NextReaderReader: func() io.Reader {
				return &alwaysFailingReader{
					Err: expected,
				}
			},
		},
		defaultCallbackPerformance,
		defaultCallbackJSON,
	)
	err := mgr.run(context.Background())
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
}

type alwaysFailingReader struct {
	Err error
}

func (r *alwaysFailingReader) Read(p []byte) (int, error) {
	return 0, r.Err
}

func TestDownloadBinaryMessageReadAllFailure(t *testing.T) {
	expected := errors.New("mocked error")
	mgr := newDownloadManager(
		&mockableWSConn{
			NextReaderMsgType: websocket.BinaryMessage,
			NextReaderReader: func() io.Reader {
				return &alwaysFailingReader{
					Err: expected,
				}
			},
		},
		defaultCallbackPerformance,
		defaultCallbackJSON,
	)
	err := mgr.run(context.Background())
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
}

func TestDownloadOnJSONCallbackError(t *testing.T) {
	mgr := newDownloadManager(
		&mockableWSConn{
			NextReaderMsgType: websocket.TextMessage,
			NextReaderReader: func() io.Reader {
				return &invalidJSONReader{}
			},
		},
		defaultCallbackPerformance,
		func(data []byte) error {
			var v interface{}
			return json.Unmarshal(data, &v)
		},
	)
	err := mgr.run(context.Background())
	if err == nil || !strings.HasSuffix(err.Error(), "unexpected end of JSON input") {
		t.Fatal("not the error we expected")
	}
}

type invalidJSONReader struct{}

func (r *invalidJSONReader) Read(p []byte) (int, error) {
	return copy(p, []byte(`{`)), io.EOF
}

func TestDownloadOnJSONLoop(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	mgr := newDownloadManager(
		&mockableWSConn{
			NextReaderMsgType: websocket.TextMessage,
			NextReaderReader: func() io.Reader {
				return &goodJSONReader{}
			},
		},
		defaultCallbackPerformance,
		func(data []byte) error {
			var v interface{}
			return json.Unmarshal(data, &v)
		},
	)
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	err := mgr.run(ctx)
	if err != nil {
		t.Fatal(err)
	}
}

type goodJSONReader struct{}

func (r *goodJSONReader) Read(p []byte) (int, error) {
	return copy(p, []byte(`{}`)), io.EOF
}

func TestDownloadReduceErr(t *testing.T) {
	mgr := downloadManager{}
	if mgr.reduceErr(context.Canceled) != nil {
		t.Fatal("reduceErr not working as intended for context.Canceled")
	}
	if mgr.reduceErr(context.DeadlineExceeded) != nil {
		t.Fatal("reduceErr not working as intended for context.DeadlineExceeded")
	}
	if mgr.reduceErr(nil) != nil {
		t.Fatal("reduceErr not working as intended for nil")
	}
	expected := errors.New("mocked error")
	if mgr.reduceErr(expected) != expected {
		t.Fatal("reduceErr not working as intended for other errors")
	}
}
