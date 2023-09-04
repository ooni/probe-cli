package netemx

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
	Domains: []string{"dns.google"},
	Addresses: []string{
		AddressDNSGoogle,
	},
	Role: ScenarioRolePublicDNS,
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

	// explicitly create the uncensored resolver
	opts = append(opts, QAEnvOptionDNSOverUDPResolvers(DefaultUncensoredResolverAddress))

	// fill options based on the scenario config
	for _, sad := range config {
		switch sad.Role {
		case ScenarioRolePublicDNS:
			for _, addr := range sad.Addresses {
				opts = append(opts, QAEnvOptionNetStack(
					addr,
					&UDPResolverFactory{},
					&HTTPSecureServerFactory{
						Factory:   &DNSOverHTTPSHandlerFactory{},
						Ports:     []int{443},
						TLSConfig: nil, // use netem's default
					},
				))
			}

		case ScenarioRoleWebServer:
			for _, addr := range sad.Addresses {
				opts = append(opts, QAEnvOptionNetStack(addr, &HTTPCleartextServerFactory{
					Factory: sad.WebServerFactory,
					Ports:   []int{80},
				}, &HTTPSecureServerFactory{
					Factory:   sad.WebServerFactory,
					Ports:     []int{443},
					TLSConfig: nil, // use netem's default
				}, &HTTP3ServerFactory{
					Factory:   sad.WebServerFactory,
					Ports:     []int{443},
					TLSConfig: nil, // use netem's default
				}))
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
