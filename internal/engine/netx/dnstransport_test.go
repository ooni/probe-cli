package netx

import (
	"errors"
	"strings"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/tracex"
)

func TestNewDNSClientInvalidURL(t *testing.T) {
	dnsclient, err := NewDNSClient(Config{}, "\t\t\t")
	if err == nil || !strings.HasSuffix(err.Error(), "invalid control character in URL") {
		t.Fatal("not the error we expected")
	}
	if dnsclient != nil {
		t.Fatal("expected nil resolver here")
	}
}

func TestNewDNSClientUnsupportedScheme(t *testing.T) {
	dnsclient, err := NewDNSClient(Config{}, "antani:///")
	if err == nil || err.Error() != "unsupported resolver scheme" {
		t.Fatal("not the error we expected")
	}
	if dnsclient != nil {
		t.Fatal("expected nil resolver here")
	}
}

func TestNewDNSClientPowerdnsDoH(t *testing.T) {
	dnsclient, err := NewDNSClient(
		Config{}, "doh://powerdns")
	if err != nil {
		t.Fatal(err)
	}
	r, ok := dnsclient.(*netxlite.SerialResolver)
	if !ok {
		t.Fatal("not the resolver we expected")
	}
	if _, ok := r.Transport().(*netxlite.DNSOverHTTPSTransport); !ok {
		t.Fatal("not the transport we expected")
	}
	dnsclient.CloseIdleConnections()
}

func TestNewDNSClientGoogleDoH(t *testing.T) {
	dnsclient, err := NewDNSClient(
		Config{}, "doh://google")
	if err != nil {
		t.Fatal(err)
	}
	r, ok := dnsclient.(*netxlite.SerialResolver)
	if !ok {
		t.Fatal("not the resolver we expected")
	}
	if _, ok := r.Transport().(*netxlite.DNSOverHTTPSTransport); !ok {
		t.Fatal("not the transport we expected")
	}
	dnsclient.CloseIdleConnections()
}

func TestNewDNSClientCloudflareDoH(t *testing.T) {
	dnsclient, err := NewDNSClient(
		Config{}, "doh://cloudflare")
	if err != nil {
		t.Fatal(err)
	}
	r, ok := dnsclient.(*netxlite.SerialResolver)
	if !ok {
		t.Fatal("not the resolver we expected")
	}
	if _, ok := r.Transport().(*netxlite.DNSOverHTTPSTransport); !ok {
		t.Fatal("not the transport we expected")
	}
	dnsclient.CloseIdleConnections()
}

func TestNewDNSClientCloudflareDoHSaver(t *testing.T) {
	saver := new(tracex.Saver)
	dnsclient, err := NewDNSClient(
		Config{Saver: saver}, "doh://cloudflare")
	if err != nil {
		t.Fatal(err)
	}
	r, ok := dnsclient.(*netxlite.SerialResolver)
	if !ok {
		t.Fatal("not the resolver we expected")
	}
	txp, ok := r.Transport().(*tracex.DNSTransportSaver)
	if !ok {
		t.Fatal("not the transport we expected")
	}
	if _, ok := txp.DNSTransport.(*netxlite.DNSOverHTTPSTransport); !ok {
		t.Fatal("not the transport we expected")
	}
	dnsclient.CloseIdleConnections()
}

func TestNewDNSClientUDP(t *testing.T) {
	dnsclient, err := NewDNSClient(
		Config{}, "udp://8.8.8.8:53")
	if err != nil {
		t.Fatal(err)
	}
	r, ok := dnsclient.(*netxlite.SerialResolver)
	if !ok {
		t.Fatal("not the resolver we expected")
	}
	if _, ok := r.Transport().(*netxlite.DNSOverUDPTransport); !ok {
		t.Fatal("not the transport we expected")
	}
	dnsclient.CloseIdleConnections()
}

func TestNewDNSClientUDPDNSSaver(t *testing.T) {
	saver := new(tracex.Saver)
	dnsclient, err := NewDNSClient(
		Config{Saver: saver}, "udp://8.8.8.8:53")
	if err != nil {
		t.Fatal(err)
	}
	r, ok := dnsclient.(*netxlite.SerialResolver)
	if !ok {
		t.Fatal("not the resolver we expected")
	}
	txp, ok := r.Transport().(*tracex.DNSTransportSaver)
	if !ok {
		t.Fatal("not the transport we expected")
	}
	if _, ok := txp.DNSTransport.(*netxlite.DNSOverUDPTransport); !ok {
		t.Fatal("not the transport we expected")
	}
	dnsclient.CloseIdleConnections()
}

func TestNewDNSClientTCP(t *testing.T) {
	dnsclient, err := NewDNSClient(
		Config{}, "tcp://8.8.8.8:53")
	if err != nil {
		t.Fatal(err)
	}
	r, ok := dnsclient.(*netxlite.SerialResolver)
	if !ok {
		t.Fatal("not the resolver we expected")
	}
	txp, ok := r.Transport().(*netxlite.DNSOverTCPTransport)
	if !ok {
		t.Fatal("not the transport we expected")
	}
	if txp.Network() != "tcp" {
		t.Fatal("not the Network we expected")
	}
	dnsclient.CloseIdleConnections()
}

func TestNewDNSClientTCPDNSSaver(t *testing.T) {
	saver := new(tracex.Saver)
	dnsclient, err := NewDNSClient(
		Config{Saver: saver}, "tcp://8.8.8.8:53")
	if err != nil {
		t.Fatal(err)
	}
	r, ok := dnsclient.(*netxlite.SerialResolver)
	if !ok {
		t.Fatal("not the resolver we expected")
	}
	txp, ok := r.Transport().(*tracex.DNSTransportSaver)
	if !ok {
		t.Fatal("not the transport we expected")
	}
	dotcp, ok := txp.DNSTransport.(*netxlite.DNSOverTCPTransport)
	if !ok {
		t.Fatal("not the transport we expected")
	}
	if dotcp.Network() != "tcp" {
		t.Fatal("not the Network we expected")
	}
	dnsclient.CloseIdleConnections()
}

func TestNewDNSClientDoT(t *testing.T) {
	dnsclient, err := NewDNSClient(
		Config{}, "dot://8.8.8.8:53")
	if err != nil {
		t.Fatal(err)
	}
	r, ok := dnsclient.(*netxlite.SerialResolver)
	if !ok {
		t.Fatal("not the resolver we expected")
	}
	txp, ok := r.Transport().(*netxlite.DNSOverTCPTransport)
	if !ok {
		t.Fatal("not the transport we expected")
	}
	if txp.Network() != "dot" {
		t.Fatal("not the Network we expected")
	}
	dnsclient.CloseIdleConnections()
}

func TestNewDNSClientDoTDNSSaver(t *testing.T) {
	saver := new(tracex.Saver)
	dnsclient, err := NewDNSClient(
		Config{Saver: saver}, "dot://8.8.8.8:53")
	if err != nil {
		t.Fatal(err)
	}
	r, ok := dnsclient.(*netxlite.SerialResolver)
	if !ok {
		t.Fatal("not the resolver we expected")
	}
	txp, ok := r.Transport().(*tracex.DNSTransportSaver)
	if !ok {
		t.Fatal("not the transport we expected")
	}
	dotls, ok := txp.DNSTransport.(*netxlite.DNSOverTCPTransport)
	if !ok {
		t.Fatal("not the transport we expected")
	}
	if dotls.Network() != "dot" {
		t.Fatal("not the Network we expected")
	}
	dnsclient.CloseIdleConnections()
}

func TestNewDNSCLientDoTWithoutPort(t *testing.T) {
	c, err := NewDNSClientWithOverrides(
		Config{}, "dot://8.8.8.8", "", "8.8.8.8", "")
	if err != nil {
		t.Fatal(err)
	}
	if c.Address() != "8.8.8.8:853" {
		t.Fatal("expected default port to be added")
	}
}

func TestNewDNSCLientTCPWithoutPort(t *testing.T) {
	c, err := NewDNSClientWithOverrides(
		Config{}, "tcp://8.8.8.8", "", "8.8.8.8", "")
	if err != nil {
		t.Fatal(err)
	}
	if c.Address() != "8.8.8.8:53" {
		t.Fatal("expected default port to be added")
	}
}

func TestNewDNSCLientUDPWithoutPort(t *testing.T) {
	c, err := NewDNSClientWithOverrides(
		Config{}, "udp://8.8.8.8", "", "8.8.8.8", "")
	if err != nil {
		t.Fatal(err)
	}
	if c.Address() != "8.8.8.8:53" {
		t.Fatal("expected default port to be added")
	}
}

func TestNewDNSClientBadDoTEndpoint(t *testing.T) {
	_, err := NewDNSClient(
		Config{}, "dot://bad:endpoint:53")
	if err == nil || !strings.Contains(err.Error(), "too many colons in address") {
		t.Fatal("expected error with bad endpoint")
	}
}

func TestNewDNSClientBadTCPEndpoint(t *testing.T) {
	_, err := NewDNSClient(
		Config{}, "tcp://bad:endpoint:853")
	if err == nil || !strings.Contains(err.Error(), "too many colons in address") {
		t.Fatal("expected error with bad endpoint")
	}
}

func TestNewDNSClientBadUDPEndpoint(t *testing.T) {
	_, err := NewDNSClient(
		Config{}, "udp://bad:endpoint:853")
	if err == nil || !strings.Contains(err.Error(), "too many colons in address") {
		t.Fatal("expected error with bad endpoint")
	}
}

func TestNewDNSCLientWithInvalidTLSVersion(t *testing.T) {
	_, err := NewDNSClientWithOverrides(
		Config{}, "dot://8.8.8.8", "", "", "TLSv999")
	if !errors.Is(err, netxlite.ErrInvalidTLSVersion) {
		t.Fatalf("not the error we expected: %+v", err)
	}
}
