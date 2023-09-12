package measurex

//
// Oddity
//
// Here we define the oddity type.
//

// Oddity is an unexpected result on the probe or
// or test helper side during a measurement. We will
// promote the oddity to anomaly if the probe and
// the test helper see different results.
type Oddity string

// This enumeration lists all known oddities.
var (
	// tcp.connect
	OddityTCPConnectTimeout         = Oddity("tcp.connect.timeout")
	OddityTCPConnectRefused         = Oddity("tcp.connect.refused")
	OddityTCPConnectHostUnreachable = Oddity("tcp.connect.host_unreachable")
	OddityTCPConnectOher            = Oddity("tcp.connect.other")

	// tls.handshake
	OddityTLSHandshakeTimeout          = Oddity("tls.handshake.timeout")
	OddityTLSHandshakeReset            = Oddity("tls.handshake.reset")
	OddityTLSHandshakeOther            = Oddity("tls.handshake.other")
	OddityTLSHandshakeUnexpectedEOF    = Oddity("tls.handshake.unexpected_eof")
	OddityTLSHandshakeInvalidHostname  = Oddity("tls.handshake.invalid_hostname")
	OddityTLSHandshakeUnknownAuthority = Oddity("tls.handshake.unknown_authority")

	// quic.handshake
	OddityQUICHandshakeTimeout         = Oddity("quic.handshake.timeout")
	OddityQUICHandshakeHostUnreachable = Oddity("quic.handshake.host_unreachable")
	OddityQUICHandshakeOther           = Oddity("quic.handshake.other")

	// dns.lookup
	OddityDNSLookupNXDOMAIN = Oddity("dns.lookup.nxdomain")
	OddityDNSLookupTimeout  = Oddity("dns.lookup.timeout")
	OddityDNSLookupRefused  = Oddity("dns.lookup.refused")
	OddityDNSLookupBogon    = Oddity("dns.lookup.bogon")
	OddityDNSLookupOther    = Oddity("dns.lookup.other")

	// http.status
	OddityStatus403   = Oddity("http.status.403")
	OddityStatus404   = Oddity("http.status.404")
	OddityStatus503   = Oddity("http.status.503")
	OddityStatusOther = Oddity("http.status.other")
)
