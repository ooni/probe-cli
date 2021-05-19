package netplumbing

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math"
	"net"
	"net/http"
	"net/url"
	"time"

	"github.com/miekg/dns"
)

// DNSQuery sends a DNS query and returns the corresponding reply. The resolverURL
// argument identifies the resolver. We support the following resolvers:
//
// - "udp://address/": Do53 using UDP;
// - "tcp://address/": Do53 using TCP;
// - "dot://address/": DNS over TLS;
// - "https://address/path": DNS over HTTPS.
//
// The address _may_ contain an optional port. If there is no port, then
// we use the default port for the specified protocol.
func (txp *Transport) DNSQuery(
	ctx context.Context, resolverURL *url.URL, query *dns.Msg) (*dns.Msg, error) {
	queryData, err := query.Pack()
	if err != nil {
		return nil, err
	}
	var replyData []byte
	switch scheme := resolverURL.Scheme; scheme {
	case "udp":
		replyData, err = txp.dnsQueryUDP(ctx, resolverURL, queryData)
	case "tcp":
		replyData, err = txp.dnsQueryTCP(
			ctx, resolverURL, queryData, txp.DialContext, "53")
	case "dot":
		replyData, err = txp.dnsQueryTCP(
			ctx, resolverURL, queryData, txp.DialTLSContext, "853")
	case "https":
		replyData, err = txp.dnsQueryHTTPS(ctx, resolverURL, queryData)
	default:
		err = fmt.Errorf("%w: %s", ErrResolverNotImplemented, scheme)
	}
	if err != nil {
		return nil, err
	}
	return txp.dnsQueryMaybeTrace(ctx, queryData, replyData)
}

// ErrResolverNotImplemented indicates that we don't implement a given resolver.
var ErrResolverNotImplemented = errors.New("netplumbing: resolver not implemented")

// dnsQueryMaybeTrace enables tracing if needed.
func (txp *Transport) dnsQueryMaybeTrace(
	ctx context.Context, queryData, replyData []byte) (*dns.Msg, error) {
	if ht := ContextTraceHeader(ctx); ht != nil {
		return txp.dnsQueryParseReplyWithTraceHeader(ctx, queryData, replyData, ht)
	}
	return txp.dnsQueryParseReply(ctx, queryData, replyData)
}

// dnsQueryParseReplyWithTraceHeader traces and then parses the reply.
func (txp *Transport) dnsQueryParseReplyWithTraceHeader(
	ctx context.Context, queryData, replyData []byte, ht *TraceHeader) (*dns.Msg, error) {
	ht.add(&DNSRoundTripTrace{Query: queryData, Reply: replyData, Time: time.Now()})
	return txp.dnsQueryParseReply(ctx, queryData, replyData)
}

// DNSRoundTripTrace is a trace collected during a DNS round trip.
type DNSRoundTripTrace struct {
	// Query contains the raw query data.
	Query []byte

	// Reply contains the raw reply data.
	Reply []byte

	// Time is the time when we collected this sample.
	Time time.Time
}

// Kind implements TraceEvent.Kind.
func (te *DNSRoundTripTrace) Kind() string {
	return TraceKindDNSRoundTrip
}

// dnsQueryParseReply parses the reply and returns it.
func (txp *Transport) dnsQueryParseReply(
	ctx context.Context, queryData, replyData []byte) (*dns.Msg, error) {
	reply := &dns.Msg{}
	if err := reply.Unpack(replyData); err != nil {
		return nil, err
	}
	return reply, nil
}

// dnsQueryUDP sends a query using UDP and receives the corresponding response.
func (txp *Transport) dnsQueryUDP(
	ctx context.Context, resolverURL *url.URL, query []byte) ([]byte, error) {
	address := resolverURL.Host
	if _, _, err := net.SplitHostPort(address); err != nil {
		address = net.JoinHostPort(address, "53")
	}
	conn, err := txp.DialContext(ctx, "udp", address)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	return txp.dnsQueryUDPWithConn(ctx, conn, query)
}

// dnsQueryUDPWithConn sends data as the query over the specified conn
// and receives the response using the same conn.
func (txp *Transport) dnsQueryUDPWithConn(
	ctx context.Context, conn net.Conn, query []byte) ([]byte, error) {
	conn.SetDeadline(time.Now().Add(4 * time.Second))
	if _, err := conn.Write(query); err != nil {
		return nil, err
	}
	buffer := make([]byte, 4096)
	cnt, err := conn.Read(buffer)
	if err != nil {
		return nil, err
	}
	return buffer[:cnt], nil
}

// dnsDialFunc is the signature of the function used to dial.
type dnsDialFunc func(ctx context.Context, network, address string) (net.Conn, error)

// dnsQueryTCP sends a query using TCP and receives the corresponding response.
func (txp *Transport) dnsQueryTCP(ctx context.Context, resolverURL *url.URL,
	query []byte, dial dnsDialFunc, defaultPort string) ([]byte, error) {
	address := resolverURL.Host
	if _, _, err := net.SplitHostPort(address); err != nil {
		address = net.JoinHostPort(address, defaultPort)
	}
	conn, err := dial(ctx, "tcp", address)
	if err != nil {
		return nil, err
	}
	defer conn.Close()
	return txp.dnsQueryTCPWithConn(ctx, conn, query)
}

// dnsQueryTCPWithConn sends data as the query over the specified conn
// and receives the response using the same conn.
func (txp *Transport) dnsQueryTCPWithConn(
	ctx context.Context, conn net.Conn, query []byte) ([]byte, error) {
	if len(query) > math.MaxUint16 {
		return nil, errors.New("netplumbing: query too long")
	}
	conn.SetDeadline(time.Now().Add(10 * time.Second))
	// Write request
	buf := []byte{byte(len(query) >> 8)}
	buf = append(buf, byte(len(query)))
	buf = append(buf, query...)
	if _, err := conn.Write(buf); err != nil {
		return nil, err
	}
	// Read response
	header := make([]byte, 2)
	if _, err := io.ReadFull(conn, header); err != nil {
		return nil, err
	}
	length := int(header[0])<<8 | int(header[1])
	reply := make([]byte, length)
	if _, err := io.ReadFull(conn, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// dnsQueryHTTPS sends a query using DNS over HTTPS.
func (txp *Transport) dnsQueryHTTPS(
	ctx context.Context, resolverURL *url.URL, query []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(
		ctx, "POST", resolverURL.String(), bytes.NewReader(query))
	if err != nil {
		return nil, err
	}
	req.Header.Set("content-type", "application/dns-message")
	clnt := &http.Client{Transport: txp} // ephemeral
	resp, err := clnt.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		// TODO(bassosimone): we should map the status code to a
		// proper Error in the DNS context.
		return nil, errors.New("netplumbing: dns server returned error")
	}
	if resp.Header.Get("content-type") != "application/dns-message" {
		return nil, errors.New("netplumbing: invalid content-type for doh")
	}
	return ioutil.ReadAll(resp.Body)
}
