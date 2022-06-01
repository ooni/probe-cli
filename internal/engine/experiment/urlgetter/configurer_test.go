package urlgetter_test

import (
	"crypto/tls"
	"errors"
	"net/url"
	"strings"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine/experiment/urlgetter"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/tracex"
)

func TestConfigurerNewConfigurationVanilla(t *testing.T) {
	saver := new(tracex.Saver)
	configurer := urlgetter.Configurer{
		Logger: log.Log,
		Saver:  saver,
	}
	configuration, err := configurer.NewConfiguration()
	if err != nil {
		t.Fatal(err)
	}
	defer configuration.CloseIdleConnections()
	if configuration.HTTPConfig.BogonIsError != false {
		t.Fatal("not the BogonIsError we expected")
	}
	if configuration.HTTPConfig.CacheResolutions != true {
		t.Fatal("not the CacheResolutions we expected")
	}
	if configuration.HTTPConfig.ContextByteCounting != true {
		t.Fatal("not the ContextByteCounting we expected")
	}
	if configuration.HTTPConfig.DialSaver != saver {
		t.Fatal("not the DialSaver we expected")
	}
	if configuration.HTTPConfig.HTTPSaver != saver {
		t.Fatal("not the HTTPSaver we expected")
	}
	if configuration.HTTPConfig.Logger != log.Log {
		t.Fatal("not the Logger we expected")
	}
	if configuration.HTTPConfig.ReadWriteSaver != saver {
		t.Fatal("not the ReadWriteSaver we expected")
	}
	if configuration.HTTPConfig.ResolveSaver != saver {
		t.Fatal("not the ResolveSaver we expected")
	}
	if configuration.HTTPConfig.TLSSaver != saver {
		t.Fatal("not the TLSSaver we expected")
	}
	if configuration.HTTPConfig.BaseResolver == nil {
		t.Fatal("not the BaseResolver we expected")
	}
	if len(configuration.HTTPConfig.TLSConfig.NextProtos) != 2 {
		t.Fatal("not the TLSConfig we expected")
	}
	if configuration.HTTPConfig.TLSConfig.NextProtos[0] != "h2" {
		t.Fatal("not the TLSConfig we expected")
	}
	if configuration.HTTPConfig.TLSConfig.NextProtos[1] != "http/1.1" {
		t.Fatal("not the TLSConfig we expected")
	}
	if configuration.HTTPConfig.NoTLSVerify == true {
		t.Fatal("not the NoTLSVerify we expected")
	}
	if configuration.HTTPConfig.ProxyURL != nil {
		t.Fatal("not the ProxyURL we expected")
	}
}

func TestConfigurerNewConfigurationResolverDNSOverHTTPSPowerdns(t *testing.T) {
	saver := new(tracex.Saver)
	configurer := urlgetter.Configurer{
		Config: urlgetter.Config{
			ResolverURL: "doh://google",
		},
		Logger: log.Log,
		Saver:  saver,
	}
	configuration, err := configurer.NewConfiguration()
	if err != nil {
		t.Fatal(err)
	}
	defer configuration.CloseIdleConnections()
	if configuration.HTTPConfig.BogonIsError != false {
		t.Fatal("not the BogonIsError we expected")
	}
	if configuration.HTTPConfig.CacheResolutions != true {
		t.Fatal("not the CacheResolutions we expected")
	}
	if configuration.HTTPConfig.ContextByteCounting != true {
		t.Fatal("not the ContextByteCounting we expected")
	}
	if configuration.HTTPConfig.DialSaver != saver {
		t.Fatal("not the DialSaver we expected")
	}
	if configuration.HTTPConfig.HTTPSaver != saver {
		t.Fatal("not the HTTPSaver we expected")
	}
	if configuration.HTTPConfig.Logger != log.Log {
		t.Fatal("not the Logger we expected")
	}
	if configuration.HTTPConfig.ReadWriteSaver != saver {
		t.Fatal("not the ReadWriteSaver we expected")
	}
	if configuration.HTTPConfig.ResolveSaver != saver {
		t.Fatal("not the ResolveSaver we expected")
	}
	if configuration.HTTPConfig.TLSSaver != saver {
		t.Fatal("not the TLSSaver we expected")
	}
	if configuration.HTTPConfig.BaseResolver == nil {
		t.Fatal("not the BaseResolver we expected")
	}
	sr, ok := configuration.HTTPConfig.BaseResolver.(*netxlite.SerialResolver)
	if !ok {
		t.Fatal("not the resolver we expected")
	}
	stxp, ok := sr.Txp.(*tracex.DNSTransportSaver)
	if !ok {
		t.Fatal("not the DNS transport we expected")
	}
	dohtxp, ok := stxp.DNSTransport.(*netxlite.DNSOverHTTPSTransport)
	if !ok {
		t.Fatal("not the DNS transport we expected")
	}
	if dohtxp.URL != "https://dns.google/dns-query" {
		t.Fatal("not the DoH URL we expected")
	}
	if len(configuration.HTTPConfig.TLSConfig.NextProtos) != 2 {
		t.Fatal("not the TLSConfig we expected")
	}
	if configuration.HTTPConfig.TLSConfig.NextProtos[0] != "h2" {
		t.Fatal("not the TLSConfig we expected")
	}
	if configuration.HTTPConfig.TLSConfig.NextProtos[1] != "http/1.1" {
		t.Fatal("not the TLSConfig we expected")
	}
	if configuration.HTTPConfig.NoTLSVerify == true {
		t.Fatal("not the NoTLSVerify we expected")
	}
	if configuration.HTTPConfig.ProxyURL != nil {
		t.Fatal("not the ProxyURL we expected")
	}
}

func TestConfigurerNewConfigurationResolverDNSOverHTTPSGoogle(t *testing.T) {
	saver := new(tracex.Saver)
	configurer := urlgetter.Configurer{
		Config: urlgetter.Config{
			ResolverURL: "doh://google",
		},
		Logger: log.Log,
		Saver:  saver,
	}
	configuration, err := configurer.NewConfiguration()
	if err != nil {
		t.Fatal(err)
	}
	defer configuration.CloseIdleConnections()
	if configuration.HTTPConfig.BogonIsError != false {
		t.Fatal("not the BogonIsError we expected")
	}
	if configuration.HTTPConfig.CacheResolutions != true {
		t.Fatal("not the CacheResolutions we expected")
	}
	if configuration.HTTPConfig.ContextByteCounting != true {
		t.Fatal("not the ContextByteCounting we expected")
	}
	if configuration.HTTPConfig.DialSaver != saver {
		t.Fatal("not the DialSaver we expected")
	}
	if configuration.HTTPConfig.HTTPSaver != saver {
		t.Fatal("not the HTTPSaver we expected")
	}
	if configuration.HTTPConfig.Logger != log.Log {
		t.Fatal("not the Logger we expected")
	}
	if configuration.HTTPConfig.ReadWriteSaver != saver {
		t.Fatal("not the ReadWriteSaver we expected")
	}
	if configuration.HTTPConfig.ResolveSaver != saver {
		t.Fatal("not the ResolveSaver we expected")
	}
	if configuration.HTTPConfig.TLSSaver != saver {
		t.Fatal("not the TLSSaver we expected")
	}
	if configuration.HTTPConfig.BaseResolver == nil {
		t.Fatal("not the BaseResolver we expected")
	}
	sr, ok := configuration.HTTPConfig.BaseResolver.(*netxlite.SerialResolver)
	if !ok {
		t.Fatal("not the resolver we expected")
	}
	stxp, ok := sr.Txp.(*tracex.DNSTransportSaver)
	if !ok {
		t.Fatal("not the DNS transport we expected")
	}
	dohtxp, ok := stxp.DNSTransport.(*netxlite.DNSOverHTTPSTransport)
	if !ok {
		t.Fatal("not the DNS transport we expected")
	}
	if dohtxp.URL != "https://dns.google/dns-query" {
		t.Fatal("not the DoH URL we expected")
	}
	if len(configuration.HTTPConfig.TLSConfig.NextProtos) != 2 {
		t.Fatal("not the TLSConfig we expected")
	}
	if configuration.HTTPConfig.TLSConfig.NextProtos[0] != "h2" {
		t.Fatal("not the TLSConfig we expected")
	}
	if configuration.HTTPConfig.TLSConfig.NextProtos[1] != "http/1.1" {
		t.Fatal("not the TLSConfig we expected")
	}
	if configuration.HTTPConfig.NoTLSVerify == true {
		t.Fatal("not the NoTLSVerify we expected")
	}
	if configuration.HTTPConfig.ProxyURL != nil {
		t.Fatal("not the ProxyURL we expected")
	}
}

func TestConfigurerNewConfigurationResolverDNSOverHTTPSCloudflare(t *testing.T) {
	saver := new(tracex.Saver)
	configurer := urlgetter.Configurer{
		Config: urlgetter.Config{
			ResolverURL: "doh://cloudflare",
		},
		Logger: log.Log,
		Saver:  saver,
	}
	configuration, err := configurer.NewConfiguration()
	if err != nil {
		t.Fatal(err)
	}
	defer configuration.CloseIdleConnections()
	if configuration.HTTPConfig.BogonIsError != false {
		t.Fatal("not the BogonIsError we expected")
	}
	if configuration.HTTPConfig.CacheResolutions != true {
		t.Fatal("not the CacheResolutions we expected")
	}
	if configuration.HTTPConfig.ContextByteCounting != true {
		t.Fatal("not the ContextByteCounting we expected")
	}
	if configuration.HTTPConfig.DialSaver != saver {
		t.Fatal("not the DialSaver we expected")
	}
	if configuration.HTTPConfig.HTTPSaver != saver {
		t.Fatal("not the HTTPSaver we expected")
	}
	if configuration.HTTPConfig.Logger != log.Log {
		t.Fatal("not the Logger we expected")
	}
	if configuration.HTTPConfig.ReadWriteSaver != saver {
		t.Fatal("not the ReadWriteSaver we expected")
	}
	if configuration.HTTPConfig.ResolveSaver != saver {
		t.Fatal("not the ResolveSaver we expected")
	}
	if configuration.HTTPConfig.TLSSaver != saver {
		t.Fatal("not the TLSSaver we expected")
	}
	if configuration.HTTPConfig.BaseResolver == nil {
		t.Fatal("not the BaseResolver we expected")
	}
	sr, ok := configuration.HTTPConfig.BaseResolver.(*netxlite.SerialResolver)
	if !ok {
		t.Fatal("not the resolver we expected")
	}
	stxp, ok := sr.Txp.(*tracex.DNSTransportSaver)
	if !ok {
		t.Fatal("not the DNS transport we expected")
	}
	dohtxp, ok := stxp.DNSTransport.(*netxlite.DNSOverHTTPSTransport)
	if !ok {
		t.Fatal("not the DNS transport we expected")
	}
	if dohtxp.URL != "https://cloudflare-dns.com/dns-query" {
		t.Fatal("not the DoH URL we expected")
	}
	if len(configuration.HTTPConfig.TLSConfig.NextProtos) != 2 {
		t.Fatal("not the TLSConfig we expected")
	}
	if configuration.HTTPConfig.TLSConfig.NextProtos[0] != "h2" {
		t.Fatal("not the TLSConfig we expected")
	}
	if configuration.HTTPConfig.TLSConfig.NextProtos[1] != "http/1.1" {
		t.Fatal("not the TLSConfig we expected")
	}
	if configuration.HTTPConfig.NoTLSVerify == true {
		t.Fatal("not the NoTLSVerify we expected")
	}
	if configuration.HTTPConfig.ProxyURL != nil {
		t.Fatal("not the ProxyURL we expected")
	}
}

func TestConfigurerNewConfigurationResolverUDP(t *testing.T) {
	saver := new(tracex.Saver)
	configurer := urlgetter.Configurer{
		Config: urlgetter.Config{
			ResolverURL: "udp://8.8.8.8:53",
		},
		Logger: log.Log,
		Saver:  saver,
	}
	configuration, err := configurer.NewConfiguration()
	if err != nil {
		t.Fatal(err)
	}
	defer configuration.CloseIdleConnections()
	if configuration.HTTPConfig.BogonIsError != false {
		t.Fatal("not the BogonIsError we expected")
	}
	if configuration.HTTPConfig.CacheResolutions != true {
		t.Fatal("not the CacheResolutions we expected")
	}
	if configuration.HTTPConfig.ContextByteCounting != true {
		t.Fatal("not the ContextByteCounting we expected")
	}
	if configuration.HTTPConfig.DialSaver != saver {
		t.Fatal("not the DialSaver we expected")
	}
	if configuration.HTTPConfig.HTTPSaver != saver {
		t.Fatal("not the HTTPSaver we expected")
	}
	if configuration.HTTPConfig.Logger != log.Log {
		t.Fatal("not the Logger we expected")
	}
	if configuration.HTTPConfig.ReadWriteSaver != saver {
		t.Fatal("not the ReadWriteSaver we expected")
	}
	if configuration.HTTPConfig.ResolveSaver != saver {
		t.Fatal("not the ResolveSaver we expected")
	}
	if configuration.HTTPConfig.TLSSaver != saver {
		t.Fatal("not the TLSSaver we expected")
	}
	if configuration.HTTPConfig.BaseResolver == nil {
		t.Fatal("not the BaseResolver we expected")
	}
	sr, ok := configuration.HTTPConfig.BaseResolver.(*netxlite.SerialResolver)
	if !ok {
		t.Fatal("not the resolver we expected")
	}
	stxp, ok := sr.Txp.(*tracex.DNSTransportSaver)
	if !ok {
		t.Fatal("not the DNS transport we expected")
	}
	udptxp, ok := stxp.DNSTransport.(*netxlite.DNSOverUDPTransport)
	if !ok {
		t.Fatal("not the DNS transport we expected")
	}
	if udptxp.Address() != "8.8.8.8:53" {
		t.Fatal("not the DoH URL we expected")
	}
	if len(configuration.HTTPConfig.TLSConfig.NextProtos) != 2 {
		t.Fatal("not the TLSConfig we expected")
	}
	if configuration.HTTPConfig.TLSConfig.NextProtos[0] != "h2" {
		t.Fatal("not the TLSConfig we expected")
	}
	if configuration.HTTPConfig.TLSConfig.NextProtos[1] != "http/1.1" {
		t.Fatal("not the TLSConfig we expected")
	}
	if configuration.HTTPConfig.NoTLSVerify == true {
		t.Fatal("not the NoTLSVerify we expected")
	}
	if configuration.HTTPConfig.ProxyURL != nil {
		t.Fatal("not the ProxyURL we expected")
	}
}

func TestConfigurerNewConfigurationDNSCacheInvalidString(t *testing.T) {
	saver := new(tracex.Saver)
	configurer := urlgetter.Configurer{
		Config: urlgetter.Config{
			DNSCache: "a",
		},
		Logger: log.Log,
		Saver:  saver,
	}
	_, err := configurer.NewConfiguration()
	if err == nil || !strings.HasSuffix(err.Error(), "invalid DNSCache string") {
		t.Fatal("not the error we expected")
	}
}

func TestConfigurerNewConfigurationDNSCacheNotDomain(t *testing.T) {
	saver := new(tracex.Saver)
	configurer := urlgetter.Configurer{
		Config: urlgetter.Config{
			DNSCache: "b b",
		},
		Logger: log.Log,
		Saver:  saver,
	}
	_, err := configurer.NewConfiguration()
	if err == nil || !strings.HasSuffix(err.Error(), "invalid domain in DNSCache") {
		t.Fatal("not the error we expected")
	}
}

func TestConfigurerNewConfigurationDNSCacheNotIP(t *testing.T) {
	saver := new(tracex.Saver)
	configurer := urlgetter.Configurer{
		Config: urlgetter.Config{
			DNSCache: "x.org b",
		},
		Logger: log.Log,
		Saver:  saver,
	}
	_, err := configurer.NewConfiguration()
	if err == nil || !strings.HasSuffix(err.Error(), "invalid IP in DNSCache") {
		t.Fatal("not the error we expected")
	}
}

func TestConfigurerNewConfigurationDNSCacheGood(t *testing.T) {
	saver := new(tracex.Saver)
	configurer := urlgetter.Configurer{
		Config: urlgetter.Config{
			DNSCache: "dns.google.com 8.8.8.8 8.8.4.4",
		},
		Logger: log.Log,
		Saver:  saver,
	}
	configuration, err := configurer.NewConfiguration()
	if err != nil {
		t.Fatal(err)
	}
	if len(configuration.HTTPConfig.DNSCache) != 1 {
		t.Fatal("invalid number of entries in DNSCache")
	}
	if len(configuration.HTTPConfig.DNSCache["dns.google.com"]) != 2 {
		t.Fatal("invalid number of IPs saved in DNSCache")
	}
	if configuration.HTTPConfig.DNSCache["dns.google.com"][0] != "8.8.8.8" {
		t.Fatal("invalid IPs saved in DNSCache")
	}
	if configuration.HTTPConfig.DNSCache["dns.google.com"][1] != "8.8.4.4" {
		t.Fatal("invalid IPs saved in DNSCache")
	}
}

func TestConfigurerNewConfigurationResolverInvalidURL(t *testing.T) {
	saver := new(tracex.Saver)
	configurer := urlgetter.Configurer{
		Config: urlgetter.Config{
			ResolverURL: "\t",
		},
		Logger: log.Log,
		Saver:  saver,
	}
	_, err := configurer.NewConfiguration()
	if err == nil || !strings.HasSuffix(err.Error(), "invalid control character in URL") {
		t.Fatal("not the error we expected")
	}
}

func TestConfigurerNewConfigurationResolverInvalidURLScheme(t *testing.T) {
	saver := new(tracex.Saver)
	configurer := urlgetter.Configurer{
		Config: urlgetter.Config{
			ResolverURL: "antani://8.8.8.8:53",
		},
		Logger: log.Log,
		Saver:  saver,
	}
	_, err := configurer.NewConfiguration()
	if err == nil || !strings.HasSuffix(err.Error(), "unsupported resolver scheme") {
		t.Fatal("not the error we expected")
	}
}

func TestConfigurerNewConfigurationTLSServerName(t *testing.T) {
	saver := new(tracex.Saver)
	configurer := urlgetter.Configurer{
		Config: urlgetter.Config{
			TLSServerName: "www.x.org",
		},
		Logger: log.Log,
		Saver:  saver,
	}
	configuration, err := configurer.NewConfiguration()
	if err != nil {
		t.Fatal(err)
	}
	if configuration.HTTPConfig.TLSConfig.ServerName != "www.x.org" {
		t.Fatal("invalid ServerName")
	}
	if len(configuration.HTTPConfig.TLSConfig.NextProtos) != 2 {
		t.Fatal("invalid len(NextProtos)")
	}
	if configuration.HTTPConfig.TLSConfig.NextProtos[0] != "h2" {
		t.Fatal("invalid NextProtos[0]")
	}
	if configuration.HTTPConfig.TLSConfig.NextProtos[1] != "http/1.1" {
		t.Fatal("invalid NextProtos[1]")
	}
}

func TestConfigurerNewConfigurationNoTLSVerify(t *testing.T) {
	saver := new(tracex.Saver)
	configurer := urlgetter.Configurer{
		Config: urlgetter.Config{
			NoTLSVerify: true,
		},
		Logger: log.Log,
		Saver:  saver,
	}
	configuration, err := configurer.NewConfiguration()
	if err != nil {
		t.Fatal(err)
	}
	if configuration.HTTPConfig.NoTLSVerify != true {
		t.Fatal("not the NoTLSVerify we expected")
	}
}

func TestConfigurerNewConfigurationTLSv1(t *testing.T) {
	saver := new(tracex.Saver)
	configurer := urlgetter.Configurer{
		Config: urlgetter.Config{
			TLSVersion: "TLSv1",
		},
		Logger: log.Log,
		Saver:  saver,
	}
	configuration, err := configurer.NewConfiguration()
	if err != nil {
		t.Fatal(err)
	}
	if len(configuration.HTTPConfig.TLSConfig.NextProtos) != 2 {
		t.Fatal("invalid len(NextProtos)")
	}
	if configuration.HTTPConfig.TLSConfig.NextProtos[0] != "h2" {
		t.Fatal("invalid NextProtos[0]")
	}
	if configuration.HTTPConfig.TLSConfig.NextProtos[1] != "http/1.1" {
		t.Fatal("invalid NextProtos[1]")
	}
	if configuration.HTTPConfig.TLSConfig.MinVersion != tls.VersionTLS10 {
		t.Fatal("invalid MinVersion")
	}
	if configuration.HTTPConfig.TLSConfig.MaxVersion != tls.VersionTLS10 {
		t.Fatal("invalid MaxVersion")
	}
}

func TestConfigurerNewConfigurationTLSv1dot0(t *testing.T) {
	saver := new(tracex.Saver)
	configurer := urlgetter.Configurer{
		Config: urlgetter.Config{
			TLSVersion: "TLSv1.0",
		},
		Logger: log.Log,
		Saver:  saver,
	}
	configuration, err := configurer.NewConfiguration()
	if err != nil {
		t.Fatal(err)
	}
	if len(configuration.HTTPConfig.TLSConfig.NextProtos) != 2 {
		t.Fatal("invalid len(NextProtos)")
	}
	if configuration.HTTPConfig.TLSConfig.NextProtos[0] != "h2" {
		t.Fatal("invalid NextProtos[0]")
	}
	if configuration.HTTPConfig.TLSConfig.NextProtos[1] != "http/1.1" {
		t.Fatal("invalid NextProtos[1]")
	}
	if configuration.HTTPConfig.TLSConfig.MinVersion != tls.VersionTLS10 {
		t.Fatal("invalid MinVersion")
	}
	if configuration.HTTPConfig.TLSConfig.MaxVersion != tls.VersionTLS10 {
		t.Fatal("invalid MaxVersion")
	}
}

func TestConfigurerNewConfigurationTLSv1dot1(t *testing.T) {
	saver := new(tracex.Saver)
	configurer := urlgetter.Configurer{
		Config: urlgetter.Config{
			TLSVersion: "TLSv1.1",
		},
		Logger: log.Log,
		Saver:  saver,
	}
	configuration, err := configurer.NewConfiguration()
	if err != nil {
		t.Fatal(err)
	}
	if len(configuration.HTTPConfig.TLSConfig.NextProtos) != 2 {
		t.Fatal("invalid len(NextProtos)")
	}
	if configuration.HTTPConfig.TLSConfig.NextProtos[0] != "h2" {
		t.Fatal("invalid NextProtos[0]")
	}
	if configuration.HTTPConfig.TLSConfig.NextProtos[1] != "http/1.1" {
		t.Fatal("invalid NextProtos[1]")
	}
	if configuration.HTTPConfig.TLSConfig.MinVersion != tls.VersionTLS11 {
		t.Fatal("invalid MinVersion")
	}
	if configuration.HTTPConfig.TLSConfig.MaxVersion != tls.VersionTLS11 {
		t.Fatal("invalid MaxVersion")
	}
}

func TestConfigurerNewConfigurationTLSv1dot2(t *testing.T) {
	saver := new(tracex.Saver)
	configurer := urlgetter.Configurer{
		Config: urlgetter.Config{
			TLSVersion: "TLSv1.2",
		},
		Logger: log.Log,
		Saver:  saver,
	}
	configuration, err := configurer.NewConfiguration()
	if err != nil {
		t.Fatal(err)
	}
	if len(configuration.HTTPConfig.TLSConfig.NextProtos) != 2 {
		t.Fatal("invalid len(NextProtos)")
	}
	if configuration.HTTPConfig.TLSConfig.NextProtos[0] != "h2" {
		t.Fatal("invalid NextProtos[0]")
	}
	if configuration.HTTPConfig.TLSConfig.NextProtos[1] != "http/1.1" {
		t.Fatal("invalid NextProtos[1]")
	}
	if configuration.HTTPConfig.TLSConfig.MinVersion != tls.VersionTLS12 {
		t.Fatal("invalid MinVersion")
	}
	if configuration.HTTPConfig.TLSConfig.MaxVersion != tls.VersionTLS12 {
		t.Fatal("invalid MaxVersion")
	}
}

func TestConfigurerNewConfigurationTLSv1dot3(t *testing.T) {
	saver := new(tracex.Saver)
	configurer := urlgetter.Configurer{
		Config: urlgetter.Config{
			TLSVersion: "TLSv1.3",
		},
		Logger: log.Log,
		Saver:  saver,
	}
	configuration, err := configurer.NewConfiguration()
	if err != nil {
		t.Fatal(err)
	}
	if len(configuration.HTTPConfig.TLSConfig.NextProtos) != 2 {
		t.Fatal("invalid len(NextProtos)")
	}
	if configuration.HTTPConfig.TLSConfig.NextProtos[0] != "h2" {
		t.Fatal("invalid NextProtos[0]")
	}
	if configuration.HTTPConfig.TLSConfig.NextProtos[1] != "http/1.1" {
		t.Fatal("invalid NextProtos[1]")
	}
	if configuration.HTTPConfig.TLSConfig.MinVersion != tls.VersionTLS13 {
		t.Fatal("invalid MinVersion")
	}
	if configuration.HTTPConfig.TLSConfig.MaxVersion != tls.VersionTLS13 {
		t.Fatal("invalid MaxVersion")
	}
}

func TestConfigurerNewConfigurationTLSvDefault(t *testing.T) {
	saver := new(tracex.Saver)
	configurer := urlgetter.Configurer{
		Config: urlgetter.Config{},
		Logger: log.Log,
		Saver:  saver,
	}
	configuration, err := configurer.NewConfiguration()
	if err != nil {
		t.Fatal(err)
	}
	if len(configuration.HTTPConfig.TLSConfig.NextProtos) != 2 {
		t.Fatal("invalid len(NextProtos)")
	}
	if configuration.HTTPConfig.TLSConfig.NextProtos[0] != "h2" {
		t.Fatal("invalid NextProtos[0]")
	}
	if configuration.HTTPConfig.TLSConfig.NextProtos[1] != "http/1.1" {
		t.Fatal("invalid NextProtos[1]")
	}
	if configuration.HTTPConfig.TLSConfig.MinVersion != 0 {
		t.Fatal("invalid MinVersion")
	}
	if configuration.HTTPConfig.TLSConfig.MaxVersion != 0 {
		t.Fatal("invalid MaxVersion")
	}
}

func TestConfigurerNewConfigurationTLSvInvalid(t *testing.T) {
	saver := new(tracex.Saver)
	configurer := urlgetter.Configurer{
		Config: urlgetter.Config{
			TLSVersion: "SSLv3",
		},
		Logger: log.Log,
		Saver:  saver,
	}
	_, err := configurer.NewConfiguration()
	if !errors.Is(err, netxlite.ErrInvalidTLSVersion) {
		t.Fatalf("not the error we expected: %+v", err)
	}
}

func TestConfigurerNewConfigurationProxyURL(t *testing.T) {
	URL, _ := url.Parse("socks5://127.0.0.1:9050")
	saver := new(tracex.Saver)
	configurer := urlgetter.Configurer{
		Logger:   log.Log,
		Saver:    saver,
		ProxyURL: URL,
	}
	configuration, err := configurer.NewConfiguration()
	if err != nil {
		t.Fatal(err)
	}
	if configuration.HTTPConfig.ProxyURL != URL {
		t.Fatal("invalid ProxyURL")
	}
}
