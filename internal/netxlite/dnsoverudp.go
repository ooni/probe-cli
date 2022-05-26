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
// most if not all DNS resolvers also do that, therefore it does not seem to
// be a realistic censorship vector. At the same time, connected sockets
// provide us for free the feature that we don't need to bother with checking
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

// NewDNSOverUDPTransport creates a DNSOverUDPTransport instance.
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
func NewDNSOverUDPTransport(dialer model.Dialer, address string) *DNSOverUDPTransport {
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
	outch := t.AsyncRoundTrip(query, 1) // buffer to avoid background's goroutine leak
	return outch.Next(ctx)
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

// DNSOverUDPResponse is a response received by a DNSOverUDPTransport when you
// use its AsyncRoundTrip method as opposed to using RoundTrip.
type DNSOverUDPResponse struct {
	// Err is the error that occurred (nil in case of success).
	Err error

	// LocalAddr is the local UDP address we're using.
	LocalAddr string

	// Operation is the operation that failed.
	Operation string

	// Query is the related DNS query.
	Query model.DNSQuery

	// RemoteAddr is the remote server address.
	RemoteAddr string

	// Response is the response (nil iff error is not nil).
	Response model.DNSResponse
}

// newDNSOverUDPResponse creates a new DNSOverUDPResponse instance.
func (t *DNSOverUDPTransport) newDNSOverUDPResponse(localAddr string, err error,
	query model.DNSQuery, resp model.DNSResponse, operation string) *DNSOverUDPResponse {
	return &DNSOverUDPResponse{
		Err:        err,
		LocalAddr:  localAddr,
		Operation:  operation,
		Query:      query,
		RemoteAddr: t.Endpoint, // The common case is to have an IP:port here (domains are discouraged)
		Response:   resp,
	}
}

// DNSOverUDPChannel is a wrapper around a channel for reading zero
// or more *DNSOverUDPResponse that makes extracting information from
// the underlying channels more user friendly than interacting with
// the channels directly, thanks to useful wrapper methods implementing
// common access patterns. You can still use the channels directly if
// there's no convenience method for your specific access pattern.
type DNSOverUDPChannel struct {
	// Response is the channel where we'll post responses. This channel
	// WON'T be closed when the background goroutine terminates.
	Response <-chan *DNSOverUDPResponse

	// Joined is a channel that IS CLOSED when the background
	// goroutine performing this round trip TERMINATES.
	Joined <-chan bool
}

// Next blocks until the next response is received on Response or the
// given context expires, whatever happens first. This function will
// completely ignore the Joined channel and will just timeout in case
// you call Next after the background goroutine had joined. In fact,
// the use case for this function is using it to get a response or
// a timeout when you know the DNS round trip is pending.
func (ch *DNSOverUDPChannel) Next(ctx context.Context) (model.DNSResponse, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case out := <-ch.Response: // Note: AsyncRoundTrip WILL NOT close the channel or emit a nil
		return out.Response, out.Err
	}
}

// TryNextResponses attempts to read all the buffered messages inside of the "Response"
// channel that contains successful DNS responses. That is, this function will silently skip
// any possible DNSOverUDPResponse with its Err != nil. The use case for this function is
// to obtain all the subsequent response messages we received while we were performing
// other operations (e.g., contacting the test helper of fetching a webpage).
func (ch *DNSOverUDPChannel) TryNextResponses() (out []model.DNSResponse) {
	for {
		select {
		case r := <-ch.Response: // Note: AsyncRoundTrip WILL NOT close the channel or emit a nil
			if r.Err == nil && r.Response != nil {
				out = append(out, r.Response)
			}
		default:
			return
		}
	}
}

// AsyncRoundTrip performs an async DNS round trip. The "buffer" argument
// controls how many buffer slots the returned DNSOverUDPChannel's Response
// channel should have. A zero or negative value causes this function to
// create a channel having a single-slot buffer.
//
// The real round trip runs in a background goroutine. We will terminate the background
// goroutine when (1) the IOTimeout expires for the connection we're using or (2) we
// cannot write on the "Response" channel. Note that the background goroutine WILL NOT
// close the "Response" channel to signal its completion. Hence, who reads such a
// channel MUST be prepared for read operations to block forever and use a
// select for draining the channel in a deadlock-safe way. Also, we WILL NOT ever
// emit a nil message over the "Response" channel.
//
// The returned DNSOverUDPChannel contains another channel called Joined that is
// closed when the background goroutine terminates, so you can use this channel
// should you need to synchronize with such goroutine's termination.
//
// If you are using the Next or TryNextResponses methods of the DNSOverUDPChannel type,
// you don't need to worry about these low level details though.
func (t *DNSOverUDPTransport) AsyncRoundTrip(query model.DNSQuery, buffer int) *DNSOverUDPChannel {
	if buffer < 2 {
		buffer = 1 // as documented
	}
	outch := make(chan *DNSOverUDPResponse, buffer)
	joinedch := make(chan bool)
	go t.roundTripLoop(query, outch, joinedch)
	return &DNSOverUDPChannel{
		Response: outch,
		Joined:   joinedch,
	}
}

// roundTripLoop performs the round trip and writes results into the "outch" channel. This
// function ASSUMES that "outch" is configured to have AT LEAST one buffer slot. This function
// TAKES OWNERSHIP of "outch" but WILL NOT close it when done. This function instead OWNS
// the "joinedch" channel and WILL CLOSE it when done.
func (t *DNSOverUDPTransport) roundTripLoop(
	query model.DNSQuery, outch chan<- *DNSOverUDPResponse, joinedch chan<- bool) {
	defer close(joinedch) // as documented
	rawQuery, err := query.Bytes()
	if err != nil {
		outch <- t.newDNSOverUDPResponse(
			"", err, query, nil, "serialize_query") // one-sized buffer, can't block
		return
	}
	// While dial operations return immediately for UDP, we MAY be calling the
	// dialer's resolver if t.Endpoint contains a domain name. So, let us basically
	// enforce the same overall deadline covering DNS lookup and I/O operations.
	deadline := time.Now().Add(t.IOTimeout)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()
	conn, err := t.Dialer.DialContext(ctx, "udp", t.Endpoint)
	if err != nil {
		outch <- t.newDNSOverUDPResponse(
			"", err, query, nil, ConnectOperation) // one-sized buffer, can't block
		return
	}
	defer conn.Close() // we own the conn
	conn.SetDeadline(deadline)
	localAddr := conn.LocalAddr().String()
	if _, err = conn.Write(rawQuery); err != nil {
		outch <- t.newDNSOverUDPResponse(
			localAddr, err, query, nil, WriteOperation) // one-sized buffer, can't block
		return
	}
	for {
		resp, err := t.recv(query, conn)
		select {
		case outch <- t.newDNSOverUDPResponse(localAddr, err, query, resp, ReadOperation):
		default:
			return // no-one is reading the channel -- so long...
		}
		if err != nil {
			// We are going to consider all errors as fatal for now until we
			// hear of specific errs that it might have sense to ignore.
			//
			// Note that erroring out here includes the expiration of the conn's
			// I/O deadline, which we set above precisely because we want
			// the total runtime of this goroutine to be bounded.
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
