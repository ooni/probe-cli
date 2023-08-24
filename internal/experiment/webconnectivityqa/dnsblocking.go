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
			env.EmulateAndroidGetaddrinfo(true)
			env.ISPResolverConfig().RemoveRecord("www.example.com")
		},
		ExpectErr:      false,
		ExpectTestKeys: &testKeys{Accessible: false, Blocking: "dns"},
	}
}
