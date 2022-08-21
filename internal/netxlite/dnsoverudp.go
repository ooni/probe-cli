package netxlite

//
// DNS-over-UDP transport
//

import (
	"context"
	"net"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// DNSOverUDPTransport is a DNS-over-UDP DNSTransport.
//
// To construct this type, either manually fill the fields marked as MANDATORY
// or just use the NewDNSOverUDPTransport factory directly.
//
// RoundTrip creates a new connected UDP socket for each outgoing query. Using a
// new socket is good because some censored environments will block the client UDP
// endpoint for several seconds when you query for blocked domains. We could also
// have used an unconnected UDP socket here, but:
//
// 1. connected sockets are great because they get some ICMP errors to be
// translated into socket errors (among them, host_unreachable);
//
// 2. connected sockets ignore responses from illegitimate IP addresses but
// most if not all DNS resolvers also do that, therefore this does not seem to
// be a realistic censorship vector. At the same time, connected sockets
// provide us for free with the feature that we don't need to bother with checking
// whether the reply comes from the expected server.
//
// Being able to observe some ICMP errors is good because it could possibly
// make this code suitable to implement parasitic traceroute.
//
// This transport is capable of collecting additional responses after the first
// response. To see these responses, use the AsyncRoundTrip method.
type DNSOverUDPTransport struct {
	// Decoder is the MANDATORY DNSDecoder to use.
	Decoder model.DNSDecoder

	// Dialer is the MANDATORY dialer used to create the conn.
	Dialer model.Dialer

	// Endpoint is the MANDATORY server's endpoint (e.g., 1.1.1.1:53)
	Endpoint string

	// IOTimeout is the MANDATORY I/O timeout after which any
	// conn created to perform round trips times out.
	IOTimeout time.Duration
}

// NewUnwrappedDNSOverUDPTransport creates a DNSOverUDPTransport instance
// that has not been wrapped yet.
//
// Arguments:
//
// - dialer is any type that implements the Dialer interface;
//
// - address is the endpoint address (e.g., 8.8.8.8:53).
//
// If the address contains a domain name rather than an IP address
// (e.g., dns.google:53), we will end up using the first of the
// IP addresses returned by the underlying DNS lookup performed using
// the dialer. This usage pattern is NOT RECOMMENDED because we'll
// have less control over which IP address is being used.
func NewUnwrappedDNSOverUDPTransport(dialer model.Dialer, address string) *DNSOverUDPTransport {
	return &DNSOverUDPTransport{
		Decoder:   &DNSDecoderMiekg{},
		Dialer:    dialer,
		Endpoint:  address,
		IOTimeout: 10 * time.Second,
	}
}

// RoundTrip sends a query and receives a response.
func (t *DNSOverUDPTransport) RoundTrip(
	ctx context.Context, query model.DNSQuery) (model.DNSResponse, error) {
	// QUIRK: the original code had a five seconds timeout, which is
	// consistent with the Bionic implementation. Let's enforce such a
	// timeout using the context in the outer operation because we
	// need to run for more seconds in the background to catch as many
	// duplicate replies as possible.
	//
	// See https://labs.ripe.net/Members/baptiste_jonglez_1/persistent-dns-connections-for-reliability-and-performance
	const opTimeout = 5 * time.Second
	ctx, cancel := context.WithTimeout(ctx, opTimeout)
	defer cancel()
	rawQuery, err := query.Bytes()
	if err != nil {
		return nil, err
	}
	conn, err := t.Dialer.DialContext(ctx, "udp", t.Endpoint)
	if err != nil {
		return nil, err
	}
	conn.SetDeadline(time.Now().Add(t.IOTimeout))
	joinedch := make(chan bool)
	myaddr := conn.LocalAddr().String()
	if _, err := conn.Write(rawQuery); err != nil {
		return nil, err
	}
	resp, err := t.recv(query, conn)
	if err != nil {
		return resp, err
	}
	// start a routine to listen for any delayed DNS response
	go t.sendRecvLoop(ctx, conn, query, myaddr, joinedch)
	return resp, nil
}

// RequiresPadding returns false for UDP according to RFC8467.
func (t *DNSOverUDPTransport) RequiresPadding() bool {
	return false
}

// Network returns the transport network, i.e., "udp".
func (t *DNSOverUDPTransport) Network() string {
	return "udp"
}

// Address returns the upstream server address.
func (t *DNSOverUDPTransport) Address() string {
	return t.Endpoint
}

// CloseIdleConnections closes idle connections, if any.
func (t *DNSOverUDPTransport) CloseIdleConnections() {
	// The underlying dialer MAY have idle connections so let's
	// forward the call...
	t.Dialer.CloseIdleConnections()
}

var _ model.DNSTransport = &DNSOverUDPTransport{}

// sendRecvLoop listens for delayed DNS responses after we have returned the first response
func (t *DNSOverUDPTransport) sendRecvLoop(ctx context.Context, conn net.Conn,
	query model.DNSQuery, myaddr string, eofch chan<- bool) {
	defer close(eofch) // synchronize with the caller
	trace := ContextTraceOrDefault(ctx)
	for {
		started := trace.TimeNow()
		resp, err := t.recv(query, conn)
		finished := trace.TimeNow()
		if err != nil {
			trace.OnDelayedDNSResponse(started, t, query, resp, []string{}, err, finished)
			return
		}
		addrs, err := resp.DecodeLookupHost()
		err = trace.OnDelayedDNSResponse(started, t, query, resp, addrs, err, finished)
		if err != nil {
			return
		}
	}
}

// recv receives a single response for the given query using the given conn.
func (t *DNSOverUDPTransport) recv(query model.DNSQuery, conn net.Conn) (model.DNSResponse, error) {
	const maxmessagesize = 1 << 17
	rawResponse := make([]byte, maxmessagesize)
	count, err := conn.Read(rawResponse)
	if err != nil {
		return nil, err
	}
	rawResponse = rawResponse[:count]
	return t.Decoder.DecodeResponse(rawResponse, query)
}
