package signalclient

//
// The address controls (ts-041 Controls): extra lookups and handshakes, after all targets, that
// show which method blocked a target (DNS vs address, address vs SNI, SNI vs the client's TLS
// fingerprint or version) and whether the real clients' own fallbacks would still connect.
//

import (
	"context"
	"crypto/rand"
	"crypto/x509"
	"math/big"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// Control kinds, in the order they run for a target.
const (
	ControlDoH       = "doh"
	ControlUDP       = "udp_1111"
	ControlStatic    = "static_address"
	ControlDecoySNI  = "decoy_sni"
	ControlAlternate = "alternate_hello"
	ControlBrowser   = "browser_hello"
	ControlTLS12     = "tls12"
)

// ControlResult is the outcome of one control. DNS kinds fill resolver and addresses; handshake
// kinds fill the address, SNI, hello and TLS fields.
type ControlResult struct {
	Kind            string    `json:"kind"`
	Resolver        string    `json:"resolver,omitempty"`
	Addresses       *[]string `json:"addresses,omitempty"` // DNS kinds: always present, possibly empty
	Address         string    `json:"address,omitempty"`
	AddressSource   string    `json:"address_source,omitempty"`
	SNI             string    `json:"sni,omitempty"`
	ClientHello     string    `json:"client_hello,omitempty"`
	Verified        *bool     `json:"verified,omitempty"`
	LeafKeyType     string    `json:"leaf_key_type,omitempty"`
	Status          *int64    `json:"status,omitempty"`
	SignalTimestamp *bool     `json:"x_signal_timestamp,omitempty"`
	Failure         *string   `json:"failure"`
	FailedOperation string    `json:"failed_operation,omitempty"`
	T0              float64   `json:"t0"`
	T               float64   `json:"t"`
}

// Limits (variables so tests can shorten them).
var (
	controlsBudget          = 60 * time.Second
	controlsParallelism     = 8
	dohTimeout              = 10 * time.Second
	udpTimeout              = 5 * time.Second
	controlHandshakeTimeout = 10 * time.Second
)

// The resolvers libsignal and Signal-Android fall back to. The IPv6 DoH resolver is used when
// the IPv4 one fails.
var (
	dohResolvers = []string{"https://1.1.1.1/dns-query", "https://[2606:4700:4700::1111]/dns-query"}
	udpResolver  = "1.1.1.1:53"
)

// failedBeforeSignalAnswered are the failed operations after which on_failure runs controls.
var failedBeforeSignalAnswered = map[string]bool{
	netxlite.ResolveOperation: true, netxlite.ConnectOperation: true, netxlite.TLSHandshakeOperation: true,
	netxlite.HTTPRoundTripOperation: true, "websocket_upgrade": true,
}

// controlJob is one control, possibly shared by several targets.
type controlJob struct {
	index int64
	kind  string
	run   func(ctx context.Context, job *controlJob) *ControlResult

	mu      sync.Mutex
	targets []*TargetResult // who records the result
	names   []string        // their names, for the trace tags

	done   chan struct{} // closed when finished or skipped
	result *ControlResult

	// evidence, merged into the test keys after all controls
	queries    []*model.ArchivalDNSLookupResult
	tcpConnect []*model.ArchivalTCPConnectResult
	handshakes []*model.ArchivalTLSOrQUICHandshakeResult
	events     []*model.ArchivalNetworkEvent
}

func (j *controlJob) addTarget(r *TargetResult) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.targets = append(j.targets, r)
	j.names = append(j.names, r.Name)
}

// newTrace returns a trace tagged with the job's targets and kind.
func (j *controlJob) newTrace(zero time.Time) *measurexlite.Trace {
	j.mu.Lock()
	tags := append(append([]string(nil), j.names...), "control:"+j.kind)
	j.mu.Unlock()
	return measurexlite.NewTrace(j.index, zero, tags...)
}

// collect keeps a trace's evidence.
func (j *controlJob) collect(trace *measurexlite.Trace) {
	j.queries = append(j.queries, trace.DNSLookupsFromRoundTrip()...)
	j.tcpConnect = append(j.tcpConnect, trace.TCPConnects()...)
	j.events = append(j.events, trace.NetworkEvents()...)
}

// controlRunner plans and runs the controls of one measurement.
type controlRunner struct {
	m      *Measurer
	logger model.Logger
	zero   time.Time
	roots  *x509.CertPool // Signal's (or SignalCA's)
	opts   *runOptions
	index  int64

	mu    sync.Mutex
	jobs  []*controlJob          // in start order
	byKey map[string]*controlJob // for deduplication
}

func newControlRunner(m *Measurer, logger model.Logger, zero time.Time, roots *x509.CertPool, opts *runOptions,
	firstIndex int64) *controlRunner {
	return &controlRunner{m: m, logger: logger, zero: zero, roots: roots, opts: opts, index: firstIndex,
		byKey: map[string]*controlJob{}}
}

// add plans a control for r, or adds r to an identical planned one.
func (cr *controlRunner) add(r *TargetResult, key, kind string, run func(context.Context, *controlJob) *ControlResult) {
	if job, ok := cr.byKey[key]; ok {
		job.addTarget(r)
		return
	}
	job := &controlJob{index: cr.index, kind: kind, run: run, done: make(chan struct{})}
	cr.index++
	job.addTarget(r)
	cr.byKey[key] = job
	cr.jobs = append(cr.jobs, job)
}

// plan decides the controls of every target, in target order and, within a target, in kind order.
func (cr *controlRunner) plan(targets []Target, results []*TargetResult) {
	for i, target := range targets {
		r := results[i]
		always := cr.opts.controls == ControlsAlways
		if !always && !failedBeforeSignalAnswered[r.FailedOperation] {
			continue
		}
		if !always && r.Failure != nil && answeredFailures[*r.Failure] {
			continue // the server answered (e.g. refused the upgrade): nothing on the path to explain
		}
		op := r.FailedOperation
		tlsAnswered := always || op == netxlite.TLSHandshakeOperation || op == netxlite.HTTPRoundTripOperation ||
			op == "websocket_upgrade"
		tlsFailed := always || op == netxlite.TLSHandshakeOperation
		host := target.Hostname
		list := cr.opts.static[host]
		tried := ""
		if r.Address != nil && *r.Address != "" {
			tried, _, _ = net.SplitHostPort(*r.Address)
		}

		dohJob := cr.planDNS(r, ControlDoH, host)
		cr.planDNS(r, ControlUDP, host)
		for _, a := range staticCandidates(list, tried) {
			cr.planHandshake(r, ControlStatic, target, a, target.Hello, true)
		}
		cr.planStaticFromDoH(r, target, dohJob, list, tried)

		// the address the target tried, or the first static address when resolution failed
		same := staticAddress{ip: tried, source: "target"}
		if tried == "" {
			var ok bool
			if same, ok = firstStatic(list); !ok {
				continue
			}
		}
		if tlsAnswered {
			cr.planHandshake(r, ControlDecoySNI, target, same, target.Hello, false)
			cr.planHandshake(r, ControlAlternate, target, same, alternateHello(target.Hello), true)
		}
		if tlsFailed {
			cr.planHandshake(r, ControlBrowser, target, same, HelloChrome, false)
			cr.planHandshake(r, ControlTLS12, target, same, helloStdlibTLS12(target.Hello), true)
		}
	}
}

// alternateHello is another Signal client's hello: Android's REST hello for targets that use
// libsignal's or Desktop's, libsignal's for targets that use an Android hello.
func alternateHello(h Hello) Hello {
	if h.libsignal || h.Name == HelloDesktop.Name {
		return HelloAndroidREST
	}
	return HelloLibsignal
}

func (cr *controlRunner) planDNS(r *TargetResult, kind, host string) *controlJob {
	cr.add(r, kind+"|"+host, kind, func(ctx context.Context, job *controlJob) *ControlResult {
		if kind == ControlDoH {
			return cr.doh(ctx, job, host)
		}
		return cr.udp(ctx, job, host)
	})
	return cr.byKey[kind+"|"+host]
}

func (cr *controlRunner) planHandshake(r *TargetResult, kind string, target Target, a staticAddress, hello Hello, verify bool) {
	key := strings.Join([]string{kind, a.ip, target.Hostname, hello.Name}, "|")
	cr.add(r, key, kind, func(ctx context.Context, job *controlJob) *ControlResult {
		return cr.handshake(ctx, job, kind, target, a, hello, verify)
	})
}

// planStaticFromDoH plans the static_address control to the first IPv4 answer of doh, when that
// answer is in no static list and was not tried. It waits for doh; if an identical control
// exists by then, r is added to it instead.
func (cr *controlRunner) planStaticFromDoH(r *TargetResult, target Target, doh *controlJob, list []staticAddress, tried string) {
	job := &controlJob{index: cr.index, kind: ControlStatic, done: make(chan struct{})}
	cr.index++
	job.run = func(ctx context.Context, job *controlJob) *ControlResult {
		select {
		case <-doh.done:
		case <-ctx.Done():
			return nil
		}
		if doh.result == nil || doh.result.Addresses == nil {
			return nil
		}
		ip := ""
		for _, a := range *doh.result.Addresses {
			if net.ParseIP(a).To4() != nil {
				ip = a
				break
			}
		}
		if ip == "" || ip == tried || inList(list, ip) {
			return nil
		}
		a := staticAddress{ip: ip, source: "doh"}
		key := strings.Join([]string{ControlStatic, ip, target.Hostname, target.Hello.Name}, "|")
		cr.mu.Lock()
		if other, ok := cr.byKey[key]; ok {
			cr.mu.Unlock()
			other.addTarget(r) // recorded with the other's result, if it ran
			return nil
		}
		cr.byKey[key] = job
		cr.mu.Unlock()
		return cr.handshake(ctx, job, ControlStatic, target, a, target.Hello, true)
	}
	job.addTarget(r)
	cr.jobs = append(cr.jobs, job)
}

// run starts the controls in order, at most controlsParallelism at a time, within
// controlsBudget, and records them. It returns whether some were never started.
func (cr *controlRunner) run(ctx context.Context, tk *TestKeys) (truncated bool) {
	ctx, cancel := context.WithTimeout(ctx, controlsBudget)
	defer cancel()
	sem := make(chan struct{}, controlsParallelism)
	var wg sync.WaitGroup
	for i, job := range cr.jobs {
		select {
		case sem <- struct{}{}:
		case <-ctx.Done():
		}
		if ctx.Err() != nil {
			truncated = true
			for _, j := range cr.jobs[i:] {
				close(j.done) // never started: not recorded
			}
			break
		}
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() { <-sem }()
			job.result = job.run(ctx, job)
			close(job.done)
		}()
	}
	wg.Wait()

	for _, job := range cr.jobs {
		tk.Queries = append(tk.Queries, job.queries...)
		tk.TCPConnect = append(tk.TCPConnect, job.tcpConnect...)
		tk.TLSHandshakes = append(tk.TLSHandshakes, job.handshakes...)
		tk.NetworkEvents = append(tk.NetworkEvents, job.events...)
		if job.result == nil {
			continue
		}
		for _, r := range job.targets {
			r.Controls = append(r.Controls, job.result)
		}
	}
	return truncated
}

// doh resolves host with DNS-over-HTTPS as libsignal does: A and AAAA, by POST over HTTP/2 to
// Cloudflare, with Mozilla's roots; the IPv6 resolver only if the IPv4 one fails.
func (cr *controlRunner) doh(ctx context.Context, job *controlJob, host string) *ControlResult {
	res := &ControlResult{Kind: ControlDoH, T0: time.Since(cr.zero).Seconds()}
	trace := job.newTrace(cr.zero)
	defer job.collect(trace)
	for _, url := range dohResolvers {
		res.Resolver = url
		qctx, cancel := context.WithTimeout(ctx, dohTimeout)
		reso := trace.NewParallelDNSOverHTTPSResolver(cr.logger, url)
		addrs, err := reso.LookupHost(qctx, host)
		reso.CloseIdleConnections()
		cancel()
		answers := append([]string{}, addrs...)
		res.Addresses, res.Failure, res.FailedOperation = &answers, nil, ""
		if err == nil {
			break
		}
		res.Failure, res.FailedOperation = measurexlite.NewFailure(err), netxlite.ResolveOperation
		if strings.HasPrefix(*res.Failure, "dns_") { // the resolver answered: no fallback
			break
		}
	}
	res.T = time.Since(cr.zero).Seconds()
	return res
}

// udp sends an A query for host to 1.1.1.1:53 over UDP, as Signal-Android's REST clients do.
func (cr *controlRunner) udp(ctx context.Context, job *controlJob, host string) *ControlResult {
	res := &ControlResult{Kind: ControlUDP, Resolver: udpResolver, T0: time.Since(cr.zero).Seconds()}
	ctx, cancel := context.WithTimeout(ctx, udpTimeout)
	defer cancel()
	trace := job.newTrace(cr.zero)
	defer job.collect(trace)
	txp := netxlite.NewUnwrappedDNSOverUDPTransport(trace.NewDialerWithoutResolver(cr.logger), udpResolver)
	query := (&netxlite.DNSEncoderMiekg{}).Encode(host, dns.TypeA, txp.RequiresPadding())
	started := time.Since(cr.zero)
	resp, err := txp.RoundTrip(ctx, query)
	var addrs []string
	if err == nil {
		addrs, err = resp.DecodeLookupHost()
	}
	if err != nil {
		err = netxlite.NewErrWrapper(netxlite.ClassifyResolverError, netxlite.ResolveOperation, err)
		res.Failure, res.FailedOperation = measurexlite.NewFailure(err), netxlite.ResolveOperation
	}
	answers := append([]string{}, addrs...)
	res.Addresses = &answers
	job.queries = append(job.queries, measurexlite.NewArchivalDNSLookupResultFromRoundTrip(trace.Index(), started,
		txp, query, resp, addrs, err, time.Since(cr.zero), trace.Tags()...))
	res.T = time.Since(cr.zero).Seconds()
	return res
}

// handshake runs a handshake control to address a. static_address controls then run the
// target's method up to the upgrade or GET response.
func (cr *controlRunner) handshake(ctx context.Context, job *controlJob, kind string, target Target, a staticAddress,
	hello Hello, verify bool) *ControlResult {
	sni := target.Hostname
	if kind == ControlDecoySNI {
		sni = decoySNI(len(sni))
	}
	address := net.JoinHostPort(a.ip, "443")
	zeroStatus, noTimestamp := int64(0), false
	res := &ControlResult{Kind: kind, Address: address, AddressSource: a.source, SNI: sni, ClientHello: hello.Name,
		Verified: &verify, Status: &zeroStatus, SignalTimestamp: &noTimestamp, T0: time.Since(cr.zero).Seconds()}
	defer func() { res.T = time.Since(cr.zero).Seconds() }()
	ctx, cancel := context.WithTimeout(ctx, controlHandshakeTimeout)
	defer cancel()
	trace := job.newTrace(cr.zero)
	defer job.collect(trace)
	var roots *x509.CertPool
	if verify {
		roots = cr.roots
	}
	tr, err := dialTLS(ctx, trace, cr.logger, cr.zero, address, sni, hello, roots, trace.Tags()...)
	if tr != nil && tr.handshake != nil {
		job.handshakes = append(job.handshakes, tr.handshake)
	}
	if err != nil {
		res.Failure, res.FailedOperation = measurexlite.NewFailure(err), netxlite.TLSHandshakeOperation
		if tr == nil {
			res.FailedOperation = netxlite.ConnectOperation
		}
		return res
	}
	res.LeafKeyType = tr.leafKey
	if kind != ControlStatic {
		_ = tr.conn.Close()
		return res
	}
	tres := &TargetResult{UserAgent: cr.m.config.userAgent(target.Hello)}
	cr.m.exerciseInto(ctx, ctx, tr, target, target.Hostname, target.Path, tres, cr.zero, nil)
	*res.Status, *res.SignalTimestamp = tres.Status, tres.SignalTimestamp
	res.Failure, res.FailedOperation = tres.Failure, tres.FailedOperation
	return res
}

// decoySNI returns a random lowercase name under example.com (RFC 2606) of length n, so that a
// hello carrying it differs from the real one only in the SNI's bytes.
func decoySNI(n int) string {
	const suffix, letters = ".example.com", "abcdefghijklmnopqrstuvwxyz"
	label := make([]byte, max(n-len(suffix), 1))
	for i := range label {
		c, err := rand.Int(rand.Reader, big.NewInt(int64(len(letters))))
		if err != nil {
			label[i] = 'x'
			continue
		}
		label[i] = letters[c.Int64()]
	}
	return string(label) + suffix
}
