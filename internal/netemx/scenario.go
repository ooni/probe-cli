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
	// Domains contains a related set of domains domains (MANDATORY field).
	Domains []string

	// Addresses contains the MANDATORY list of addresses belonging to the domain.
	Addresses []string

	// Role is the MANDATORY role of this domain (e.g., ScenarioRoleOONIAPI).
	Role uint64

	// WebServerFactory is the factory to use when Role is ScenarioRoleWebServer.
	WebServerFactory HTTPHandlerFactory
}

// InternetScenario contains the domains and addresses used by [NewInternetScenario].
var InternetScenario = []*ScenarioDomainAddresses{{
	Domains: []string{"api.ooni.io"},
	Addresses: []string{
		AddressApiOONIIo,
	},
	Role: ScenarioRoleOONIAPI,
}, {
	Domains: []string{"geoip.ubuntu.com"},
	Addresses: []string{
		AddressGeoIPUbuntuCom,
	},
	Role: ScenarioRoleUbuntuGeoIP,
}, {
	Domains: []string{"www.example.com", "example.com", "www.example.org", "example.org"},
	Addresses: []string{
		AddressWwwExampleCom,
	},
	Role:             ScenarioRoleWebServer,
	WebServerFactory: ExampleWebPageHandlerFactory(),
}, {
	Domains: []string{"0.th.ooni.org"},
	Addresses: []string{
		AddressZeroThOONIOrg,
	},
	Role: ScenarioRoleOONITestHelper,
}, {
	Domains: []string{"1.th.ooni.org"},
	Addresses: []string{
		AddressOneThOONIOrg,
	},
	Role: ScenarioRoleOONITestHelper,
}, {
	Domains: []string{"2.th.ooni.org"},
	Addresses: []string{
		AddressTwoThOONIOrg,
	},
	Role: ScenarioRoleOONITestHelper,
}, {
	Domains: []string{"3.th.ooni.org"},
	Addresses: []string{
		AddressThreeThOONIOrg,
	},
	Role: ScenarioRoleOONITestHelper,
}, {
	Domains: []string{"d33d1gs9kpq1c5.cloudfront.net"},
	Addresses: []string{
		AddressTHCloudfront,
	},
	Role: ScenarioRoleOONITestHelper,
}, {
	Domains: []string{"dns.quad9.net"},
	Addresses: []string{
		AddressDNSQuad9Net,
	},
	Role: ScenarioRolePublicDNS,
}, {
	Domains: []string{"mozilla.cloudflare-dns.com"},
	Addresses: []string{
		AddressMozillaCloudflareDNSCom,
	},
	Role: ScenarioRolePublicDNS,
}, {
	Domains: []string{"dns.google", "dns.google.com"},
	Addresses: []string{
		AddressDNSGoogle8844,
		AddressDNSGoogle8888,
	},
	Role: ScenarioRolePublicDNS,
}, {
	Domains: []string{},
	Addresses: []string{
		AddressPublicBlockpage,
	},
	Role:             ScenarioRoleBlockpageServer,
	WebServerFactory: BlockpageHandlerFactory(),
}, {
	Domains: []string{},
	Addresses: []string{
		ISPProxyAddress,
	},
	Role:             ScenarioRoleProxy,
	WebServerFactory: nil,
}, {
	Domains: []string{"bit.ly", "bitly.com"},
	Addresses: []string{
		AddressBitly,
	},
	Role: ScenarioRoleURLShortener,
}, {
	Domains: []string{
		"wrong.host.badssl.com",
		"untrusted-root.badssl.com",
		"expired.badssl.com",
	},
	Addresses: []string{
		AddressBadSSLCom,
	},
	Role: ScenarioRoleBadSSL,
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
						Factory:   &DNSOverHTTPSHandlerFactory{},
						Ports:     []int{443},
						TLSConfig: nil, // use netem's default
					},
				))
			}

		case ScenarioRoleWebServer:
			for _, addr := range sad.Addresses {
				opts = append(opts, QAEnvOptionHTTPServer(addr, sad.WebServerFactory))
			}

		case ScenarioRoleOONIAPI:
			for _, addr := range sad.Addresses {
				opts = append(opts, QAEnvOptionNetStack(addr, &HTTPSecureServerFactory{
					Factory:   &OOAPIHandlerFactory{},
					Ports:     []int{443},
					TLSConfig: nil, // use netem's default
				}))
			}

		case ScenarioRoleOONITestHelper:
			for _, addr := range sad.Addresses {
				opts = append(opts, QAEnvOptionNetStack(addr, &HTTPSecureServerFactory{
					Factory:   &OOHelperDFactory{},
					Ports:     []int{443},
					TLSConfig: nil, // use netem's default
				}))
			}

		case ScenarioRoleUbuntuGeoIP:
			for _, addr := range sad.Addresses {
				opts = append(opts, QAEnvOptionNetStack(addr, &HTTPSecureServerFactory{
					Factory: &GeoIPHandlerFactoryUbuntu{
						ProbeIP: DefaultClientAddress,
					},
					Ports:     []int{443},
					TLSConfig: nil, // use netem's default
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
							return testingx.HTTPHandlerProxy(env.Logger(), &netxlite.Netx{
								Underlying: &netxlite.NetemUnderlyingNetworkAdapter{UNet: stack}})
						}),
						Ports: []int{80},
					},
					NewTLSProxyServerFactory(log.Log, 443),
				))
			}

		case ScenarioRoleURLShortener:
			for _, addr := range sad.Addresses {
				opts = append(opts, QAEnvOptionHTTPServer(addr, URLShortenerFactory(DefaultURLShortenerMapping)))
			}

		case ScenarioRoleBadSSL:
			for _, addr := range sad.Addresses {
				opts = append(opts, qaEnvOptionNetStack(addr, &BadSSLServerFactory{}))
			}
		}
	}

	// create the QAEnv
	env := MustNewQAEnv(opts...)

	// configure all the domain names
	for _, sad := range config {
		for _, domain := range sad.Domains {
			env.AddRecordToAllResolvers(domain, "", sad.Addresses...)
		}
	}

	return env
}
