package ndt7

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/apex/log"
	"github.com/gorilla/websocket"
)

func TestDialDownloadWithCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately halt
	mgr := newDialManager("wss://hostname.fake", log.Log, "miniooni/0.1.0-dev")
	conn, err := mgr.dialDownload(ctx)
	if err == nil || !strings.HasSuffix(err.Error(), "operation was canceled") {
		t.Fatal("not the error we expected")
	}
	if conn != nil {
		t.Fatal("expected nil conn here")
	}
}

func TestDialUploadWithCancelledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // immediately halt
	mgr := newDialManager("wss://hostname.fake", log.Log, "miniooni/0.1.0-dev")
	conn, err := mgr.dialUpload(ctx)
	if err == nil || !strings.HasSuffix(err.Error(), "operation was canceled") {
		t.Fatal("not the error we expected")
	}
	if conn != nil {
		t.Fatal("expected nil conn here")
	}
}

func TestDialIncludesUserAgent(t *testing.T) {
	do := func(testName string) {
		var userAgent string
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userAgent = r.UserAgent()
			w.WriteHeader(500)
		})
		server := httptest.NewServer(handler)
		defer server.Close()
		url, err := url.Parse(server.URL)
		if err != nil {
			t.Fatal(err)
		}
		url.Scheme = "ws"
		mgr := newDialManager(url.String(), log.Log, "miniooni/0.1.0-dev")
		conn, err := mgr.dialWithTestName(context.Background(), testName)
		if !errors.Is(err, websocket.ErrBadHandshake) {
			t.Fatal("not the error we expected")
		}
		if conn != nil {
			t.Fatal("expected nil conn here")
		}
		if userAgent != "miniooni/0.1.0-dev" {
			t.Fatal("User-Agent not sent")
		}
	}
	do("download")
	do("upload")
}
