package netemx

import (
	"net/http"

	"github.com/apex/log"
	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

const (
	// ScenarioRolePublicDNS means we should create DNS-over-HTTPS and DNS-over-UDP servers.
	ScenarioRolePublicDNS = iota

	// ScenarioRoleWebServer means we should instantiate a webserver using a specific factory.
	ScenarioRoleWebServer

	// ScenarioRoleOONIAPI means we should instantiate the OONI API.
	ScenarioRoleOONIAPI

	// ScenarioRoleUbuntuGeoIP means we should instantiate the Ubuntu geoip service.
	ScenarioRoleUbuntuGeoIP

	// ScenarioRoleOONITestHelper means we should instantiate the oohelperd.
	ScenarioRoleOONITestHelper

	// ScenarioRoleBlockpageServer means we should serve a blockpage using HTTP.
	ScenarioRoleBlockpageServer

	// ScenarioRoleProxy means the host is a transparent proxy.
	ScenarioRoleProxy

	// ScenarioRoleURLShortener means that the host is an URL shortener.
	ScenarioRoleURLShortener

	// ScenarioRoleBadSSL means that the host hosts services to
	// measure against common TLS issues.
	ScenarioRoleBadSSL
)

// ScenarioDomainAddresses describes a domain and address used in a scenario.
type ScenarioDomainAddresses struct {
	// Addresses contains the MANDATORY list of addresses belonging to the domain.
	Addresses []string

	// Domains contains a related set of domains domains (MANDATORY field).
	Domains []string

	// Role is the MANDATORY role of this domain (e.g., ScenarioRoleOONIAPI).
	Role uint64

	// ServerNameMain is the MANDATORY server name to use as common name for X.509 certs.
	ServerNameMain string

	// ServerNameExtras contains OPTIONAL extra names to also configure into the cert.
	ServerNameExtras []string

	// WebServerFactory is the factory to use when Role is ScenarioRoleWebServer.
	WebServerFactory HTTPHandlerFactory
}

// InternetScenario contains the domains and addresses used by [NewInternetScenario].
var InternetScenario = []*ScenarioDomainAddresses{{
	Domains: []string{"api.ooni.io"},
	Addresses: []string{
		AddressApiOONIIo,
	},
	Role:             ScenarioRoleOONIAPI,
	ServerNameMain:   "api.ooni.io",
	ServerNameExtras: []string{},
}, {
	Domains: []string{"geoip.ubuntu.com"},
	Addresses: []string{
		AddressGeoIPUbuntuCom,
	},
	Role:             ScenarioRoleUbuntuGeoIP,
	ServerNameMain:   "geoip.ubuntu.com",
	ServerNameExtras: []string{},
}, {
	Domains: []string{"www.example.com", "example.com", "www.example.org", "example.org"},
	Addresses: []string{
		AddressWwwExampleCom,
	},
	Role:             ScenarioRoleWebServer,
	WebServerFactory: ExampleWebPageHandlerFactory(),
	ServerNameMain:   "www.example.com",
	ServerNameExtras: []string{"example.com", "www.example.org", "example.org"},
}, {
	Domains: []string{"0.th.ooni.org"},
	Addresses: []string{
		AddressZeroThOONIOrg,
	},
	Role:             ScenarioRoleOONITestHelper,
	ServerNameMain:   "0.th.ooni.org",
	ServerNameExtras: []string{},
}, {
	Domains: []string{"1.th.ooni.org"},
	Addresses: []string{
		AddressOneThOONIOrg,
	},
	Role:             ScenarioRoleOONITestHelper,
	ServerNameMain:   "1.th.ooni.org",
	ServerNameExtras: []string{},
}, {
	Domains: []string{"2.th.ooni.org"},
	Addresses: []string{
		AddressTwoThOONIOrg,
	},
	Role:             ScenarioRoleOONITestHelper,
	ServerNameMain:   "2.th.ooni.org",
	ServerNameExtras: []string{},
}, {
	Domains: []string{"3.th.ooni.org"},
	Addresses: []string{
		AddressThreeThOONIOrg,
	},
	Role:             ScenarioRoleOONITestHelper,
	ServerNameMain:   "3.th.ooni.org",
	ServerNameExtras: []string{},
}, {
	Domains: []string{"d33d1gs9kpq1c5.cloudfront.net"},
	Addresses: []string{
		AddressTHCloudfront,
	},
	Role:             ScenarioRoleOONITestHelper,
	ServerNameMain:   "d33d1gs9kpq1c5.cloudfront.net",
	ServerNameExtras: []string{},
}, {
	Domains: []string{"dns.quad9.net"},
	Addresses: []string{
		AddressDNSQuad9Net,
	},
	Role:             ScenarioRolePublicDNS,
	ServerNameMain:   "dns.quad9.net",
	ServerNameExtras: []string{},
}, {
	Domains: []string{"mozilla.cloudflare-dns.com"},
	Addresses: []string{
		AddressMozillaCloudflareDNSCom,
	},
	Role:             ScenarioRolePublicDNS,
	ServerNameMain:   "mozilla.cloudflare-dns.com",
	ServerNameExtras: []string{},
}, {
	Domains: []string{"dns.google", "dns.google.com"},
	Addresses: []string{
		AddressDNSGoogle8844,
		AddressDNSGoogle8888,
	},
	Role:             ScenarioRolePublicDNS,
	ServerNameMain:   "dns.google",
	ServerNameExtras: []string{"dns.google.com"},
}, {
	Domains: []string{},
	Addresses: []string{
		AddressPublicBlockpage,
	},
	Role:             ScenarioRoleBlockpageServer,
	WebServerFactory: BlockpageHandlerFactory(),
	ServerNameMain:   "blockpage.local",
	ServerNameExtras: []string{},
}, {
	Domains: []string{},
	Addresses: []string{
		ISPProxyAddress,
	},
	Role:             ScenarioRoleProxy,
	ServerNameMain:   "proxy.local",
	ServerNameExtras: []string{},
}, {
	Domains: []string{"bit.ly", "bitly.com"},
	Addresses: []string{
		AddressBitly,
	},
	Role:             ScenarioRoleURLShortener,
	ServerNameMain:   "bit.ly",
	ServerNameExtras: []string{"bitly.com"},
}, {
	Domains: []string{
		"wrong.host.badssl.com",
		"untrusted-root.badssl.com",
		"expired.badssl.com",
	},
	Addresses: []string{
		AddressBadSSLCom,
	},
	Role:             ScenarioRoleBadSSL,
	ServerNameMain:   "badssl.com",
	ServerNameExtras: []string{},
}}

// MustNewScenario constructs a complete testing scenario using the domains and IP
// addresses contained by the given [ScenarioDomainAddresses] array.
func MustNewScenario(config []*ScenarioDomainAddresses) *QAEnv {
	var opts []QAEnvOption

	// fill options based on the scenario config
	for _, sad := range config {
		switch sad.Role {
		case ScenarioRolePublicDNS:
			for _, addr := range sad.Addresses {
				opts = append(opts, QAEnvOptionNetStack(
					addr,
					&DNSOverUDPServerFactory{},
					&HTTPSecureServerFactory{
						Factory:          &DNSOverHTTPSHandlerFactory{},
						Ports:            []int{443},
						ServerNameMain:   sad.ServerNameMain,
						ServerNameExtras: sad.ServerNameExtras,
					},
					&HTTP3ServerFactory{
						Factory:          &DNSOverHTTPSHandlerFactory{},
						Ports:            []int{443},
						ServerNameMain:   sad.ServerNameMain,
						ServerNameExtras: sad.ServerNameExtras,
					},
				))
			}

		case ScenarioRoleWebServer:
			for _, addr := range sad.Addresses {
				opts = append(opts, qaEnvOptionNetStack(
					addr,
					&HTTPCleartextServerFactory{
						Factory: sad.WebServerFactory,
						Ports:   []int{80},
					},
					&HTTPSecureServerFactory{
						Factory:          sad.WebServerFactory,
						Ports:            []int{443},
						ServerNameMain:   sad.ServerNameMain,
						ServerNameExtras: sad.ServerNameExtras,
					},
					&HTTP3ServerFactory{
						Factory:          sad.WebServerFactory,
						Ports:            []int{443},
						ServerNameMain:   sad.ServerNameMain,
						ServerNameExtras: sad.ServerNameExtras,
					},
				))
			}

		case ScenarioRoleOONIAPI:
			for _, addr := range sad.Addresses {
				opts = append(opts, QAEnvOptionNetStack(addr, &HTTPSecureServerFactory{
					Factory:          &OOAPIHandlerFactory{},
					Ports:            []int{443},
					ServerNameMain:   sad.ServerNameMain,
					ServerNameExtras: sad.ServerNameExtras,
				}))
			}

		case ScenarioRoleOONITestHelper:
			for _, addr := range sad.Addresses {
				opts = append(opts, QAEnvOptionNetStack(addr, &HTTPSecureServerFactory{
					Factory:          &OOHelperDFactory{},
					Ports:            []int{443},
					ServerNameMain:   sad.ServerNameMain,
					ServerNameExtras: sad.ServerNameExtras,
				}))
			}

		case ScenarioRoleUbuntuGeoIP:
			for _, addr := range sad.Addresses {
				opts = append(opts, QAEnvOptionNetStack(addr, &HTTPSecureServerFactory{
					Factory: &GeoIPHandlerFactoryUbuntu{
						ProbeIP: DefaultClientAddress,
					},
					Ports:            []int{443},
					ServerNameMain:   sad.ServerNameMain,
					ServerNameExtras: sad.ServerNameExtras,
				}))
			}

		case ScenarioRoleBlockpageServer:
			for _, addr := range sad.Addresses {
				opts = append(opts, QAEnvOptionNetStack(addr, &HTTPCleartextServerFactory{
					Factory: BlockpageHandlerFactory(),
					Ports:   []int{80},
				}))
			}

		case ScenarioRoleProxy:
			for _, addr := range sad.Addresses {
				opts = append(opts, QAEnvOptionNetStack(addr,
					&HTTPCleartextServerFactory{
						Factory: HTTPHandlerFactoryFunc(func(env NetStackServerFactoryEnv, stack *netem.UNetStack) http.Handler {
							return testingx.NewHTTPProxyHandler(env.Logger(), &netxlite.Netx{
								Underlying: &netxlite.NetemUnderlyingNetworkAdapter{UNet: stack}})
						}),
						Ports: []int{80},
					},
					NewTLSProxyServerFactory(log.Log, 443),
				))
			}

		case ScenarioRoleURLShortener:
			for _, addr := range sad.Addresses {
				opts = append(opts, QAEnvOptionNetStack(addr,
					&HTTPSecureServerFactory{
						Factory:          URLShortenerFactory(DefaultURLShortenerMapping),
						Ports:            []int{443},
						ServerNameMain:   sad.ServerNameMain,
						ServerNameExtras: sad.ServerNameExtras,
					},
				))
			}

		case ScenarioRoleBadSSL:
			for _, addr := range sad.Addresses {
				opts = append(opts, qaEnvOptionNetStack(addr, &BadSSLServerFactory{}))
			}
		}
	}

	// create QAEnv
	env := MustNewQAEnv(opts...)

	// configure all the domain names
	for _, sad := range config {
		for _, domain := range sad.Domains {
			env.AddRecordToAllResolvers(domain, "", sad.Addresses...)
		}
	}

	return env
}
