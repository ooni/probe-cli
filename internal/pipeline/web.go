package pipeline

import (
	"errors"
	"fmt"
	"net"

	"github.com/ooni/probe-cli/v3/internal/geoipx"
	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/optional"
)

// WebEndpointObservation is an endpoint observation made by the probe.
//
// Optional values represent data that may not be there if we do not
// find the expected events. Non-optional data should always be there.
//
// This type is inspired by and adapted from https://github.com/ooni/data
// and adapts the WebObservation type to probe-engine.
type WebEndpointObservation struct {
	// TransactionID is the ID of the transaction.
	TransactionID int64

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

	// TCPConnectT0 is when we started connecting.
	TCPConnectT0 optional.Value[float64]

	// TCPConnectT is when connect returned.
	TCPConnectT optional.Value[float64]

	// TCPConnectFailure is the error that occurred.
	TCPConnectFailure optional.Value[Failure]

	// DNSLookupGetaddrinfoXref contains references to the getaddrinfo
	// based DNS lookups that produced this IP address.
	DNSLookupGetaddrinfoXref []*DNSObservation

	// DNSLookupUDPXref contains references to the DNS-over-UDP
	// based DNS lookups that produced this IP address.
	DNSLookupUDPXref []*DNSObservation

	// DNSLookupHTTPSXref contains references to the DNS-over-HTTPS
	// based DNS lookups that produced this IP address.
	DNSLookupHTTPSXref []*DNSObservation

	// DNSLookupTHXref is true when this address was discovered by the TH
	DNSLookupTHXref bool

	// QUICHandshakeT0 is when we started handshaking.
	QUICHandshakeT0 optional.Value[float64]

	// QUICHandshakeT is when the QUIC handshake finished.
	QUICHandshakeT optional.Value[float64]

	// QUICHandshakeFailure is the error that occurred.
	QUICHandshakeFailure optional.Value[Failure]

	// TLSHandshakeT0 is when we started handshaking.
	TLSHandshakeT0 optional.Value[float64]

	// TLSHandshakeT is when the TLS handshake finished.
	TLSHandshakeT optional.Value[float64]

	// TLSHandshakeFailure is the error that occurred.
	TLSHandshakeFailure optional.Value[Failure]

	// TLSServerName is the SNI value.
	TLSServerName optional.Value[string]

	// TLSVersion is the negotiated TLS version.
	TLSVersion optional.Value[string]

	// TLSCipherSuite is the negotiated TLS cipher suite.
	TLSCipherSuite optional.Value[string]

	// TLSNegotiatedProtocol is the negotiated TLS protocol.
	TLSNegotiatedProtocol optional.Value[string]

	// THEndpointXref is a reference to the corresponding TH endpoint.
	THEndpointXref optional.Value[*EndpointObservationTH]

	// HTTPRequestURL is the HTTP request URL.
	HTTPRequestURL optional.Value[string]

	// HTTPFailure is the error that occurred.
	HTTPFailure optional.Value[Failure]

	// HTTPResponseStatusCode is the response status code.
	HTTPResponseStatusCode optional.Value[int64]

	// HTTPResponseBodyLength is the length of the response body.
	HTTPResponseBodyLength optional.Value[int64]

	// HTTPResponseBodyIsTruncated indicates whether the response body is truncated.
	HTTPResponseBodyIsTruncated optional.Value[bool]

	// HTTPResponseHeadersKeys contains the response headers keys.
	HTTPResponseHeadersKeys map[string]Origin

	// HTTPResponseTitle contains the response title.
	HTTPResponseTitle optional.Value[string]
}

func (db *DB) addNetworkEventsTCPConnect(evs ...*model.ArchivalNetworkEvent) error {
	for _, ev := range evs {
		switch {
		case ev.Operation == netxlite.ConnectOperation && ev.Proto == "tcp":
			wobs, err := db.newWebEndpointObservation(ev.TransactionID)
			if err != nil {
				return err
			}
			addr, port, err := net.SplitHostPort(ev.Address)
			if err != nil {
				return err
			}
			wobs.Proto = "tcp"
			wobs.IPAddress = addr
			wobs.Port = port
			wobs.Endpoint = fmt.Sprintf("%s/%s", ev.Address, ev.Proto)
			if asn, _, err := geoipx.LookupASN(addr); err == nil {
				wobs.IPAddressASN = optional.Some(int64(asn))
			}
			wobs.IPAddressIsBogon = netxlite.IsBogon(addr)
			wobs.TCPConnectT0 = optional.Some(ev.T0)
			wobs.TCPConnectT = optional.Some(ev.T)
			wobs.TCPConnectFailure = optional.Some(NewFailure(ev.Failure))

		default:
			// nothing
		}
	}
	return nil
}

func (db *DB) addTLSHandshakeEvents(evs ...*model.ArchivalTLSOrQUICHandshakeResult) error {
	for _, ev := range evs {
		wobs, err := db.getWebEndpointObservation(ev.TransactionID)
		if err != nil {
			return err
		}
		wobs.TLSHandshakeT0 = optional.Some(ev.T0)
		wobs.TLSHandshakeT = optional.Some(ev.T)
		wobs.TLSHandshakeFailure = optional.Some(NewFailure(ev.Failure))
		wobs.TLSServerName = optional.Some(ev.ServerName)
		wobs.TLSVersion = optional.Some(ev.TLSVersion)
		wobs.TLSCipherSuite = optional.Some(ev.CipherSuite)
		wobs.TLSNegotiatedProtocol = optional.Some(ev.NegotiatedProtocol)
	}
	return nil
}

func (db *DB) addQUICHandshakeEvents(evs ...*model.ArchivalTLSOrQUICHandshakeResult) error {
	for _, ev := range evs {
		wobs, err := db.newWebEndpointObservation(ev.TransactionID)
		if err != nil {
			return err
		}
		addr, port, err := net.SplitHostPort(ev.Address)
		if err != nil {
			return err
		}
		wobs.Proto = ev.Network
		wobs.IPAddress = addr
		wobs.Port = port
		wobs.Endpoint = fmt.Sprintf("%s/%s", ev.Address, ev.Network)
		if asn, _, err := geoipx.LookupASN(addr); err == nil {
			wobs.IPAddressASN = optional.Some(int64(asn))
		}
		wobs.IPAddressIsBogon = netxlite.IsBogon(addr)
		wobs.QUICHandshakeT0 = optional.Some(ev.T0)
		wobs.QUICHandshakeT = optional.Some(ev.T)
		wobs.QUICHandshakeFailure = optional.Some(NewFailure(ev.Failure))
		wobs.TLSServerName = optional.Some(ev.ServerName)
		wobs.TLSVersion = optional.Some(ev.TLSVersion)
		wobs.TLSCipherSuite = optional.Some(ev.CipherSuite)
		wobs.TLSNegotiatedProtocol = optional.Some(ev.NegotiatedProtocol)
	}
	return nil
}

func (db *DB) addHTTPRoundTrips(evs ...*model.ArchivalHTTPRequestResult) error {
	for _, ev := range evs {
		wobs, err := db.getWebEndpointObservation(ev.TransactionID)
		if err != nil {
			return err
		}
		wobs.HTTPRequestURL = optional.Some(ev.Request.URL)
		wobs.HTTPFailure = optional.Some(NewFailure(ev.Failure))
		wobs.HTTPResponseStatusCode = optional.Some(ev.Response.Code)
		wobs.HTTPResponseBodyLength = optional.Some(int64(len(ev.Response.Body)))
		wobs.HTTPResponseBodyIsTruncated = optional.Some(ev.Response.BodyIsTruncated)
		wobs.HTTPResponseHeadersKeys = make(map[string]Origin)
		for key := range ev.Response.Headers {
			wobs.HTTPResponseHeadersKeys[key] = OriginProbe
		}
		if title := measurexlite.WebGetTitleString(string(ev.Response.Body)); title != "" {
			wobs.HTTPResponseTitle = optional.Some(title)
		}
	}
	return nil
}

var errNoSuchTransaction = errors.New("analysis: no such transaction")

func (db *DB) getWebEndpointObservation(txid int64) (*WebEndpointObservation, error) {
	wobs, good := db.WebByTxID[txid]
	if !good {
		return nil, fmt.Errorf("%w: %d", errNoSuchTransaction, txid)
	}
	return wobs, nil
}

var errTransactionAlreadyExists = errors.New("analysis: transaction already exists")

func (db *DB) newWebEndpointObservation(txid int64) (*WebEndpointObservation, error) {
	if _, good := db.WebByTxID[txid]; good {
		return nil, errTransactionAlreadyExists
	}
	wobs := &WebEndpointObservation{
		TransactionID: txid,
	}
	db.WebByTxID[txid] = wobs
	return wobs, nil
}
