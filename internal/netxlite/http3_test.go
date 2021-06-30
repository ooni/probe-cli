package netxlite

import (
	"crypto/tls"
	"net"
	"net/http"
	"testing"
)

func TestHTTP3TransportWorks(t *testing.T) {
	d := &QUICDialerResolver{
		Dialer: &QUICDialerQUICGo{
			QUICListener: &QUICListenerStdlib{},
		},
		Resolver: &net.Resolver{},
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
