package tlsmiddlebox

//
// Iterative network tracing
//

import (
	"context"
	"crypto/tls"
	"sort"
	"sync"
	"time"

	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	utls "gitlab.com/yawning/utls.git"
)

// ClientIDs to map configurable inputs to uTLS fingerprints
// We use a non-zero index to map to each ClientID
var ClientIDs = map[int]*utls.ClientHelloID{
	1: &utls.HelloGolang,
	2: &utls.HelloChrome_Auto,
	3: &utls.HelloFirefox_Auto,
	4: &utls.HelloIOS_Auto,
}

// DNSLookup performs a DNS Lookup for the passed domain
func (m *Measurer) DNSLookup(ctx context.Context, index int64, zeroTime time.Time,
	logger model.Logger, domain string, tk *TestKeys) ([]string, error) {
	url := m.config.resolverURL()
	trace := measurexlite.NewTrace(index, zeroTime)
	ol := measurexlite.NewOperationLogger(logger, "DNSLookup #%d, %s, %s", index, url, domain)
	// TODO(DecFox): We are currently using the DoH resolver, we will
	// switch to the TRR2 resolver once we have it in measurexlite
	// Issue: https://github.com/ooni/probe/issues/2185
	resolver := trace.NewParallelDNSOverHTTPSResolver(logger, url)
	addrs, err := resolver.LookupHost(ctx, domain)
	ol.Stop(err)
	tk.addQueries(trace.DNSLookupsFromRoundTrip())
	return addrs, err
}

// TCPConnect performs a TCP connect to filter working addresses
func (m *Measurer) TCPConnect(ctx context.Context, index int64, zeroTime time.Time,
	logger model.Logger, address string, tk *TestKeys) error {
	trace := measurexlite.NewTrace(index, zeroTime)
	dialer := trace.NewDialerWithoutResolver(logger)
	ol := measurexlite.NewOperationLogger(logger, "TCPConnect #%d %s", index, address)
	conn, err := dialer.DialContext(ctx, "tcp", address)
	ol.Stop(err)
	measurexlite.MaybeClose(conn)
	tcpEvents := trace.TCPConnects()
	tk.addTCPConnect(tcpEvents)
	return err
}

// MeasureTLS performs tracing using control and target SNI
func (m *Measurer) TLSTrace(ctx context.Context, index int64, zeroTime time.Time, logger model.Logger,
	address string, targetSNI string, trace *CompleteTrace) {
	// perform an iterative trace with the control SNI
	trace.ControlTrace = m.IterativeTrace(ctx, index, zeroTime, logger, address, m.config.snicontrol())
	// perform an iterative trace with the target SNI
	trace.TargetTrace = m.IterativeTrace(ctx, index, zeroTime, logger, address, targetSNI)
}

// IterativeTrace creates a Trace and calls iterativeTrace
func (m *Measurer) IterativeTrace(ctx context.Context, index int64, zeroTime time.Time, logger model.Logger,
	address string, sni string) (tr *IterativeTrace) {
	tr = &IterativeTrace{
		SNI:        sni,
		Iterations: []*Iteration{},
	}
	maxTTL := m.config.maxttl()
	m.iterativeTrace(ctx, index, zeroTime, logger, address, sni, maxTTL, tr)
	tr.Iterations = alignIterations(tr.Iterations)
	return
}

// iterativeTrace performs iterative tracing with increasing TTL values
func (m *Measurer) iterativeTrace(ctx context.Context, index int64, zeroTime time.Time, logger model.Logger,
	address string, sni string, maxTTL int64, trace *IterativeTrace) {
	ticker := time.NewTicker(m.config.delay())
	wg := new(sync.WaitGroup)
	for i := int64(1); i <= maxTTL; i++ {
		wg.Add(1)
		go m.handshakeWithTTL(ctx, index, zeroTime, logger, address, sni, int(i), trace, wg)
		<-ticker.C
	}
	wg.Wait()
}

// HandshakeWithTTL performs the TLS Handshake using the passed ttl value
func (m *Measurer) handshakeWithTTL(ctx context.Context, index int64, zeroTime time.Time, logger model.Logger,
	address string, sni string, ttl int, tr *IterativeTrace, wg *sync.WaitGroup) {
	defer wg.Done()
	trace := measurexlite.NewTrace(index, zeroTime)
	// TODO(DecFox): Do we need a trace for this TCP connect?
	d := NewDialerTTLWrapper()
	ol := measurexlite.NewOperationLogger(logger, "Handshake Trace #%d TTL %d %s %s", index, ttl, address, sni)
	conn, err := d.DialContext(ctx, "tcp", address)
	if err != nil {
		iteration := newIterationFromHandshake(ttl, err, nil)
		tr.addIterations(iteration)
		ol.Stop(err)
		return
	}
	defer conn.Close()
	err = setConnTTL(conn, ttl)
	if err != nil {
		iteration := newIterationFromHandshake(ttl, err, nil)
		tr.addIterations(iteration)
		ol.Stop(err)
		return
	}
	thx := trace.NewTLSHandshakerStdlib(logger)
	// initialise a uTLS Handshaker if the ClientID is non-zero
	clientId := m.config.clientid()
	if clientId > 0 {
		thx = trace.NewTLSHandshakerUTLS(logger, ClientIDs[clientId])
	}
	_, _, err = thx.Handshake(ctx, conn, genTLSConfig(sni))
	ol.Stop(err)
	// reset the TTL value to ensure that conn closes successfully
	// Note: we do not check for errors here
	setConnTTL(conn, 64)
	iteration := newIterationFromHandshake(ttl, nil, trace.FirstTLSHandshakeOrNil())
	tr.addIterations(iteration)
}

// genTLSConfig generates tls.Config from a given SNI
func genTLSConfig(sni string) *tls.Config {
	return &tls.Config{
		RootCAs:            netxlite.NewDefaultCertPool(),
		ServerName:         sni,
		NextProtos:         []string{"h2", "http/1.1"},
		InsecureSkipVerify: true,
	}
}

// alignIterEvents sorts the iterEvents according to increasing TTL
// and stops when we receive a nil or connection_reset
func alignIterations(in []*Iteration) (out []*Iteration) {
	out = []*Iteration{}
	sort.Slice(in, func(i int, j int) bool {
		return in[i].TTL < in[j].TTL
	})
	for _, iter := range in {
		out = append(out, iter)
		if iter.Handshake.Failure == nil || *iter.Handshake.Failure == netxlite.FailureConnectionReset {
			break
		}
	}
	return out
}
