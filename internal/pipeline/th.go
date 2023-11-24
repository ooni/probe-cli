package pipeline

import (
	"errors"
	"fmt"
	"net"

	"github.com/ooni/probe-cli/v3/internal/geoipx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/optional"
)

// EndpointObservationTH is an endpoint observation made by the test helper (TH).
//
// Optional values represent data that may not be there if we do not
// find the expected events. Non-optional data should always be there.
//
// This type is inspired by and adapted from https://github.com/ooni/data
// and adapts the WebControlObservation type to probe-engine.
type EndpointObservationTH struct {
	// Proto is "tcp" (http/https) or "udp" (http/3).
	Proto string

	// IPAddress is the IPv4/IPv6 address.
	IPAddress string

	// Port is the TCP/UDP port.
	Port string

	// Endpoint summarizes IPAddress, Port, and Proto.
	Endpoint string

	// IPAddressASN is the IPAddress of the ASN.
	IPAddressASN optional.Value[int64]

	// IPAddressIsBogon indicates that the IP address is a bogon.
	IPAddressIsBogon bool

	// TCPConnectFailure is the error that occurred.
	TCPConnectFailure optional.Value[*string]

	// QUICHandshakeFailure is the error that occurred.
	QUICHandshakeFailure optional.Value[*string]

	// TLSHandshakeFailure is the error that occurred.
	TLSHandshakeFailure optional.Value[*string]

	// TLSServerName is the SNI value.
	TLSServerName optional.Value[string]
}

// WebObservationTH is a web observation made by the TH.
//
// Optional values represent data that may not be there if we do not
// find the expected events. Non-optional data should always be there.
//
// This type is inspired by and adapted from https://github.com/ooni/data
// and adapts the WebControlObservation type to probe-engine.
type WebObservationTH struct {
	// HTTPFailure is the error that occurred.
	HTTPFailure optional.Value[*string]

	// HTTPResponseStatusCode is the response status code.
	HTTPResponseStatusCode optional.Value[int64]

	// HTTPResponseBodyLength is the length of the response body.
	HTTPResponseBodyLength optional.Value[int64]

	// HTTPResponseHeadersKeys contains the response headers keys.
	HTTPResponseHeadersKeys map[string]Origin

	// HTTPResponseTitle contains the response title.
	HTTPResponseTitle optional.Value[string]
}

func (db *DB) thAddDNS(resp *model.THResponse) error {
	db.thDNSFailure = resp.DNS.Failure
	for _, addr := range resp.DNS.Addrs {
		db.thDNSAddrs[addr] = true
	}
	return nil
}

var errInconsistentTHResponse = errors.New("analysis: inconsistent TH response")

func (db *DB) thAddTCPConnect(resp *model.THResponse) error {
	for addrport, status := range resp.TCPConnect {
		addr, port, err := net.SplitHostPort(addrport)
		if err != nil {
			return err
		}

		endpoint := fmt.Sprintf("%s/tcp", addrport)
		var asn optional.Value[int64]
		if v, _, err := geoipx.LookupASN(addr); err == nil {
			asn = optional.Some(int64(v))
		}

		// Implementation note: because we're reading a map, we can't have duplicates
		// so we can blindly insert into the destination map here
		db.thEpntByEpnt[endpoint] = &EndpointObservationTH{
			Proto:             "tcp",
			IPAddress:         addr,
			Port:              port,
			Endpoint:          endpoint,
			IPAddressASN:      asn,
			IPAddressIsBogon:  netxlite.IsBogon(addr),
			TCPConnectFailure: optional.Some(status.Failure),
		}
	}
	return nil
}

func (db *DB) thAddTLSHandshake(resp *model.THResponse) error {
	for addrport, status := range resp.TLSHandshake {
		endpoint := fmt.Sprintf("%s/tcp", addrport)

		entry, found := db.thEpntByEpnt[endpoint]
		if !found {
			return errInconsistentTHResponse
		}

		entry.TLSServerName = optional.Some(status.ServerName)
		entry.TLSHandshakeFailure = optional.Some(status.Failure)
	}
	return nil
}

var errAlreadyExistingTHWeb = errors.New("analysis: thWeb already exists")

func (db *DB) thAddHTTPResponse(resp *model.THResponse) error {
	if !db.thWeb.IsNone() {
		return errAlreadyExistingTHWeb
	}

	db.thWeb = optional.Some(&WebObservationTH{
		HTTPFailure:            optional.Some(resp.HTTPRequest.Failure),
		HTTPResponseStatusCode: optional.Some(resp.HTTPRequest.StatusCode),
		HTTPResponseBodyLength: optional.Some(resp.HTTPRequest.BodyLength),
		HTTPResponseHeadersKeys: func() (out map[string]Origin) {
			out = make(map[string]Origin)
			for key := range resp.HTTPRequest.Headers {
				out[key] = OriginTH
			}
			return
		}(),
		HTTPResponseTitle: optional.Some(resp.HTTPRequest.Title),
	})

	return nil
}
