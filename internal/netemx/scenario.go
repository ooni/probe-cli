package netemx

import "github.com/ooni/netem"

const (
	// ScenarioRoleDNSOverHTTPS means we should create a DNS-over-HTTPS server.
	ScenarioRoleDNSOverHTTPS = iota

	// ScenarioRoleWebServer means we should instantiate a webserver using a specific factory.
	ScenarioRoleWebServer

	// ScenarioRoleOONIAPI means we should instantiate the OONI API.
	ScenarioRoleOONIAPI

	// ScenarioRoleUbuntuGeoIP means we should instantiate the Ubuntu geoip service.
	ScenarioRoleUbuntuGeoIP

	// ScenarioRoleOONITestHelper means we should instantiate the oohelperd.
	ScenarioRoleOONITestHelper
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
	WebServerFactory QAEnvHTTPHandlerFactory
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
	Domains: []string{"dns.quad9.net"},
	Addresses: []string{
		AddressDNSQuad9Net,
	},
	Role: ScenarioRoleDNSOverHTTPS,
}, {
	Domains: []string{"mozilla.cloudflare-dns.com"},
	Addresses: []string{
		AddressMozillaCloudflareDNSCom,
	},
	Role: ScenarioRoleDNSOverHTTPS,
}, {
	Domains: []string{"dns.google"},
	Addresses: []string{
		AddressDNSGoogle,
	},
	Role: ScenarioRoleDNSOverHTTPS,
}, {
	Domains: []string{},
	Addresses: []string{
		AddressPublicBlockpage,
	},
	Role:             ScenarioRoleWebServer,
	WebServerFactory: BlockpageHandlerFactory(),
}}

// MustNewScenario constructs a complete testing scenario using the domains and IP
// addresses contained by the given [ScenarioDomainAddresses] array.
func MustNewScenario(config []*ScenarioDomainAddresses) *QAEnv {
	var opts []QAEnvOption

	// TODO(bassosimone): it's currently a bottleneck that the same server cannot be _at the
	// same time_ both a DNS over cleartext and a DNS over HTTPS server.
	//
	// As a result, the code below for initializing $stuff is more complex than it should.

	// create a common configuration for DoH servers
	dohConfig := netem.NewDNSConfig()
	for _, sad := range config {
		for _, domain := range sad.Domains {
			dohConfig.AddRecord(domain, "", sad.Addresses...)
		}
	}

	// explicitly create the uncensored resolver
	opts = append(opts, QAEnvOptionDNSOverUDPResolvers(DefaultUncensoredResolverAddress))

	// fill options based on the scenario config
	for _, sad := range config {
		switch sad.Role {
		case ScenarioRoleDNSOverHTTPS:
			for _, addr := range sad.Addresses {
				opts = append(opts, QAEnvOptionHTTPServer(addr, &DNSOverHTTPSHandlerFactory{
					Config: dohConfig,
				}))
			}

		case ScenarioRoleWebServer:
			for _, addr := range sad.Addresses {
				opts = append(opts, QAEnvOptionHTTPServer(addr, sad.WebServerFactory))
			}

		case ScenarioRoleOONIAPI:
			for _, addr := range sad.Addresses {
				opts = append(opts, QAEnvOptionHTTPServer(addr, &OOAPIHandlerFactory{}))
			}

		case ScenarioRoleOONITestHelper:
			for _, addr := range sad.Addresses {
				opts = append(opts, QAEnvOptionHTTPServer(addr, &OOHelperDFactory{}))
			}

		case ScenarioRoleUbuntuGeoIP:
			for _, addr := range sad.Addresses {
				opts = append(opts, QAEnvOptionHTTPServer(addr, &GeoIPHandlerFactoryUbuntu{
					ProbeIP: DefaultClientAddress,
				}))
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
