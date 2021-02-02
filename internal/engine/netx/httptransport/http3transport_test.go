package httptransport_test

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/httptransport"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/selfcensor"
)

type MockQUICDialer struct{}

func (d MockQUICDialer) Dial(network, host string, tlsCfg *tls.Config, cfg *quic.Config) (quic.EarlySession, error) {
	return quic.DialAddrEarly(host, tlsCfg, cfg)
}

type MockSNIQUICDialer struct {
	namech chan string
}

func (d MockSNIQUICDialer) Dial(network, host string, tlsCfg *tls.Config, cfg *quic.Config) (quic.EarlySession, error) {
	d.namech <- tlsCfg.ServerName
	return quic.DialAddrEarly(host, tlsCfg, cfg)
}

type MockCertQUICDialer struct {
	certch chan *x509.CertPool
}

func (d MockCertQUICDialer) Dial(network, host string, tlsCfg *tls.Config, cfg *quic.Config) (quic.EarlySession, error) {
	d.certch <- tlsCfg.RootCAs
	return quic.DialAddrEarly(host, tlsCfg, cfg)
}

func TestHTTP3TransportSNI(t *testing.T) {
	namech := make(chan string, 1)
	sni := "sni.org"
	txp := httptransport.NewHTTP3Transport(httptransport.Config{
		Dialer: selfcensor.SystemDialer{}, QUICDialer: MockSNIQUICDialer{namech: namech}, TLSConfig: &tls.Config{ServerName: sni}})
	req, err := http.NewRequest("GET", "https://www.google.com", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := txp.RoundTrip(req)
	if err == nil {
		t.Fatal("expected error here")
	}
	if resp != nil {
		t.Fatal("expected nil resp here")
	}
	if !strings.Contains(err.Error(), "certificate is valid for www.google.com, not "+sni) {
		t.Fatal("unexpected error type", err)
	}
	servername := <-namech
	if servername != sni {
		t.Fatal("unexpected server name", servername)
	}
}

func TestHTTP3TransportSNINoVerify(t *testing.T) {
	namech := make(chan string, 1)
	sni := "sni.org"
	txp := httptransport.NewHTTP3Transport(httptransport.Config{
		Dialer: selfcensor.SystemDialer{}, QUICDialer: MockSNIQUICDialer{namech: namech}, TLSConfig: &tls.Config{ServerName: sni, InsecureSkipVerify: true}})
	req, err := http.NewRequest("GET", "https://www.google.com", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := txp.RoundTrip(req)
	if err != nil {
		t.Fatalf("unexpected error: %+v", err)
	}
	if resp == nil {
		t.Fatal("unexpected nil resp")
	}
	servername := <-namech
	if servername != sni {
		t.Fatal("unexpected server name", servername)
	}
}

func TestHTTP3TransportCABundle(t *testing.T) {
	certch := make(chan *x509.CertPool, 1)
	certpool := x509.NewCertPool()
	txp := httptransport.NewHTTP3Transport(httptransport.Config{
		Dialer: selfcensor.SystemDialer{}, QUICDialer: MockCertQUICDialer{certch: certch}, TLSConfig: &tls.Config{RootCAs: certpool}})
	req, err := http.NewRequest("GET", "https://www.google.com", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := txp.RoundTrip(req)
	if err == nil {
		t.Fatal("expected error here")
	}
	if resp != nil {
		t.Fatal("expected nil resp here")
	}
	// since the certificate pool is empty, the unknown authority error should be thrown
	if !strings.Contains(err.Error(), "certificate signed by unknown authority") {
		t.Fatal("unexpected error type")
	}
	certs := <-certch
	if certs != certpool {
		t.Fatal("not the certpool we expected")
	}

}

func TestUnitHTTP3TransportSuccess(t *testing.T) {
	txp := httptransport.NewHTTP3Transport(httptransport.Config{
		Dialer: selfcensor.SystemDialer{}, QUICDialer: MockQUICDialer{}})

	req, err := http.NewRequest("GET", "https://www.google.com", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := txp.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp == nil {
		t.Fatal("unexpected nil response here")
	}
	if resp.StatusCode != 200 {
		t.Fatal("HTTP statuscode should be 200 OK", resp.StatusCode)
	}
}

func TestUnitHTTP3TransportFailure(t *testing.T) {
	txp := httptransport.NewHTTP3Transport(httptransport.Config{
		Dialer: selfcensor.SystemDialer{}, QUICDialer: MockQUICDialer{}})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // so that the request immediately fails
	req, err := http.NewRequestWithContext(ctx, "GET", "https://www.google.com", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := txp.RoundTrip(req)
	if err == nil {
		t.Fatal("expected error here")
	}
	// context.Canceled error occurs if the test host supports QUIC
	// timeout error ("Handshake did not complete in time") occurs if the test host does not support QUIC
	if !(errors.Is(err, context.Canceled) || strings.HasSuffix(err.Error(), "Handshake did not complete in time")) {
		t.Fatal("not the error we expected", err)
	}
	if resp != nil {
		t.Fatal("expected nil response here")
	}
}
