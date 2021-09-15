package measure

import (
	"context"
	"net"
	"time"

	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/netxlite/dnsx"
	"github.com/ooni/probe-cli/v3/internal/netxlite/errorsx"
)

// dnsxResolverSystem is the system resolver.
type dnsxResolverSystem struct {
	begin  time.Time
	logger Logger
}

func (r *dnsxResolverSystem) LookupHost(
	ctx context.Context, domain string) *LookupHostResult {
	resolver := netxlite.NewResolverStdlib(r.logger)
	defer resolver.CloseIdleConnections() // respect the protocol
	m := &LookupHostResult{
		Domain:  domain,
		Engine:  "system",
		Started: time.Since(r.begin),
	}
	m.Addrs, m.Failure = resolver.LookupHost(ctx, domain)
	m.Completed = time.Since(r.begin)
	return m
}

// dnsxTransport is a custom DNS transport.
type dnsxTransport interface {
	// LookupHost performs a lookup host operation for the given domain
	// name using the underlying transport and query type.
	LookupHost(ctx context.Context, domain string, qtype uint16) *LookupHostResult
}

func newDNSXTransportWithUDPConn(begin time.Time, conn net.Conn) dnsxTransport {
	return &dnsxTransportNetxlite{
		begin: begin,
		txp: dnsx.NewDNSOverUDP(
			netxlite.NewSingleUseDialer(conn),
			conn.RemoteAddr().String(),
		),
	}
}

// TODO(bassosimone): IDNA and all the other guarantees that
// we already provide for normal usage.

// TODO(bassosimone): wondering if this code should be
// actually moved to netxlite/dnsx.

type dnsxTransportNetxlite struct {
	begin time.Time
	txp   dnsx.RoundTripper
}

func (txp *dnsxTransportNetxlite) LookupHost(
	ctx context.Context, domain string, qtype uint16) *LookupHostResult {
	defer txp.txp.CloseIdleConnections() // respect the protocol
	m := &LookupHostResult{
		Engine:          txp.txp.Network(),
		Address:         txp.txp.Address(),
		QueryTypeInt:    qtype,
		QueryTypeString: dns.TypeToString[qtype],
		Domain:          domain,
		Started:         time.Since(txp.begin),
	}
	encoder := &dnsx.MiekgEncoder{}
	data, err := encoder.Encode(domain, qtype, txp.txp.RequiresPadding())
	if err != nil {
		m.Failure = errorsx.NewErrWrapper(
			errorsx.ClassifyResolverError,
			errorsx.ResolveOperation,
			err,
		)
		return m
	}
	m.Query = data
	data, err = txp.roundTrip(ctx, data)
	m.Completed = time.Since(txp.begin)
	if err != nil {
		m.Failure = errorsx.NewErrWrapper(
			errorsx.ClassifyResolverError,
			errorsx.ResolveOperation,
			err,
		)
		return m
	}
	m.Reply = data
	m.Addrs, m.Failure = txp.decode(qtype, data)
	return m
}

// roundTrip ensures we honour the context
func (txp *dnsxTransportNetxlite) roundTrip(ctx context.Context, query []byte) ([]byte, error) {
	replych := make(chan []byte, 1) // buffer
	errch := make(chan error, 1)    // buffer
	go func() {
		reply, err := txp.txp.RoundTrip(ctx, query)
		if err != nil {
			errch <- err
			return
		}
		replych <- reply
	}()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case reply := <-replych:
		return reply, nil
	case err := <-errch:
		return nil, err
	}
}

func (txp *dnsxTransportNetxlite) decode(qtype uint16, reply []byte) ([]string, error) {
	decoder := &dnsx.MiekgDecoder{}
	addrs, err := decoder.Decode(qtype, reply)
	if err != nil {
		return nil, errorsx.NewErrWrapper(
			errorsx.ClassifyResolverError,
			errorsx.TopLevelOperation,
			err,
		)
	}
	return addrs, nil
}
