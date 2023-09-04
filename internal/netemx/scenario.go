package netemx

import "github.com/ooni/netem"

const (
	// ScenarioRoleDNSOverHTTPS means we should create a DNS-over-HTTPS server.
	ScenarioRoleDNSOverHTTPS = iota

	// ScenarioRoleExampleLikeWebServer means we should instantiate a www.example.com-like web server.
	ScenarioRoleExampleLikeWebServer

	// ScenarioRoleOONIAPI means we should instantiate the OONI API.
	ScenarioRoleOONIAPI

	// ScenarioRoleUbuntuGeoIP means we should instantiate the Ubuntu geoip service.
	ScenarioRoleUbuntuGeoIP

	// ScenarioRoleOONITestHelper means we should instantiate the oohelperd.
	ScenarioRoleOONITestHelper
)

// ScenarioDomainAddresses describes a domain and address used in a scenario.
type ScenarioDomainAddresses struct {
	// Domain is the related domain name (e.g., api.ooni.io).
	Domain string

	// Addresses contains the related IP addresses.
	Addresses []string

	// Role is the role for this domain (e.g., ScenarioRoleOONIAPI).
	Role uint64
}

const (
	// InternetScenarioAddressApiOONIIo is the IP address we use for api.ooni.io in the [InternetScenario].
	InternetScenarioAddressApiOONIIo = "162.55.247.208"

	// InternetScenarioAddressGeoIPUbuntuCom is the IP address we use for geoip.ubuntu.com in the [InternetScenario].
	InternetScenarioAddressGeoIPUbuntuCom = "185.125.188.132"

	// InternetScenarioAddressWwwExampleCom is the IP address we use for www.example.com in the [InternetScenario].
	InternetScenarioAddressWwwExampleCom = "93.184.216.34"

	// InternetScenarioAddressZeroThOONIOrg is the IP address we use for 0.th.ooni.org in the [InternetScenario].
	InternetScenarioAddressZeroThOONIOrg = "68.183.70.80"

	// InternetScenarioAddressOneThOONIOrg is the IP address we use for 1.th.ooni.org in the [InternetScenario].
	InternetScenarioAddressOneThOONIOrg = "137.184.235.44"

	// InternetScenarioAddressTwoThOONIOrg is the IP address we use for 2.th.ooni.org in the [InternetScenario].
	InternetScenarioAddressTwoThOONIOrg = "178.62.195.24"

	// InternetScenarioAddressThreeThOONIOrg is the IP address we use for 3.th.ooni.org in the [InternetScenario].
	InternetScenarioAddressThreeThOONIOrg = "209.97.183.73"

	// InternetScenarioAddressDNSQuad9Net is the IP address we use for dns.quad9.net in the [InternetScenario].
	InternetScenarioAddressDNSQuad9Net = "9.9.9.9"

	// InternetScenarioAddressMozillaCloudflareDNSCom is the IP address we use for mozilla.cloudflare-dns.com
	// in the [InternetScenario].
	InternetScenarioAddressMozillaCloudflareDNSCom = "172.64.41.4"

	// InternetScenarioAddressDNSGoogle is the IP address we use for dns.google in the [InternetScenario].
	InternetScenarioAddressDNSGoogle = "8.8.4.4"
)

// InternetScenario contains the domains and addresses used by [NewInternetScenario].
var InternetScenario = []*ScenarioDomainAddresses{{
	Domain: "api.ooni.io",
	Addresses: []string{
		InternetScenarioAddressApiOONIIo,
	},
	Role: ScenarioRoleOONIAPI,
}, {
	Domain: "geoip.ubuntu.com",
	Addresses: []string{
		InternetScenarioAddressGeoIPUbuntuCom,
	},
	Role: ScenarioRoleUbuntuGeoIP,
}, {
	Domain: "www.example.com",
	Addresses: []string{
		InternetScenarioAddressWwwExampleCom,
	},
	Role: ScenarioRoleExampleLikeWebServer,
}, {
	Domain: "0.th.ooni.org",
	Addresses: []string{
		InternetScenarioAddressZeroThOONIOrg,
	},
	Role: ScenarioRoleOONITestHelper,
}, {
	Domain: "1.th.ooni.org",
	Addresses: []string{
		InternetScenarioAddressOneThOONIOrg,
	},
	Role: ScenarioRoleOONITestHelper,
}, {
	Domain: "2.th.ooni.org",
	Addresses: []string{
		InternetScenarioAddressTwoThOONIOrg,
	},
	Role: ScenarioRoleOONITestHelper,
}, {
	Domain: "3.th.ooni.org",
	Addresses: []string{
		InternetScenarioAddressThreeThOONIOrg,
	},
	Role: ScenarioRoleOONITestHelper,
}, {
	Domain: "dns.quad9.net",
	Addresses: []string{
		InternetScenarioAddressDNSQuad9Net,
	},
	Role: ScenarioRoleDNSOverHTTPS,
}, {
	Domain: "mozilla.cloudflare-dns.com",
	Addresses: []string{
		InternetScenarioAddressMozillaCloudflareDNSCom,
	},
	Role: ScenarioRoleDNSOverHTTPS,
}, {
	Domain: "dns.google",
	Addresses: []string{
		InternetScenarioAddressDNSGoogle,
	},
	Role: ScenarioRoleDNSOverHTTPS,
}}

// MustNewScenario constructs a complete testing scenario using the domains and IP
// addresses contained by the given [ScenarioDomainAddresses] array.
func MustNewScenario(cfg []*ScenarioDomainAddresses) *QAEnv {
	var opts []QAEnvOption

	// create a common configuration for DoH servers
	dohConfig := netem.NewDNSConfig()
	for _, sad := range cfg {
		dohConfig.AddRecord(sad.Domain, "", sad.Addresses...)
	}

	// explicitly create the uncensored resolver
	opts = append(opts, QAEnvOptionDNSOverUDPResolvers(QAEnvDefaultUncensoredResolverAddress))

	// fill options based on the scenario config
	for _, sad := range cfg {
		switch sad.Role {
		case ScenarioRoleDNSOverHTTPS:
			for _, addr := range sad.Addresses {
				opts = append(opts, QAEnvOptionHTTPServer(addr, &DNSOverHTTPSHandlerFactory{
					Config: dohConfig,
				}))
			}

		case ScenarioRoleExampleLikeWebServer:
			for _, addr := range sad.Addresses {
				opts = append(opts, QAEnvOptionHTTPServer(addr, ExampleWebPageHandlerFactory()))
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
					ProbeIP: QAEnvDefaultClientAddress,
				}))
			}
		}
	}

	// create the QAEnv
	env := MustNewQAEnv(opts...)

	// configure all the domain names
	for _, sad := range cfg {
		env.AddRecordToAllResolvers(sad.Domain, "", sad.Addresses...)
	}

	return env
}
