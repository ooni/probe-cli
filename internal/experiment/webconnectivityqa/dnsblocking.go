package webconnectivityqa

import (
	"github.com/ooni/probe-cli/v3/internal/netemx"
)

// dnsBlockingAndroidDNSCacheNoData is the case where we're on Android and the getaddrinfo
// resolver returns the android_dns_cache_no_data error.
func dnsBlockingAndroidDNSCacheNoData() *TestCase {
	return &TestCase{
		Name:  "measuring https://www.example.com/ with getaddrinfo errors and android_dns_cache_no_data",
		Flags: TestCaseFlagNoV04,
		Input: "https://www.example.com/",
		Configure: func(env *netemx.QAEnv) {
			// make sure the env knows we want to emulate our getaddrinfo wrapper behavior
			env.EmulateAndroidGetaddrinfo(true)

			// remove the record so that the DNS query returns NXDOMAIN, which is then
			// converted into android_dns_cache_no_data by the emulation layer
			env.ISPResolverConfig().RemoveRecord("www.example.com")
		},
		ExpectErr:      false,
		ExpectTestKeys: &testKeys{Accessible: false, Blocking: "dns"},
	}
}
