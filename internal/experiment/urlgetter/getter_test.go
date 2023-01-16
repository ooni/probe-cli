package urlgetter

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/legacy/mockable"
)

func TestGetterHTTPSWithTunnelCannotCreateTempDir(t *testing.T) {
	expected := errors.New("mocked error")
	ctx := context.Background()
	g := Getter{
		Config: Config{
			NoFollowRedirects: true, // reduce number of events
			Tunnel:            "fake",
		},
		Session: &mockable.Session{
			MockableHTTPClient: http.DefaultClient,
			MockableLogger:     log.Log,
		},
		Target: "https://www.google.com",
		testIOUtilTempDir: func(dir, pattern string) (string, error) {
			return "", expected
		},
	}
	tk, err := g.Get(ctx)
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected", err)
	}
	if tk.Agent != "agent" {
		t.Fatal("not the Agent we expected")
	}
	if tk.BootstrapTime != 0 {
		t.Fatal("not the BootstrapTime we expected")
	}
	if tk.FailedOperation == nil || *tk.FailedOperation != "top_level" {
		t.Fatal("not the FailedOperation we expected")
	}
	if tk.Failure == nil || *tk.Failure != "unknown_failure: mocked error" {
		t.Fatal("not the Failure we expected")
	}
	if len(tk.NetworkEvents) != 0 {
		t.Fatal("not the NetworkEvents we expected")
	}
	if len(tk.Queries) != 0 {
		t.Fatal("not the Queries we expected")
	}
	if len(tk.TCPConnect) != 0 {
		t.Fatal("not the TCPConnect we expected")
	}
	if len(tk.Requests) != 0 {
		t.Fatal("not the Requests we expected")
	}
	if tk.SOCKSProxy != "" {
		t.Fatal("not the SOCKSProxy we expected")
	}
	if len(tk.TLSHandshakes) != 0 {
		t.Fatal("not the TLSHandshakes we expected")
	}
	if tk.Tunnel != "fake" {
		t.Fatal("not the Tunnel we expected")
	}
	if tk.HTTPResponseStatus != 0 {
		t.Fatal("not the HTTPResponseStatus we expected")
	}
	if len(tk.HTTPResponseBody) != 0 {
		t.Fatal("not the HTTPResponseBody we expected")
	}
}
