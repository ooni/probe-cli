package signalclient

import (
	"context"
	"crypto/x509"
	"errors"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ooni/probe-cli/v3/internal/logx"
	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

const (
	testName    = "signal_client" // provisional name, see doc.go
	testVersion = "0.2.0"
)

// Method is how a target is exercised after TLS.
type Method string

const (
	// MethodProvisioningH2 is libsignal's RFC 8441 extended CONNECT to the provisioning
	// WebSocket, followed by the provisioning flow.
	MethodProvisioningH2 Method = "provisioning_h2"

	// MethodProvisioningH1 is Android OkHttp's HTTP/1.1 WebSocket upgrade to the provisioning
	// WebSocket, followed by the provisioning flow.
	MethodProvisioningH1 Method = "provisioning_h1"

	// MethodEnclaveH1 is libsignal's HTTP/1.1 WebSocket upgrade to an enclave, without
	// credentials: 401 (current enclave id) or 404 (other ids) both mean Signal answered.
	MethodEnclaveH1 Method = "enclave_h1"

	// MethodHTTPSGet is one HTTPS GET of "/" (HTTP/2 when negotiated).
	MethodHTTPSGet Method = "https_get"
)

// Target is one connection a real Signal client makes.
type Target struct {
	Name     string
	Hostname string
	Hello    Hello
	Method   Method
	Path     string
}

// enclaveProbePath is a syntactically valid enclave path that names no real enclave, so the
// result never depends on the quarterly enclave ids.
const enclaveProbePath = "/v1/0000000000000000000000000000000000000000000000000000000000000000"

// DefaultTargets are Signal's Signal-hosted endpoints, each with the hello and protocol of the
// client that uses it (signal.md §1; hellos/README.md).
var DefaultTargets = []Target{
	{"chat_libsignal", "grpc.chat.signal.org", HelloLibsignal, MethodProvisioningH2, "/v1/websocket/provisioning/"},
	{"chat_android", "chat.signal.org", HelloAndroidWebSocket, MethodProvisioningH1, "/v1/websocket/provisioning/"},
	{"cdsi", "cdsi.signal.org", HelloLibsignalEnclave, MethodEnclaveH1, enclaveProbePath + "/discovery"},
	{"svr2", "svr2.signal.org", HelloLibsignalEnclave, MethodEnclaveH1, enclaveProbePath},
	{"svrb", "svrb.signal.org", HelloLibsignalEnclave, MethodEnclaveH1, enclaveProbePath},
	{"storage", "storage.signal.org", HelloAndroidREST, MethodHTTPSGet, "/"},
	{"cdn", "cdn.signal.org", HelloAndroidREST, MethodHTTPSGet, "/"},
	{"cdn2", "cdn2.signal.org", HelloAndroidREST, MethodHTTPSGet, "/"},
	{"cdn3", "cdn3.signal.org", HelloAndroidREST, MethodHTTPSGet, "/"},
	{"updates2", "updates2.signal.org", HelloAndroidREST, MethodHTTPSGet, "/"},
	{"updates2_desktop", "updates2.signal.org", HelloDesktop, MethodHTTPSGet, "/desktop/latest-linux.yml"},
	{"sfu", "sfu.voip.signal.org", HelloAndroidREST, MethodHTTPSGet, "/"},
}

// keepalivePaddings are the keepalive body sizes: none, message-sized, attachment-chunk-sized.
var keepalivePaddings = []int{0, 1024, 16384}

// Config contains the experiment config.
type Config struct {
	// IdleSeconds keeps each provisioning socket idle this long before the keepalives, to detect
	// middleboxes that kill idle sessions. It is capped at maxIdleSeconds.
	IdleSeconds int64 `ooni:"seconds of idle time on provisioning sockets before keepalives (0 = none)"`

	// AppUserAgent sends each real client's User-Agent instead of identifying as OONI Probe.
	// By default the test identifies itself (see ooniUserAgent), so that Signal can recognize
	// and filter its traffic. The User-Agent travels inside TLS: either way, the network sees
	// the same bytes.
	AppUserAgent bool `ooni:"send the Signal apps' User-Agents instead of identifying as OONI Probe"`

	// UserAgent, when set, is sent instead of either.
	UserAgent string `ooni:"User-Agent to send instead of the default"`

	// SignalCA, when set, is a PEM bundle that replaces Signal's two built-in roots for every
	// connection verified against Signal's roots (targets, controls, inner connections of the
	// circumvention routes), for a root rotation that ships before a probe release or a private
	// Signal server. Not for the fronts or DNS-over-HTTPS.
	SignalCA string `ooni:"PEM bundle replacing Signal's two built-in roots"`

	// Controls says when the address controls run: on_failure (default), always or off.
	Controls string `ooni:"when the address controls run: on_failure (default), always or off"`

	// StaticAddresses replaces the built-in static addresses of the named hosts:
	// "host=addr,addr;host=addr".
	StaticAddresses string `ooni:"replace static addresses: host=addr,addr;host=addr"`

	// Circumvention lists the circumvention routes to measure: a comma-separated subset of
	// reflector_f, reflector_g, android_g, android_f, or all. Empty: none.
	Circumvention string `ooni:"circumvention routes: comma-separated reflector_f,reflector_g,android_g,android_f or all"`

	// CircumventionCountry is the country whose Android front configuration android_g and
	// android_f reproduce (default: the probe's country).
	CircumventionCountry string `ooni:"country of the Android front configuration (default: the probe's)"`

	// Fronts replaces the built-in front configuration of the named routes:
	// "route[.service]=host_header@sni,sni;...".
	Fronts string `ooni:"replace fronts: route[.service]=host_header@sni,sni;..."`
}

// ooniUserAgent identifies this test to Signal's servers.
const ooniUserAgent = "ooniprobe-" + testName + "/" + testVersion + " (+https://ooni.org)"

// userAgent is the User-Agent to send with hello.
func (c *Config) userAgent(hello Hello) string {
	switch {
	case c.UserAgent != "":
		return c.UserAgent
	case c.AppUserAgent && hello.appUserAgent != "":
		return hello.appUserAgent
	case hello.libsignal: // libsignal appends its own token, as it does for the apps
		return ooniUserAgent + " libsignal/" + libsignalVersion
	default:
		return ooniUserAgent
	}
}

// TargetResult is the outcome for one target.
type TargetResult struct {
	Name            string              `json:"name"`
	Hostname        string              `json:"hostname"`
	Address         *string             `json:"address"` // null inside a reflector tunnel
	ClientHello     string              `json:"client_hello"`
	Method          Method              `json:"method"`
	UserAgent       string              `json:"user_agent"`
	ALPN            string              `json:"alpn"`
	LeafKeyType     string              `json:"leaf_key_type"`
	Status          int64               `json:"status"`
	SignalTimestamp bool                `json:"x_signal_timestamp"`
	WSExtensions    string              `json:"ws_extensions,omitempty"`
	Provisioning    *ProvisioningResult `json:"provisioning,omitempty"`
	Response        *ResponseSample     `json:"response,omitempty"` // https_get, or a refused upgrade
	Failure         *string             `json:"failure"`
	FailedOperation string              `json:"failed_operation,omitempty"`
	Controls        []*ControlResult    `json:"controls"`
	T0              float64             `json:"t0"`
	T               float64             `json:"t"`
}

// TestKeys contains the experiment results.
type TestKeys struct {
	NetworkEvents []*model.ArchivalNetworkEvent             `json:"network_events"`
	Queries       []*model.ArchivalDNSLookupResult          `json:"queries"`
	TCPConnect    []*model.ArchivalTCPConnectResult         `json:"tcp_connect"`
	TLSHandshakes []*model.ArchivalTLSOrQUICHandshakeResult `json:"tls_handshakes"`
	Targets       []*TargetResult                           `json:"signal_targets"`

	// SignalServiceStatus is Signal's own outage beacon: uptime.signal.org resolves to
	// 127.0.0.1 when the service is up and 127.0.0.2 during an outage.
	SignalServiceStatus string `json:"signal_service_status"`

	// SignalRoots is the SHA-256 of each root used for Signal's hosts (lowercase hex of the DER):
	// the two built-in roots unless SignalCA replaced them.
	SignalRoots []string `json:"signal_roots"`

	// ControlsTruncated is true when the test's clock ran out before all controls were started.
	ControlsTruncated bool `json:"signal_controls_truncated"`

	// Circumvention holds the circumvention routes (null unless Circumvention is set).
	Circumvention []*RouteResult `json:"signal_circumvention"`
}

// Measurer performs the measurement.
type Measurer struct {
	config  Config
	targets []Target

	// trust anchors (tests replace these)
	roots        func() ([]*x509.Certificate, error) // Signal's
	googleRoots  func() (*x509.CertPool, error)      // the Google fronts'
	mozillaRoots func() (*x509.CertPool, error)      // the Fastly fronts'
}

// NewExperimentMeasurer creates a new ExperimentMeasurer.
func NewExperimentMeasurer(config Config) model.ExperimentMeasurer {
	return &Measurer{config: config, targets: DefaultTargets, roots: signalRoots, googleRoots: googleRoots,
		mozillaRoots: mozillaRoots}
}

// runOptions are the parsed options of one run.
type runOptions struct {
	controls string
	static   map[string][]staticAddress
	routes   []string
	country  string
	fronts   map[string]frontConfig
}

// parseOptions parses and validates the options. probeCC is the default CircumventionCountry.
func (m *Measurer) parseOptions(probeCC string) (*runOptions, error) {
	controls, err := parseControls(m.config.Controls)
	if err != nil {
		return nil, err
	}
	static := builtinStaticAddresses()
	override, err := parseStaticAddresses(m.config.StaticAddresses)
	if err != nil {
		return nil, err
	}
	for host, list := range override {
		static[host] = list
	}
	routes, err := parseCircumvention(m.config.Circumvention)
	if err != nil {
		return nil, err
	}
	country := strings.ToUpper(strings.TrimSpace(m.config.CircumventionCountry))
	if country == "" {
		country = strings.ToUpper(probeCC)
	}
	fronts := builtinFronts(country)
	if err := parseFronts(m.config.Fronts, fronts); err != nil {
		return nil, err
	}
	return &runOptions{controls: controls, static: static, routes: routes, country: country, fronts: fronts}, nil
}

// signalAnchors returns Signal's roots, or SignalCA's certificates when set.
func (m *Measurer) signalAnchors() (*trustAnchors, error) {
	if m.config.SignalCA != "" {
		certs, err := parsePEMBundle(m.config.SignalCA)
		if err != nil {
			return nil, err
		}
		return newTrustAnchors(certs), nil
	}
	certs, err := m.roots()
	if err != nil {
		return nil, err
	}
	return newTrustAnchors(certs), nil
}

// ExperimentName implements model.ExperimentMeasurer.
func (m *Measurer) ExperimentName() string { return testName }

// ExperimentVersion implements model.ExperimentMeasurer.
func (m *Measurer) ExperimentVersion() string { return testVersion }

// Run implements model.ExperimentMeasurer.
func (m *Measurer) Run(ctx context.Context, args *model.ExperimentArgs) error {
	logger := args.Session.Logger()
	zero := args.Measurement.MeasurementStartTimeSaved
	tk := &TestKeys{}
	args.Measurement.TestKeys = tk
	opts, err := m.parseOptions(args.Measurement.ProbeCC)
	if err != nil {
		return err
	}
	anchors, err := m.signalAnchors()
	if err != nil {
		return err
	}
	tk.SignalRoots = anchors.sha256
	roots := anchors.pool

	// One clock for the whole test (testTimeout). It cancels rather than times out, and whatever
	// it cuts short is recorded as interrupted (markInterrupted), never as a network timeout.
	runCtx, stop := context.WithCancel(ctx)
	defer stop()
	var firedAt atomic.Value // seconds since zero, when the clock ran out
	clock := time.AfterFunc(testTimeout, func() {
		firedAt.Store(time.Since(zero).Seconds())
		stop()
	})
	defer clock.Stop()
	progress := newProgress(args.Callbacks, len(m.targets), len(opts.routes))

	// Everything starts at once: the beacon, the targets (concurrently, as the apps connect to
	// them) and the routes. Each target and route records into its own TestKeys, merged in order.
	var wg sync.WaitGroup
	beacon := &TestKeys{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		m.measureBeacon(runCtx, logger, zero, beacon)
		progress.step("signal_client: Signal's service status: " + beacon.SignalServiceStatus)
	}()

	tk.Targets = make([]*TargetResult, len(m.targets))
	records := make([]*TestKeys, len(m.targets))
	connected := make([]*TargetResult, len(m.targets)) // each target's result when it finished connecting
	var connecting sync.WaitGroup
	for i, target := range m.targets {
		records[i] = &TestKeys{}
		res := newTargetResult(target, zero)
		tk.Targets[i] = res
		wg.Add(1)
		connecting.Add(1)
		go func() {
			defer wg.Done()
			var once sync.Once
			markConnected := func() {
				once.Do(func() {
					snapshot := *res
					connected[i] = &snapshot
					connecting.Done()
				})
			}
			defer markConnected()
			m.measureTarget(runCtx, logger, zero, int64(i+1), target, roots, res, records[i], markConnected)
			progress.step("signal_client: " + target.Name)
		}()
	}

	if len(opts.routes) > 0 {
		tk.Circumvention = make([]*RouteResult, len(opts.routes))
	}
	routeRecords := make([]*TestKeys, len(opts.routes))
	for i, route := range opts.routes {
		routeRecords[i] = &TestKeys{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			index := int64(10000 + 10*i)
			tk.Circumvention[i] = m.measureRoute(runCtx, logger, zero, index, route, opts, roots, routeRecords[i])
			progress.step("signal_client: route " + route)
		}()
	}

	// The controls start once every target has connected or failed to (DNS, TCP, TLS and the
	// upgrade or GET), while provisioning flows may still run. They decide from those snapshots.
	controls := &TestKeys{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		connecting.Wait()
		if opts.controls != ControlsOff {
			cr := newControlRunner(m, logger, zero, roots, opts, int64(len(m.targets)+1))
			cr.plan(m.targets, connected, tk.Targets)
			controls.ControlsTruncated = cr.run(runCtx, controls)
		}
		progress.step("signal_client: controls")
	}()

	wg.Wait()
	clock.Stop()
	tk.SignalServiceStatus = beacon.SignalServiceStatus
	tk.mergeRecords(beacon)
	for _, r := range records {
		tk.mergeRecords(r)
	}
	tk.ControlsTruncated = controls.ControlsTruncated
	tk.mergeRecords(controls)
	for _, r := range routeRecords {
		tk.mergeRecords(r)
	}
	if t, ok := firedAt.Load().(float64); ok {
		tk.markInterrupted(t)
	}
	return nil
}

// measureBeacon resolves Signal's outage beacon, uptime.signal.org, with the system resolver:
// 127.0.0.1 when the service is up, 127.0.0.2 during an outage (ts-041 Service status control).
func (m *Measurer) measureBeacon(ctx context.Context, logger model.Logger, zero time.Time, tk *TestKeys) {
	ctx, cancel := context.WithTimeout(ctx, opTimeout)
	defer cancel()
	trace := measurexlite.NewTrace(0, zero)
	addrs, err := trace.NewStdlibResolver(logger).LookupHost(ctx, "uptime.signal.org")
	tk.Queries = append(tk.Queries, trace.DNSLookupsFromRoundTrip()...)
	switch {
	case err != nil:
		tk.SignalServiceStatus = *measurexlite.NewFailure(err)
	case contains(addrs, "127.0.0.1"):
		tk.SignalServiceStatus = "ok"
	case contains(addrs, "127.0.0.2"):
		tk.SignalServiceStatus = "outage"
	default:
		tk.SignalServiceStatus = "unexpected_answer"
	}
}

// markInterrupted records as interrupted every failure that ended after the test's clock ran
// out at firedAt (seconds since the start): they are the clock's, not the network's.
func (tk *TestKeys) markInterrupted(firedAt float64) {
	interrupted := netxlite.FailureInterrupted
	mark := func(failure **string, t float64) {
		if *failure != nil && t >= firedAt {
			*failure = &interrupted
		}
	}
	markTarget := func(r *TargetResult) {
		if r == nil {
			return
		}
		mark(&r.Failure, r.T)
		if r.Provisioning != nil {
			mark(&r.Provisioning.Failure, r.T)
		}
		for _, c := range r.Controls {
			mark(&c.Failure, c.T)
		}
	}
	for _, r := range tk.Targets {
		markTarget(r)
	}
	for _, route := range tk.Circumvention {
		if route == nil {
			continue
		}
		mark(&route.Failure, route.T)
		for _, r := range route.Targets {
			markTarget(r)
		}
	}
}

// mergeRecords appends src's network records to tk's.
func (tk *TestKeys) mergeRecords(src *TestKeys) {
	tk.Queries = append(tk.Queries, src.Queries...)
	tk.TCPConnect = append(tk.TCPConnect, src.TCPConnect...)
	tk.TLSHandshakes = append(tk.TLSHandshakes, src.TLSHandshakes...)
	tk.NetworkEvents = append(tk.NetworkEvents, src.NetworkEvents...)
}

// progress reports the measurement's progress: the beacon, each target, the controls and each
// route are one step each. Steps may finish concurrently.
type progress struct {
	mu        sync.Mutex
	callbacks model.ExperimentCallbacks
	done      int
	total     int
}

func newProgress(callbacks model.ExperimentCallbacks, targets, routes int) *progress {
	return &progress{callbacks: callbacks, total: 1 + targets + 1 + routes}
}

func (p *progress) step(message string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.done++
	p.callbacks.OnProgress(float64(p.done)/float64(p.total), message)
}

// newTargetResult is a target's result before it is measured.
func newTargetResult(target Target, zero time.Time) *TargetResult {
	address := ""
	return &TargetResult{Name: target.Name, Hostname: target.Hostname, Address: &address, ClientHello: target.Hello.Name,
		Method: target.Method, T0: time.Since(zero).Seconds()}
}

// measureTarget measures target into res, recording its network events in tk. Each operation
// (DNS, TCP and TLS, the upgrade or GET, each provisioning step) has opTimeout; the test's clock
// bounds the whole. It calls connected once the target has connected or failed to, before any
// provisioning flow.
func (m *Measurer) measureTarget(ctx context.Context, logger model.Logger, zero time.Time, index int64,
	target Target, roots *x509.CertPool, res *TargetResult, tk *TestKeys, connected func()) {
	defer func() { res.T = time.Since(zero).Seconds() }()
	fail := func(op string, err error) {
		res.FailedOperation, res.Failure = op, measurexlite.NewFailure(err)
	}

	trace := measurexlite.NewTrace(index, zero, target.Name)
	ol := logx.NewOperationLogger(logger, "signal_client: %s %s", target.Name, target.Hostname)
	defer func() { ol.Stop(errorOrNil(res.Failure)) }()

	resolveCtx, cancelResolve := context.WithTimeout(ctx, opTimeout)
	addrs, err := trace.NewStdlibResolver(logger).LookupHost(resolveCtx, target.Hostname)
	cancelResolve()
	tk.Queries = append(tk.Queries, trace.DNSLookupsFromRoundTrip()...)
	if err != nil {
		fail(netxlite.ResolveOperation, err)
		return
	}
	ip := firstIPv4(addrs)
	if ip == "" {
		fail(netxlite.ResolveOperation, errNoAddress)
		return
	}
	address := net.JoinHostPort(ip, "443")
	*res.Address = address

	dialCtx, cancelDial := context.WithTimeout(ctx, opTimeout)
	tr, err := dialTLS(dialCtx, trace, logger, address, target.Hostname, target.Hello, roots)
	cancelDial()
	tk.recordTLS(trace, tr)
	defer func() { tk.NetworkEvents = append(tk.NetworkEvents, trace.NetworkEvents()...) }()
	if err != nil {
		fail(failedOperation(err), err)
		return
	}
	res.ALPN, res.LeafKeyType = tr.alpn, tr.leafKey
	res.UserAgent = m.config.userAgent(target.Hello)
	// the first checkpoint follows the upgrade, before the provisioning flow: connected then
	drain := func() {
		connected()
		tk.NetworkEvents = append(tk.NetworkEvents, trace.NetworkEvents()...) // the buffer holds 64
	}
	opCtx, cancelOp := context.WithTimeout(ctx, opTimeout)
	defer cancelOp()
	m.exercise(ctx, opCtx, tr, target.request(), zero, res, drain)
}

// request is what exercise sends on a target's TLS connection.
type request struct {
	method     Method
	hostHeader string // the target's hostname, or a front's Host header
	path       string
	provision  bool // run the provisioning flow after an accepted upgrade (else stop there)
}

// request is the target's own request, with the provisioning flow.
func (t Target) request() request {
	return request{method: t.Method, hostHeader: t.Hostname, path: t.Path, provision: true}
}

// exercise runs req on an established TLS connection and records the outcome in res, closing
// the connection when done. opCtx bounds the upgrade or GET; ctx, the stream and the provisioning
// flow, which calls checkpoint after each step.
func (m *Measurer) exercise(ctx, opCtx context.Context, tr *tlsResult, req request, zero time.Time,
	res *TargetResult, checkpoint func()) {
	defer func() { _ = tr.conn.Close() }()
	// when ctx ends, close the conn: this unblocks any read or write, including HTTP/2 writes
	// waiting for flow control, which do not watch ctx themselves
	stopCloser := context.AfterFunc(ctx, func() { _ = tr.conn.Close() })
	defer stopCloser()
	fail := func(op string, err error) {
		res.FailedOperation, res.Failure = op, measurexlite.NewFailure(err)
	}
	ua := res.UserAgent
	switch req.method {
	case MethodHTTPSGet:
		status, ts, sample, err := httpsGet(opCtx, tr.conn, tr.alpn, req.hostHeader, req.path, ua)
		res.Status, res.SignalTimestamp, res.Response = int64(status), ts, sample
		if err != nil {
			fail(netxlite.HTTPRoundTripOperation, err)
		}
	case MethodEnclaveH1, MethodProvisioningH1, MethodProvisioningH2:
		var (
			up  *upgradeResult
			err error
		)
		if req.method == MethodProvisioningH2 {
			up, err = websocketH2(ctx, opCtx, tr.conn, req.hostHeader, req.path, ua)
		} else {
			up, err = websocketH1(opCtx, tr.conn, req.hostHeader, req.path, ua)
		}
		if up != nil {
			res.Status, res.SignalTimestamp, res.WSExtensions = int64(up.status), up.signalTimestamp, up.extensions
			res.Response = up.response
		}
		if req.method == MethodEnclaveH1 {
			// no credentials: 401 or 404 from the enclave front end is the expected answer
			if errors.Is(err, errUpgradeRefused) && (res.Status == 401 || res.Status == 404) {
				return
			}
			if err == nil { // accepted without credentials: unexpected, but Signal answered
				_ = up.closer.Close()
				return
			}
			fail(websocketUpgradeOperation, err)
			return
		}
		if err != nil {
			fail(websocketUpgradeOperation, err)
			return
		}
		defer up.closer.Close()
		if !req.provision {
			return
		}
		checkpoint()
		res.Provisioning = runProvisioning(ctx, up.ws, zero, keepalivePaddings, idleDuration(m.config.IdleSeconds), checkpoint)
		if res.Provisioning.Failure != nil {
			res.Failure, res.FailedOperation = res.Provisioning.Failure, res.Provisioning.FailedStep
		}
	default:
		fail(netxlite.TopLevelOperation, errors.New("signal_client: unknown method"))
	}
}

// Time limits (ts-041 Time limits; variables so tests can shorten them).
var (
	// testTimeout is the whole test's clock.
	testTimeout = 60 * time.Second

	// opTimeout bounds each network operation of a target or route: DNS, TCP and TLS, the
	// upgrade or GET (provisioning steps have their own, also 10 s).
	opTimeout = 10 * time.Second
)

// maxIdleSeconds bounds IdleSeconds. Signal's connections are never idle longer than about 30 s
// (Signal-Android's provisioning keepalive every 30 s; libsignal pings after 31 s idle), so a
// middlebox that kills idle sessions only matters to Signal if it kills them within that.
const maxIdleSeconds = 30

// idleDuration converts IdleSeconds, clamped to [0, maxIdleSeconds].
func idleDuration(seconds int64) time.Duration {
	return time.Duration(min(max(seconds, 0), maxIdleSeconds)) * time.Second
}

func firstIPv4(addrs []string) string {
	for _, a := range addrs {
		if ip := net.ParseIP(a); ip != nil && ip.To4() != nil {
			return a
		}
	}
	if len(addrs) > 0 {
		return addrs[0]
	}
	return ""
}

func contains(xs []string, x string) bool {
	for _, v := range xs {
		if v == x {
			return true
		}
	}
	return false
}

func errorOrNil(failure *string) error {
	if failure == nil {
		return nil
	}
	return errors.New(*failure)
}
