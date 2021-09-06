package netxlite

import (
	"crypto/tls"
	"net/http"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/netxlite/mocks"
)

func TestHTTP3TransportWorks(t *testing.T) {
	d := &quicDialerResolver{
		Dialer: &quicDialerQUICGo{
			QUICListener: &quicListenerStdlib{},
		},
		Resolver: NewResolverSystem(log.Log),
	}
	txp := NewHTTP3Transport(d, &tls.Config{})
	client := &http.Client{Transport: txp}
	resp, err := client.Get("https://www.google.com/robots.txt")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	txp.CloseIdleConnections()
}

func TestHTTP3TransportClosesIdleConnections(t *testing.T) {
	var called bool
	d := &mocks.QUICDialer{
		MockCloseIdleConnections: func() {
			called = true
		},
	}
	txp := NewHTTP3Transport(d, &tls.Config{})
	client := &http.Client{Transport: txp}
	client.CloseIdleConnections()
	if !called {
		t.Fatal("not called")
	}
}
