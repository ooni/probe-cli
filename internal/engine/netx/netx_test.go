package netx

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"net/http"
	"strings"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/bytecounter"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/tracex"
)

func TestNewResolverVanilla(t *testing.T) {
	r := NewResolver(Config{})
	ir, ok := r.(*netxlite.ResolverIDNA)
	if !ok {
		t.Fatal("not the resolver we expected")
	}
	ewr, ok := ir.Resolver.(*netxlite.ErrorWrapperResolver)
	if !ok {
		t.Fatal("not the resolver we expected")
	}
	ar, ok := ewr.Resolver.(*netxlite.AddressResolver)
	if !ok {
		t.Fatal("not the resolver we expected")
	}
	_, ok = ar.Resolver.(*netxlite.ResolverSystemDoNotInstantiate)
	if !ok {
		t.Fatal("not the resolver we expected")
	}
}

func TestNewResolverSpecificResolver(t *testing.T) {
	r := NewResolver(Config{
		BaseResolver: &netxlite.BogonResolver{
			// not initialized because it doesn't matter in this context
		},
	})
	ir, ok := r.(*netxlite.ResolverIDNA)
	if !ok {
		t.Fatal("not the resolver we expected")
	}
	ewr, ok := ir.Resolver.(*netxlite.ErrorWrapperResolver)
	if !ok {
		t.Fatal("not the resolver we expected")
	}
	ar, ok := ewr.Resolver.(*netxlite.AddressResolver)
	if !ok {
		t.Fatal("not the resolver we expected")
	}
	_, ok = ar.Resolver.(*netxlite.BogonResolver)
	if !ok {
		t.Fatal("not the resolver we expected")
	}
}

func TestNewResolverWithBogonFilter(t *testing.T) {
	r := NewResolver(Config{
		BogonIsError: true,
	})
	ir, ok := r.(*netxlite.ResolverIDNA)
	if !ok {
		t.Fatal("not the resolver we expected")
	}
	ewr, ok := ir.Resolver.(*netxlite.ErrorWrapperResolver)
	if !ok {
		t.Fatal("not the resolver we expected")
	}
	br, ok := ewr.Resolver.(*netxlite.BogonResolver)
	if !ok {
		t.Fatal("not the resolver we expected")
	}
	ar, ok := br.Resolver.(*netxlite.AddressResolver)
	if !ok {
		t.Fatal("not the resolver we expected")
	}
	_, ok = ar.Resolver.(*netxlite.ResolverSystemDoNotInstantiate)
	if !ok {
		t.Fatal("not the resolver we expected")
	}
}

func TestNewResolverWithLogging(t *testing.T) {
	r := NewResolver(Config{
		Logger: log.Log,
	})
	ir, ok := r.(*netxlite.ResolverIDNA)
	if !ok {
		t.Fatal("not the resolver we expected")
	}
	lr, ok := ir.Resolver.(*netxlite.ResolverLogger)
	if !ok {
		t.Fatal("not the resolver we expected")
	}
	if lr.Logger != log.Log {
		t.Fatal("not the logger we expected")
	}
	ewr, ok := lr.Resolver.(*netxlite.ErrorWrapperResolver)
	if !ok {
		t.Fatalf("not the resolver we expected")
	}
	ar, ok := ewr.Resolver.(*netxlite.AddressResolver)
	if !ok {
		t.Fatal("not the resolver we expected")
	}
	_, ok = ar.Resolver.(*netxlite.ResolverSystemDoNotInstantiate)
	if !ok {
		t.Fatal("not the resolver we expected")
	}
}

func TestNewResolverWithSaver(t *testing.T) {
	saver := new(tracex.Saver)
	r := NewResolver(Config{
		ResolveSaver: saver,
	})
	ir, ok := r.(*netxlite.ResolverIDNA)
	if !ok {
		t.Fatal("not the resolver we expected")
	}
	sr, ok := ir.Resolver.(*tracex.ResolverSaver)
	if !ok {
		t.Fatal("not the resolver we expected")
	}
	if sr.Saver != saver {
		t.Fatal("not the saver we expected")
	}
	ewr, ok := sr.Resolver.(*netxlite.ErrorWrapperResolver)
	if !ok {
		t.Fatal("not the resolver we expected")
	}
	ar, ok := ewr.Resolver.(*netxlite.AddressResolver)
	if !ok {
		t.Fatal("not the resolver we expected")
	}
	_, ok = ar.Resolver.(*netxlite.ResolverSystemDoNotInstantiate)
	if !ok {
		t.Fatal("not the resolver we expected")
	}
}

func TestNewResolverWithReadWriteCache(t *testing.T) {
	r := NewResolver(Config{
		CacheResolutions: true,
	})
	ir, ok := r.(*netxlite.ResolverIDNA)
	if !ok {
		t.Fatal("not the resolver we expected")
	}
	ewr, ok := ir.Resolver.(*netxlite.ErrorWrapperResolver)
	if !ok {
		t.Fatal("not the resolver we expected")
	}
	cr, ok := ewr.Resolver.(*CacheResolver)
	if !ok {
		t.Fatal("not the resolver we expected")
	}
	if cr.ReadOnly != false {
		t.Fatal("expected readwrite cache here")
	}
	ar, ok := cr.Resolver.(*netxlite.AddressResolver)
	if !ok {
		t.Fatal("not the resolver we expected")
	}
	_, ok = ar.Resolver.(*netxlite.ResolverSystemDoNotInstantiate)
	if !ok {
		t.Fatal("not the resolver we expected")
	}
}

func TestNewResolverWithPrefilledReadonlyCache(t *testing.T) {
	r := NewResolver(Config{
		DNSCache: map[string][]string{
			"dns.google.com": {"8.8.8.8"},
		},
	})
	ir, ok := r.(*netxlite.ResolverIDNA)
	if !ok {
		t.Fatal("not the resolver we expected")
	}
	ewr, ok := ir.Resolver.(*netxlite.ErrorWrapperResolver)
	if !ok {
		t.Fatal("not the resolver we expected")
	}
	cr, ok := ewr.Resolver.(*CacheResolver)
	if !ok {
		t.Fatal("not the resolver we expected")
	}
	if cr.ReadOnly != true {
		t.Fatal("expected readonly cache here")
	}
	if cr.Get("dns.google.com")[0] != "8.8.8.8" {
		t.Fatal("cache not correctly prefilled")
	}
	ar, ok := cr.Resolver.(*netxlite.AddressResolver)
	if !ok {
		t.Fatal("not the resolver we expected")
	}
	_, ok = ar.Resolver.(*netxlite.ResolverSystemDoNotInstantiate)
	if !ok {
		t.Fatal("not the resolver we expected")
	}
}

func TestNewTLSDialerVanilla(t *testing.T) {
	td := NewTLSDialer(Config{})
	rtd, ok := td.(*netxlite.TLSDialerLegacy)
	if !ok {
		t.Fatal("not the TLSDialer we expected")
	}
	if len(rtd.Config.NextProtos) != 2 {
		t.Fatal("invalid len(config.NextProtos)")
	}
	if rtd.Config.NextProtos[0] != "h2" || rtd.Config.NextProtos[1] != "http/1.1" {
		t.Fatal("invalid Config.NextProtos")
	}
	if rtd.Config.RootCAs != defaultCertPool {
		t.Fatal("invalid Config.RootCAs")
	}
	if rtd.Dialer == nil {
		t.Fatal("invalid Dialer")
	}
	if rtd.TLSHandshaker == nil {
		t.Fatal("invalid TLSHandshaker")
	}
	ewth, ok := rtd.TLSHandshaker.(*netxlite.ErrorWrapperTLSHandshaker)
	if !ok {
		t.Fatal("not the TLSHandshaker we expected")
	}
	if _, ok := ewth.TLSHandshaker.(*netxlite.TLSHandshakerConfigurable); !ok {
		t.Fatal("not the TLSHandshaker we expected")
	}
}

func TestNewTLSDialerWithConfig(t *testing.T) {
	td := NewTLSDialer(Config{
		TLSConfig: new(tls.Config),
	})
	rtd, ok := td.(*netxlite.TLSDialerLegacy)
	if !ok {
		t.Fatal("not the TLSDialer we expected")
	}
	if len(rtd.Config.NextProtos) != 0 {
		t.Fatal("invalid len(config.NextProtos)")
	}
	if rtd.Config.RootCAs != defaultCertPool {
		t.Fatal("invalid Config.RootCAs")
	}
	if rtd.Dialer == nil {
		t.Fatal("invalid Dialer")
	}
	if rtd.TLSHandshaker == nil {
		t.Fatal("invalid TLSHandshaker")
	}
	ewth, ok := rtd.TLSHandshaker.(*netxlite.ErrorWrapperTLSHandshaker)
	if !ok {
		t.Fatal("not the TLSHandshaker we expected")
	}
	if _, ok := ewth.TLSHandshaker.(*netxlite.TLSHandshakerConfigurable); !ok {
		t.Fatal("not the TLSHandshaker we expected")
	}
}

func TestNewTLSDialerWithLogging(t *testing.T) {
	td := NewTLSDialer(Config{
		Logger: log.Log,
	})
	rtd, ok := td.(*netxlite.TLSDialerLegacy)
	if !ok {
		t.Fatal("not the TLSDialer we expected")
	}
	if len(rtd.Config.NextProtos) != 2 {
		t.Fatal("invalid len(config.NextProtos)")
	}
	if rtd.Config.NextProtos[0] != "h2" || rtd.Config.NextProtos[1] != "http/1.1" {
		t.Fatal("invalid Config.NextProtos")
	}
	if rtd.Config.RootCAs != defaultCertPool {
		t.Fatal("invalid Config.RootCAs")
	}
	if rtd.Dialer == nil {
		t.Fatal("invalid Dialer")
	}
	if rtd.TLSHandshaker == nil {
		t.Fatal("invalid TLSHandshaker")
	}
	lth, ok := rtd.TLSHandshaker.(*netxlite.TLSHandshakerLogger)
	if !ok {
		t.Fatal("not the TLSHandshaker we expected")
	}
	if lth.DebugLogger != log.Log {
		t.Fatal("not the Logger we expected")
	}
	ewth, ok := lth.TLSHandshaker.(*netxlite.ErrorWrapperTLSHandshaker)
	if !ok {
		t.Fatal("not the TLSHandshaker we expected")
	}
	if _, ok := ewth.TLSHandshaker.(*netxlite.TLSHandshakerConfigurable); !ok {
		t.Fatal("not the TLSHandshaker we expected")
	}
}

func TestNewTLSDialerWithSaver(t *testing.T) {
	saver := new(tracex.Saver)
	td := NewTLSDialer(Config{
		TLSSaver: saver,
	})
	rtd, ok := td.(*netxlite.TLSDialerLegacy)
	if !ok {
		t.Fatal("not the TLSDialer we expected")
	}
	if len(rtd.Config.NextProtos) != 2 {
		t.Fatal("invalid len(config.NextProtos)")
	}
	if rtd.Config.NextProtos[0] != "h2" || rtd.Config.NextProtos[1] != "http/1.1" {
		t.Fatal("invalid Config.NextProtos")
	}
	if rtd.Config.RootCAs != defaultCertPool {
		t.Fatal("invalid Config.RootCAs")
	}
	if rtd.Dialer == nil {
		t.Fatal("invalid Dialer")
	}
	if rtd.TLSHandshaker == nil {
		t.Fatal("invalid TLSHandshaker")
	}
	sth, ok := rtd.TLSHandshaker.(*tracex.TLSHandshakerSaver)
	if !ok {
		t.Fatal("not the TLSHandshaker we expected")
	}
	if sth.Saver != saver {
		t.Fatal("not the Logger we expected")
	}
	ewth, ok := sth.TLSHandshaker.(*netxlite.ErrorWrapperTLSHandshaker)
	if !ok {
		t.Fatal("not the TLSHandshaker we expected")
	}
	if _, ok := ewth.TLSHandshaker.(*netxlite.TLSHandshakerConfigurable); !ok {
		t.Fatal("not the TLSHandshaker we expected")
	}
}

func TestNewTLSDialerWithNoTLSVerifyAndConfig(t *testing.T) {
	td := NewTLSDialer(Config{
		TLSConfig:   new(tls.Config),
		NoTLSVerify: true,
	})
	rtd, ok := td.(*netxlite.TLSDialerLegacy)
	if !ok {
		t.Fatal("not the TLSDialer we expected")
	}
	if len(rtd.Config.NextProtos) != 0 {
		t.Fatal("invalid len(config.NextProtos)")
	}
	if rtd.Config.InsecureSkipVerify != true {
		t.Fatal("expected true InsecureSkipVerify")
	}
	if rtd.Config.RootCAs != defaultCertPool {
		t.Fatal("invalid Config.RootCAs")
	}
	if rtd.Dialer == nil {
		t.Fatal("invalid Dialer")
	}
	if rtd.TLSHandshaker == nil {
		t.Fatal("invalid TLSHandshaker")
	}
	ewth, ok := rtd.TLSHandshaker.(*netxlite.ErrorWrapperTLSHandshaker)
	if !ok {
		t.Fatal("not the TLSHandshaker we expected")
	}
	if _, ok := ewth.TLSHandshaker.(*netxlite.TLSHandshakerConfigurable); !ok {
		t.Fatal("not the TLSHandshaker we expected")
	}
}

func TestNewTLSDialerWithNoTLSVerifyAndNoConfig(t *testing.T) {
	td := NewTLSDialer(Config{
		NoTLSVerify: true,
	})
	rtd, ok := td.(*netxlite.TLSDialerLegacy)
	if !ok {
		t.Fatal("not the TLSDialer we expected")
	}
	if len(rtd.Config.NextProtos) != 2 {
		t.Fatal("invalid len(config.NextProtos)")
	}
	if rtd.Config.NextProtos[0] != "h2" || rtd.Config.NextProtos[1] != "http/1.1" {
		t.Fatal("invalid Config.NextProtos")
	}
	if rtd.Config.InsecureSkipVerify != true {
		t.Fatal("expected true InsecureSkipVerify")
	}
	if rtd.Config.RootCAs != defaultCertPool {
		t.Fatal("invalid Config.RootCAs")
	}
	if rtd.Dialer == nil {
		t.Fatal("invalid Dialer")
	}
	if rtd.TLSHandshaker == nil {
		t.Fatal("invalid TLSHandshaker")
	}
	ewth, ok := rtd.TLSHandshaker.(*netxlite.ErrorWrapperTLSHandshaker)
	if !ok {
		t.Fatal("not the TLSHandshaker we expected")
	}
	if _, ok := ewth.TLSHandshaker.(*netxlite.TLSHandshakerConfigurable); !ok {
		t.Fatal("not the TLSHandshaker we expected")
	}
}

func TestNewVanilla(t *testing.T) {
	txp := NewHTTPTransport(Config{})
	if _, ok := txp.(*netxlite.HTTPTransportWrapper); !ok {
		t.Fatal("not the transport we expected")
	}
}

func TestNewWithDialer(t *testing.T) {
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

func TestNewWithTLSDialer(t *testing.T) {
	expected := errors.New("mocked error")
	tlsDialer := &netxlite.TLSDialerLegacy{
		Config: new(tls.Config),
		Dialer: &mocks.Dialer{
			MockDialContext: func(ctx context.Context, network string, address string) (net.Conn, error) {
				return nil, expected
			},
			MockCloseIdleConnections: func() {
				// nothing
			},
		},
		TLSHandshaker: &netxlite.TLSHandshakerConfigurable{},
	}
	txp := NewHTTPTransport(Config{
		TLSDialer: tlsDialer,
	})
	client := &http.Client{Transport: txp}
	resp, err := client.Get("https://www.google.com")
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected")
	}
	if resp != nil {
		t.Fatal("not the response we expected")
	}
}

func TestNewWithByteCounter(t *testing.T) {
	counter := bytecounter.New()
	txp := NewHTTPTransport(Config{
		ByteCounter: counter,
	})
	bctxp, ok := txp.(*bytecounter.HTTPTransport)
	if !ok {
		t.Fatal("not the transport we expected")
	}
	if bctxp.Counter != counter {
		t.Fatal("not the byte counter we expected")
	}
	if _, ok := bctxp.HTTPTransport.(*netxlite.HTTPTransportWrapper); !ok {
		t.Fatal("not the transport we expected")
	}
}

func TestNewWithLogger(t *testing.T) {
	txp := NewHTTPTransport(Config{
		Logger: log.Log,
	})
	ltxp, ok := txp.(*netxlite.HTTPTransportLogger)
	if !ok {
		t.Fatal("not the transport we expected")
	}
	if ltxp.Logger != log.Log {
		t.Fatal("not the logger we expected")
	}
	if _, ok := ltxp.HTTPTransport.(*netxlite.HTTPTransportWrapper); !ok {
		t.Fatal("not the transport we expected")
	}
}

func TestNewWithSaver(t *testing.T) {
	saver := new(tracex.Saver)
	txp := NewHTTPTransport(Config{
		HTTPSaver: saver,
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
	if _, ok := stxptxp.HTTPTransport.(*netxlite.HTTPTransportWrapper); !ok {
		t.Fatal("not the transport we expected")
	}
}

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

func TestNewDNSClientSystemResolver(t *testing.T) {
	dnsclient, err := NewDNSClient(
		Config{}, "system:///")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := dnsclient.(*netxlite.ResolverSystemDoNotInstantiate); !ok {
		t.Fatal("not the resolver we expected")
	}
	dnsclient.CloseIdleConnections()
}

func TestNewDNSClientEmpty(t *testing.T) {
	dnsclient, err := NewDNSClient(
		Config{}, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := dnsclient.(*netxlite.ResolverSystemDoNotInstantiate); !ok {
		t.Fatal("not the resolver we expected")
	}
	dnsclient.CloseIdleConnections()
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
		Config{ResolveSaver: saver}, "doh://cloudflare")
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
		Config{ResolveSaver: saver}, "udp://8.8.8.8:53")
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
		Config{ResolveSaver: saver}, "tcp://8.8.8.8:53")
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
		Config{ResolveSaver: saver}, "dot://8.8.8.8:53")
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

func TestSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	log.SetLevel(log.DebugLevel)
	counter := bytecounter.New()
	config := Config{
		BogonIsError:        true,
		ByteCounter:         counter,
		CacheResolutions:    true,
		ContextByteCounting: true,
		DialSaver:           &tracex.Saver{},
		HTTPSaver:           &tracex.Saver{},
		Logger:              log.Log,
		ReadWriteSaver:      &tracex.Saver{},
		ResolveSaver:        &tracex.Saver{},
		TLSSaver:            &tracex.Saver{},
	}
	txp := NewHTTPTransport(config)
	client := &http.Client{Transport: txp}
	resp, err := client.Get("https://www.google.com")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = netxlite.ReadAllContext(context.Background(), resp.Body); err != nil {
		t.Fatal(err)
	}
	if err = resp.Body.Close(); err != nil {
		t.Fatal(err)
	}
	if counter.Sent.Load() <= 0 {
		t.Fatal("no bytes sent?!")
	}
	if counter.Received.Load() <= 0 {
		t.Fatal("no bytes received?!")
	}
	if ev := config.DialSaver.Read(); len(ev) <= 0 {
		t.Fatal("no dial events?!")
	}
	if ev := config.HTTPSaver.Read(); len(ev) <= 0 {
		t.Fatal("no HTTP events?!")
	}
	if ev := config.ReadWriteSaver.Read(); len(ev) <= 0 {
		t.Fatal("no R/W events?!")
	}
	if ev := config.ResolveSaver.Read(); len(ev) <= 0 {
		t.Fatal("no resolver events?!")
	}
	if ev := config.TLSSaver.Read(); len(ev) <= 0 {
		t.Fatal("no TLS events?!")
	}
}

func TestBogonResolutionNotBroken(t *testing.T) {
	saver := new(tracex.Saver)
	r := NewResolver(Config{
		BogonIsError: true,
		DNSCache: map[string][]string{
			"www.google.com": {"127.0.0.1"},
		},
		ResolveSaver: saver,
		Logger:       log.Log,
	})
	addrs, err := r.LookupHost(context.Background(), "www.google.com")
	if !errors.Is(err, netxlite.ErrDNSBogon) {
		t.Fatal("not the error we expected")
	}
	if err.Error() != netxlite.FailureDNSBogonError {
		t.Fatal("error not correctly wrapped")
	}
	if len(addrs) > 0 {
		t.Fatal("expected no addresses here")
	}
}
