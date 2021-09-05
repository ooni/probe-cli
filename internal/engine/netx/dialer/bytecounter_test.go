package dialer

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/bytecounter"
	"github.com/ooni/probe-cli/v3/internal/iox"
	"github.com/ooni/probe-cli/v3/internal/netxlite/mocks"
)

func dorequest(ctx context.Context, url string) error {
	txp := http.DefaultTransport.(*http.Transport).Clone()
	defer txp.CloseIdleConnections()
	dialer := &byteCounterDialer{Dialer: new(net.Dialer)}
	txp.DialContext = dialer.DialContext
	client := &http.Client{Transport: txp}
	req, err := http.NewRequestWithContext(ctx, "GET", "http://www.google.com", nil)
	if err != nil {
		return err
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	if _, err := iox.CopyContext(ctx, io.Discard, resp.Body); err != nil {
		return err
	}
	return resp.Body.Close()
}

func TestByteCounterNormalUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	sess := bytecounter.New()
	ctx := context.Background()
	ctx = WithSessionByteCounter(ctx, sess)
	if err := dorequest(ctx, "http://www.google.com"); err != nil {
		t.Fatal(err)
	}
	exp := bytecounter.New()
	ctx = WithExperimentByteCounter(ctx, exp)
	if err := dorequest(ctx, "http://facebook.com"); err != nil {
		t.Fatal(err)
	}
	if exp.Received.Load() <= 0 {
		t.Fatal("experiment should have received some bytes")
	}
	if sess.Received.Load() <= exp.Received.Load() {
		t.Fatal("session should have received more than experiment")
	}
	if exp.Sent.Load() <= 0 {
		t.Fatal("experiment should have sent some bytes")
	}
	if sess.Sent.Load() <= exp.Sent.Load() {
		t.Fatal("session should have sent more than experiment")
	}
}

func TestByteCounterNoHandlers(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	ctx := context.Background()
	if err := dorequest(ctx, "http://www.google.com"); err != nil {
		t.Fatal(err)
	}
	if err := dorequest(ctx, "http://facebook.com"); err != nil {
		t.Fatal(err)
	}
}

func TestByteCounterConnectFailure(t *testing.T) {
	dialer := &byteCounterDialer{Dialer: &mocks.Dialer{
		MockDialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
			return nil, io.EOF
		},
	}}
	conn, err := dialer.DialContext(context.Background(), "tcp", "www.google.com:80")
	if !errors.Is(err, io.EOF) {
		t.Fatal("not the error we expected")
	}
	if conn != nil {
		t.Fatal("expected nil conn here")
	}
}
