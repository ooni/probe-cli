package resolver

import (
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// Variables that other packages expect to find here but have been
// moved into the internal/netxlite package.
var (
	NewSerialResolver               = netxlite.NewSerialResolver
	NewDNSOverUDP                   = netxlite.NewDNSOverUDP
	NewDNSOverTCP                   = netxlite.NewDNSOverTCP
	NewDNSOverTLS                   = netxlite.NewDNSOverTLS
	NewDNSOverHTTPS                 = netxlite.NewDNSOverHTTPS
	NewDNSOverHTTPSWithHostOverride = netxlite.NewDNSOverHTTPSWithHostOverride
)

// Types that other packages expect to find here but have been
// moved into the internal/netxlite package.
type (
	DNSOverHTTPS    = netxlite.DNSOverHTTPS
	DNSOverTCP      = netxlite.DNSOverTCP
	DNSOverUDP      = netxlite.DNSOverUDP
	MiekgEncoder    = netxlite.DNSEncoderMiekg
	MiekgDecoder    = netxlite.DNSDecoderMiekg
	RoundTripper    = model.DNSTransport
	SerialResolver  = netxlite.SerialResolver
	Dialer          = model.Dialer
	DialContextFunc = netxlite.DialContextFunc
)
