package resolver

import "github.com/ooni/probe-cli/v3/internal/netxlite/dnsx"

// Variables that other packages expect to find here but have been
// moved into the internal/netxlite/dnsx package.
var (
	NewSerialResolver               = dnsx.NewSerialResolver
	NewDNSOverUDP                   = dnsx.NewDNSOverUDP
	NewDNSOverTCP                   = dnsx.NewDNSOverTCP
	NewDNSOverTLS                   = dnsx.NewDNSOverTLS
	NewDNSOverHTTPS                 = dnsx.NewDNSOverHTTPS
	NewDNSOverHTTPSWithHostOverride = dnsx.NewDNSOverHTTPSWithHostOverride
)

// Types that other packages expect to find here but have been
// moved into the internal/netxlite/dnsx package.
type (
	DNSOverHTTPS    = dnsx.DNSOverHTTPS
	DNSOverTCP      = dnsx.DNSOverTCP
	DNSOverUDP      = dnsx.DNSOverUDP
	MiekgEncoder    = dnsx.MiekgEncoder
	MiekgDecoder    = dnsx.MiekgDecoder
	RoundTripper    = dnsx.RoundTripper
	SerialResolver  = dnsx.SerialResolver
	Dialer          = dnsx.Dialer
	DialContextFunc = dnsx.DialContextFunc
)
