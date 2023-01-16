package netx

import (
	"context"
	"errors"
	"net"
	"net/http"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/tracex"
)

func TestNewHTTPTransportWithDialer(t *testing.T) {
	expected := errors.New("mocked error")
	dialer := &mocks.Dialer{
		MockDialContext: func(ctx context.Context, network, address string) (net.Conn, error) {
			return nil, expected
		},
	}
	txp := NewHTTPTransport(Config{
		Dialer: dialer,
	})
	client := &http.Client{Transport: txp}
	resp, err := client.Get("http://www.google.com")
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
	if resp != nil {
		t.Fatal("not the response we expected")
	}
}

func TestNewHTTPTransportWithSaver(t *testing.T) {
	saver := new(tracex.Saver)
	txp := NewHTTPTransport(Config{
		Saver: saver,
	})
	stxptxp, ok := txp.(*tracex.HTTPTransportSaver)
	if !ok {
		t.Fatal("not the transport we expected")
	}
	if stxptxp.Saver != saver {
		t.Fatal("not the logger we expected")
	}
	if stxptxp.Saver != saver {
		t.Fatal("not the logger we expected")
	}
	// We are going to trust the underlying type returned by netxlite
}
