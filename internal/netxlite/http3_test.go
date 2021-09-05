package netxlite

import (
	"crypto/tls"
	"net/http"
	"testing"

	"github.com/apex/log"
)

func TestHTTP3TransportWorks(t *testing.T) {
	d := &quicDialerResolver{
		Dialer: &quicDialerQUICGo{
			QUICListener: &quicListenerStdlib{},
		},
		Resolver: NewResolver(&ResolverConfig{Logger: log.Log}),
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
