// Jafar is a censorship simulation tool used for testing OONI.
package main

import (
	"encoding/pem"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"

	"github.com/apex/log"
	"github.com/apex/log/handlers/cli"
	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/cmd/jafar/badproxy"
	"github.com/ooni/probe-cli/v3/internal/cmd/jafar/flagx"
	"github.com/ooni/probe-cli/v3/internal/cmd/jafar/httpproxy"
	"github.com/ooni/probe-cli/v3/internal/cmd/jafar/iptables"
	"github.com/ooni/probe-cli/v3/internal/cmd/jafar/resolver"
	"github.com/ooni/probe-cli/v3/internal/cmd/jafar/shellx"
	"github.com/ooni/probe-cli/v3/internal/cmd/jafar/tlsproxy"
	"github.com/ooni/probe-cli/v3/internal/cmd/jafar/uncensored"
	"github.com/ooni/probe-cli/v3/internal/engine/runtimex"
)

var (
	badProxyAddress     *string
	badProxyAddressTLS  *string
	badProxyTLSOutputCA *string

	dnsProxyAddress *string
	dnsProxyBlock   flagx.StringArray
	dnsProxyHijack  flagx.StringArray
	dnsProxyIgnore  flagx.StringArray

	httpProxyAddress *string
	httpProxyBlock   flagx.StringArray

	iptablesDropIP          flagx.StringArray
	iptablesDropKeywordHex  flagx.StringArray
	iptablesDropKeyword     flagx.StringArray
	iptablesHijackDNSTo     *string
	iptablesHijackHTTPSTo   *string
	iptablesHijackHTTPTo    *string
	iptablesResetIP         flagx.StringArray
	iptablesResetKeywordHex flagx.StringArray
	iptablesResetKeyword    flagx.StringArray

	mainCh      chan os.Signal
	mainCommand *string
	mainUser    *string

	tag *string

	tlsProxyAddress *string
	tlsProxyBlock   flagx.StringArray

	uncensoredResolverURL *string
)

func init() {
	// badProxy
	badProxyAddress = flag.String(
		"bad-proxy-address", "127.0.0.1:7117",
		"Address where to listen for TCP connections",
	)
	badProxyAddressTLS = flag.String(
		"bad-proxy-address-tls", "127.0.0.1:4114",
		"Address where to listen for TLS connections",
	)
	badProxyTLSOutputCA = flag.String(
		"bad-proxy-tls-output-ca", "badproxy.pem",
		"File where to write the CA used by the bad proxy",
	)

	// dnsProxy
	dnsProxyAddress = flag.String(
		"dns-proxy-address", "127.0.0.1:53",
		"Address where the DNS proxy should listen",
	)
	flag.Var(
		&dnsProxyBlock, "dns-proxy-block",
		"Register keyword triggering NXDOMAIN censorship",
	)
	flag.Var(
		&dnsProxyHijack, "dns-proxy-hijack",
		"Register keyword triggering redirection to 127.0.0.1",
	)
	flag.Var(
		&dnsProxyIgnore, "dns-proxy-ignore",
		"Register keyword causing the proxy to ignore the query",
	)

	// httpProxy
	httpProxyAddress = flag.String(
		"http-proxy-address", "127.0.0.1:80",
		"Address where the HTTP proxy should listen",
	)
	flag.Var(
		&httpProxyBlock, "http-proxy-block",
		"Register keyword triggering HTTP 451 censorship",
	)

	// iptables
	flag.Var(
		&iptablesDropIP, "iptables-drop-ip",
		"Drop traffic to the specified IP address",
	)
	flag.Var(
		&iptablesDropKeywordHex, "iptables-drop-keyword-hex",
		"Drop traffic containing the specified keyword in hex",
	)
	flag.Var(
		&iptablesDropKeyword, "iptables-drop-keyword",
		"Drop traffic containing the specified keyword",
	)
	iptablesHijackDNSTo = flag.String(
		"iptables-hijack-dns-to", "",
		"Hijack all DNS UDP traffic to the specified endpoint",
	)
	iptablesHijackHTTPSTo = flag.String(
		"iptables-hijack-https-to", "",
		"Hijack all HTTPS traffic to the specified endpoint",
	)
	iptablesHijackHTTPTo = flag.String(
		"iptables-hijack-http-to", "",
		"Hijack all HTTP traffic to the specified endpoint",
	)
	flag.Var(
		&iptablesResetIP, "iptables-reset-ip",
		"Reset TCP/IP traffic to the specified IP address",
	)
	flag.Var(
		&iptablesResetKeywordHex, "iptables-reset-keyword-hex",
		"Reset TCP/IP traffic containing the specified keyword in hex",
	)
	flag.Var(
		&iptablesResetKeyword, "iptables-reset-keyword",
		"Reset TCP/IP traffic containing the specified keyword",
	)

	// main
	mainCh = make(chan os.Signal, 1)
	signal.Notify(
		mainCh, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT,
	)
	mainCommand = flag.String("main-command", "", "Optional command to execute")
	mainUser = flag.String("main-user", "nobody", "Run command as user")

	// tag
	tag = flag.String("tag", "", "Add tag to a specific run")

	// tlsProxy
	tlsProxyAddress = flag.String(
		"tls-proxy-address", "127.0.0.1:443",
		"Address where the HTTP proxy should listen",
	)
	flag.Var(
		&tlsProxyBlock, "tls-proxy-block",
		"Register keyword triggering TLS censorship",
	)

	// uncensored
	uncensoredResolverURL = flag.String(
		"uncensored-resolver-url", "dot://1.1.1.1:853",
		"URL of an hopefully uncensored resolver",
	)
}

func badProxyStart() net.Listener {
	proxy := badproxy.NewCensoringProxy()
	listener, err := proxy.Start(*badProxyAddress)
	runtimex.PanicOnError(err, "proxy.Start failed")
	return listener
}

func badProxyStartTLS() net.Listener {
	proxy := badproxy.NewCensoringProxy()
	listener, cert, err := proxy.StartTLS(*badProxyAddressTLS)
	runtimex.PanicOnError(err, "proxy.StartTLS failed")
	err = ioutil.WriteFile(*badProxyTLSOutputCA, pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert.Raw,
	}), 0644)
	runtimex.PanicOnError(err, "ioutil.WriteFile failed")
	return listener
}

func dnsProxyStart(uncensored *uncensored.Client) *dns.Server {
	proxy := resolver.NewCensoringResolver(
		dnsProxyBlock, dnsProxyHijack, dnsProxyIgnore, uncensored,
	)
	server, err := proxy.Start(*dnsProxyAddress)
	runtimex.PanicOnError(err, "proxy.Start failed")
	return server
}

func httpProxyStart(uncensored *uncensored.Client) *http.Server {
	proxy := httpproxy.NewCensoringProxy(httpProxyBlock, uncensored)
	server, _, err := proxy.Start(*httpProxyAddress)
	runtimex.PanicOnError(err, "proxy.Start failed")
	return server
}

func iptablesStart() *iptables.CensoringPolicy {
	policy := iptables.NewCensoringPolicy()
	// For robustness waive the policy so we start afresh
	policy.Waive()
	policy.DropIPs = iptablesDropIP
	policy.DropKeywordsHex = iptablesDropKeywordHex
	policy.DropKeywords = iptablesDropKeyword
	policy.HijackDNSAddress = *iptablesHijackDNSTo
	policy.HijackHTTPSAddress = *iptablesHijackHTTPSTo
	policy.HijackHTTPAddress = *iptablesHijackHTTPTo
	policy.ResetIPs = iptablesResetIP
	policy.ResetKeywordsHex = iptablesResetKeywordHex
	policy.ResetKeywords = iptablesResetKeyword
	err := policy.Apply()
	runtimex.PanicOnError(err, "policy.Apply failed")
	return policy
}

func tlsProxyStart(uncensored *uncensored.Client) net.Listener {
	proxy := tlsproxy.NewCensoringProxy(tlsProxyBlock, uncensored)
	listener, err := proxy.Start(*tlsProxyAddress)
	runtimex.PanicOnError(err, "proxy.Start failed")
	return listener
}

func newUncensoredClient() *uncensored.Client {
	clnt, err := uncensored.NewClient(*uncensoredResolverURL)
	runtimex.PanicOnError(err, "uncensored.NewClient failed")
	return clnt
}

func mustx(err error, message string, osExit func(int)) {
	if err != nil {
		var (
			exitcode = 1
			exiterr  *exec.ExitError
		)
		if errors.As(err, &exiterr) {
			exitcode = exiterr.ExitCode()
		}
		log.Errorf("%s", message)
		osExit(exitcode)
	}
}

func main() {
	flag.Parse()
	// TODO(bassosimone): we may want a verbose flag
	log.SetLevel(log.InfoLevel)
	log.SetHandler(cli.Default)
	log.Infof("jafar command line: [%s]", strings.Join(os.Args, ", "))
	log.Infof("jafar tag: %s", *tag)
	uncensoredClient := newUncensoredClient()
	defer uncensoredClient.CloseIdleConnections()
	badlistener := badProxyStart()
	defer badlistener.Close()
	badtlslistener := badProxyStartTLS()
	defer badtlslistener.Close()
	dnsproxy := dnsProxyStart(uncensoredClient)
	defer dnsproxy.Shutdown()
	httpproxy := httpProxyStart(uncensoredClient)
	defer httpproxy.Close()
	tlslistener := tlsProxyStart(uncensoredClient)
	defer tlslistener.Close()
	policy := iptablesStart()
	var err error
	if *mainCommand != "" {
		err = shellx.RunCommandline(fmt.Sprintf(
			"sudo -u '%s' -- %s", *mainUser, *mainCommand,
		))
	} else {
		<-mainCh
	}
	policy.Waive()
	mustx(err, "subcommand failed", os.Exit)
}
