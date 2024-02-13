package webconnectivitylte

import "sync/atomic"

const (
	idGeneratorGetaddrinfoOffset       = 10_000
	idGeneratorDNSOverUDPOffset        = 20_000
	idGeneratorDNSOverHTTPSOffset      = 30_000
	idGeneratorEndpointCleartextOffset = 40_000
	idGeneratorEndpointSecureOffset    = 50_000
)

// IDGenerator helps with generating IDs that neatly fall into namespaces.
//
// The zero value is invalid, please use [NewIDGenerator].
type IDGenerator struct {
	// getaddrinfo generates IDs for getaddrinfo.
	getaddrinfo *atomic.Int64

	// dnsOverUDP generates IDs for DNS-over-UDP lookups.
	dnsOverUDP *atomic.Int64

	// dnsOverHTTPS generates IDs for DNS-over-HTTPS lookups.
	dnsOverHTTPS *atomic.Int64

	// endpointCleartext generates IDs for endpoints using HTTP.
	endpointCleartext *atomic.Int64

	// endpointSecure generates IDs for endpoints using HTTPS.
	endpointSecure *atomic.Int64
}

// NewIDGenerator creates a new [*IDGenerator] instance.
func NewIDGenerator() *IDGenerator {
	return &IDGenerator{
		getaddrinfo:       &atomic.Int64{},
		dnsOverUDP:        &atomic.Int64{},
		dnsOverHTTPS:      &atomic.Int64{},
		endpointCleartext: &atomic.Int64{},
		endpointSecure:    &atomic.Int64{},
	}
}

// NewIDForGetaddrinfo returns a new ID for a getaddrinfo lookup.
func (idgen *IDGenerator) NewIDForGetaddrinfo() int64 {
	return idgen.getaddrinfo.Add(1) + idGeneratorGetaddrinfoOffset
}

// NewIDForDNSOverUDP returns a new ID for a DNS-over-UDP lookup.
func (idgen *IDGenerator) NewIDForDNSOverUDP() int64 {
	return idgen.dnsOverUDP.Add(1) + idGeneratorDNSOverUDPOffset
}

// NewIDForDNSOverHTTPS returns a new ID for a DNS-over-HTTPS lookup.
func (idgen *IDGenerator) NewIDForDNSOverHTTPS() int64 {
	return idgen.dnsOverHTTPS.Add(1) + idGeneratorDNSOverHTTPSOffset
}

// NewIDForEndpointCleartext returns a new ID for a cleartext endpoint operation.
func (idgen *IDGenerator) NewIDForEndpointCleartext() int64 {
	return idgen.endpointCleartext.Add(1) + idGeneratorEndpointCleartextOffset
}

// NewIDForEndpointSecure returns a new ID for a secure endpoint operation.
func (idgen *IDGenerator) NewIDForEndpointSecure() int64 {
	return idgen.endpointSecure.Add(1) + idGeneratorEndpointSecureOffset
}
