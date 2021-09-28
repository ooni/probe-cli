package resolver

import "github.com/ooni/probe-cli/v3/internal/netxlite"

// Variables that other packages expect to find here but have been
// moved into the internal/netxlite/dnsx package.
var (
	NewSerialResolver               = netxlite.NewSerialResolver
	NewDNSOverUDP                   = netxlite.NewDNSOverUDP
	NewDNSOverTCP                   = netxlite.NewDNSOverTCP
	NewDNSOverTLS                   = netxlite.NewDNSOverTLS
	NewDNSOverHTTPS                 = netxlite.NewDNSOverHTTPS
	NewDNSOverHTTPSWithHostOverride = netxlite.NewDNSOverHTTPSWithHostOverride
)

// Types that other packages expect to find here but have been
// moved into the internal/netxlite/dnsx package.
type (
	DNSOverHTTPS    = netxlite.DNSOverHTTPS
	DNSOverTCP      = netxlite.DNSOverTCP
	DNSOverUDP      = netxlite.DNSOverUDP
	MiekgEncoder    = netxlite.DNSEncoderMiekg
	MiekgDecoder    = netxlite.DNSDecoderMiekg
	RoundTripper    = netxlite.DNSTransport
	SerialResolver  = netxlite.SerialResolver
	Dialer          = netxlite.Dialer
	DialContextFunc = netxlite.DialContextFunc
)
