package dialer_test

import (
	"context"
	"errors"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/engine/netx/bytecounter"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/dialer"
)

func dorequest(ctx context.Context, url string) error {
	txp := http.DefaultTransport.(*http.Transport).Clone()
	defer txp.CloseIdleConnections()
	dialer := dialer.ByteCounterDialer{Dialer: new(net.Dialer)}
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
	if _, err := io.Copy(ioutil.Discard, resp.Body); err != nil {
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
	ctx = dialer.WithSessionByteCounter(ctx, sess)
	if err := dorequest(ctx, "http://www.google.com"); err != nil {
		t.Fatal(err)
	}
	exp := bytecounter.New()
	ctx = dialer.WithExperimentByteCounter(ctx, exp)
	if err := dorequest(ctx, "http://facebook.com"); err != nil {
		t.Fatal(err)
	}
	if sess.Received.Load() <= exp.Received.Load() {
		t.Fatal("session should have received more than experiment")
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
	dialer := dialer.ByteCounterDialer{Dialer: dialer.EOFDialer{}}
	conn, err := dialer.DialContext(context.Background(), "tcp", "www.google.com:80")
	if !errors.Is(err, io.EOF) {
		t.Fatal("not the error we expected")
	}
	if conn != nil {
		t.Fatal("expected nil conn here")
	}
}
