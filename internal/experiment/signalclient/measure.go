package signalclient

import (
	"context"
	"crypto/x509"
	"errors"
	"net"
	"strings"
	"sync"
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
	// IdleSeconds keeps each provisioning socket idle this long before the keepalives,
	// to detect middleboxes that kill idle sessions (libsignal pings after 31 s, drops after 45 s).
	// It is capped at maxIdleSeconds: Signal-Server closes provisioning sockets 90 s after they open.
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
		return ooniUserAgent + " libsignal/0.105.0"
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

	// ControlsTruncated is true when the controls' time budget ran out before all were started.
	ControlsTruncated bool `json:"signal_controls_truncated"`

	// RunTruncated is true when the whole measurement's budget (runBudget) ran out, cutting short
	// whatever was still running. Each phase has its own limits, so this is only a backstop.
	RunTruncated bool `json:"signal_run_truncated"`

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

// Experiment-specific failure strings (in addition to df-007's).
const (
	// FailureExtendedConnectNotSupported: the HTTP/2 server did not offer RFC 8441 extended CONNECT.
	FailureExtendedConnectNotSupported = "extended_connect_not_supported"

	// FailureWebSocketUpgradeRefused: the server answered the WebSocket upgrade or extended
	// CONNECT with a status other than success (see the target's status).
	FailureWebSocketUpgradeRefused = "websocket_upgrade_refused"

	// FailureWebSocketUpgradeInvalid: the server answered the HTTP/1.1 upgrade with 101 but without
	// the Upgrade, Connection or Sec-WebSocket-Accept headers RFC 6455 requires, which the apps'
	// WebSocket clients reject.
	FailureWebSocketUpgradeInvalid = "websocket_upgrade_invalid"

	// FailureUnexpectedMessage: the server sent a WebSocket message the flow does not allow.
	FailureUnexpectedMessage = "unexpected_message"

	// FailureMalformedMessage: a WebSocket message was not a valid WebSocketMessage protobuf, could
	// not be inflated, or exceeded the size limit, or a frame violated RFC 6455 or RFC 7692.
	FailureMalformedMessage = "malformed_message"

	// FailureHTTP2StreamReset: the peer reset the HTTP/2 stream carrying the WebSocket.
	FailureHTTP2StreamReset = "http2_stream_reset"
)

// newFailure wraps err so that it is recorded with the given failure string.
func newFailure(failure, operation string, err error) error {
	return &netxlite.ErrWrapper{Failure: failure, Operation: operation, WrappedErr: err}
}

// errNoAddress means DNS returned no usable address.
var errNoAddress = newFailure(netxlite.FailureDNSNoAnswer, netxlite.ResolveOperation, errors.New("no usable address"))

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

	// the whole measurement's budget, a backstop: each phase below has its own limits
	runCtx, cancelRun := context.WithTimeout(ctx, runBudget(idleDuration(m.config.IdleSeconds), len(opts.routes) > 0))
	defer cancelRun()
	defer func() { tk.RunTruncated = runCtx.Err() == context.DeadlineExceeded && ctx.Err() == nil }()
	progress := newProgress(args.Callbacks, len(m.targets), len(opts.routes))

	// Signal's outage beacon, a control: if Signal says it is down, failures mean little.
	trace := measurexlite.NewTrace(0, zero)
	beaconCtx, cancelBeacon := context.WithTimeout(runCtx, beaconTimeout)
	addrs, err := trace.NewStdlibResolver(logger).LookupHost(beaconCtx, "uptime.signal.org")
	cancelBeacon()
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

	progress.step("signal_client: Signal's service status: " + tk.SignalServiceStatus)

	// the targets run concurrently, as the apps connect to them: the phase lasts as long as the
	// slowest target, not the sum. Each records into its own TestKeys, merged in target order.
	tk.Targets = make([]*TargetResult, len(m.targets))
	records := make([]*TestKeys, len(m.targets))
	var wg sync.WaitGroup
	for i, target := range m.targets {
		records[i] = &TestKeys{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			tk.Targets[i] = m.measureTarget(runCtx, logger, zero, int64(i+1), target, roots, records[i])
			progress.step("signal_client: " + target.Name)
		}()
	}
	wg.Wait()
	for _, r := range records {
		tk.mergeRecords(r)
	}

	// the controls run after every target, so they never delay or change a target's result
	if opts.controls != ControlsOff {
		cr := newControlRunner(m, logger, zero, roots, opts, int64(len(m.targets)+1))
		cr.plan(m.targets, tk.Targets)
		tk.ControlsTruncated = cr.run(runCtx, tk)
	}
	progress.step("signal_client: controls")

	// then the circumvention routes, concurrently, which never change what the targets say
	if len(opts.routes) > 0 {
		tk.Circumvention = make([]*RouteResult, len(opts.routes))
		records := make([]*TestKeys, len(opts.routes))
		for i, route := range opts.routes {
			records[i] = &TestKeys{}
			wg.Add(1)
			go func() {
				defer wg.Done()
				index := int64(10000 + 10*i)
				tk.Circumvention[i] = m.measureRoute(runCtx, logger, zero, index, route, opts, roots, records[i])
				progress.step("signal_client: route " + route)
			}()
		}
		wg.Wait()
		for _, r := range records {
			tk.mergeRecords(r)
		}
	}
	return nil
}

// mergeRecords appends src's network records to tk's.
func (tk *TestKeys) mergeRecords(src *TestKeys) {
	tk.Queries = append(tk.Queries, src.Queries...)
	tk.TCPConnect = append(tk.TCPConnect, src.TCPConnect...)
	tk.TLSHandshakes = append(tk.TLSHandshakes, src.TLSHandshakes...)
	tk.NetworkEvents = append(tk.NetworkEvents, src.NetworkEvents...)
}

// beaconTimeout bounds the outage beacon's DNS lookup.
var beaconTimeout = 10 * time.Second

// runMargin is the slack runBudget adds over the phases' own limits.
const runMargin = 10 * time.Second

// runBudget bounds the whole measurement (ts-041 Time limits): the beacon, the slowest target
// (targets run concurrently), the controls and, with routes, the slowest route, plus a margin.
// Every phase is bounded by its own limits; this is a backstop.
func runBudget(idle time.Duration, routes bool) time.Duration {
	b := beaconTimeout + targetTimeout(MethodProvisioningH1, idle) + controlsBudget + runMargin
	if routes {
		b += routeTimeout(idle)
	}
	return b
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

func (m *Measurer) measureTarget(ctx context.Context, logger model.Logger, zero time.Time, index int64,
	target Target, roots *x509.CertPool, tk *TestKeys) *TargetResult {
	address := ""
	res := &TargetResult{Name: target.Name, Hostname: target.Hostname, Address: &address, ClientHello: target.Hello.Name,
		Method: target.Method, T0: time.Since(zero).Seconds()}
	defer func() { res.T = time.Since(zero).Seconds() }()
	fail := func(op string, err error) *TargetResult {
		res.FailedOperation, res.Failure = op, measurexlite.NewFailure(err)
		return res
	}
	idle := idleDuration(m.config.IdleSeconds)
	ctx, cancel := context.WithTimeout(ctx, targetTimeout(target.Method, idle))
	defer cancel()
	// DNS, TCP, TLS and the upgrade or GET share the connect budget; only the provisioning flow
	// that follows uses the rest of ctx.
	connectCtx, cancelConnect := context.WithTimeout(ctx, connectBudget)
	defer cancelConnect()

	trace := measurexlite.NewTrace(index, zero, target.Name)
	ol := logx.NewOperationLogger(logger, "signal_client: %s %s", target.Name, target.Hostname)
	defer func() { ol.Stop(errorOrNil(res.Failure)) }()

	addrs, err := trace.NewStdlibResolver(logger).LookupHost(connectCtx, target.Hostname)
	tk.Queries = append(tk.Queries, trace.DNSLookupsFromRoundTrip()...)
	if err != nil {
		return fail(netxlite.ResolveOperation, err)
	}
	ip := firstIPv4(addrs)
	if ip == "" {
		return fail(netxlite.ResolveOperation, errNoAddress)
	}
	address = net.JoinHostPort(ip, "443")

	tr, err := dialTLS(connectCtx, trace, logger, zero, address, target.Hostname, target.Hello, roots, target.Name)
	tk.TCPConnect = append(tk.TCPConnect, trace.TCPConnects()...)
	defer func() { tk.NetworkEvents = append(tk.NetworkEvents, trace.NetworkEvents()...) }()
	if tr != nil && tr.handshake != nil {
		tk.TLSHandshakes = append(tk.TLSHandshakes, tr.handshake)
	}
	if err != nil {
		op := netxlite.TLSHandshakeOperation
		if tr == nil {
			op = netxlite.ConnectOperation
		}
		return fail(op, err)
	}
	res.ALPN, res.LeafKeyType = tr.alpn, tr.leafKey
	res.UserAgent = m.config.userAgent(target.Hello)
	drain := func() { tk.NetworkEvents = append(tk.NetworkEvents, trace.NetworkEvents()...) } // the buffer holds 64
	m.exerciseInto(ctx, connectCtx, tr, target, target.Hostname, target.Path, res, zero, drain)
	return res
}

// exerciseInto runs target's method on an established TLS connection and records the outcome
// in res, closing the connection when done. hostHeader and path are what the request names
// (they differ from the target's own through a front). With a nil checkpoint, provisioning
// methods stop after the upgrade (the static_address control); otherwise the provisioning flow
// runs, calling checkpoint after each step.
func (m *Measurer) exerciseInto(ctx, connectCtx context.Context, tr *tlsResult, target Target, hostHeader, path string,
	res *TargetResult, zero time.Time, checkpoint func()) {
	defer func() { _ = tr.conn.Close() }()
	// when ctx ends, close the conn: this unblocks any read or write, including HTTP/2 writes
	// waiting for flow control, which do not watch ctx themselves
	stopCloser := context.AfterFunc(ctx, func() { _ = tr.conn.Close() })
	defer stopCloser()
	fail := func(op string, err error) {
		res.FailedOperation, res.Failure = op, measurexlite.NewFailure(err)
	}
	ua := res.UserAgent
	switch target.Method {
	case MethodHTTPSGet:
		status, ts, sample, err := httpsGet(connectCtx, tr.conn, tr.alpn, hostHeader, path, ua)
		res.Status, res.SignalTimestamp, res.Response = int64(status), ts, sample
		if err != nil {
			fail(netxlite.HTTPRoundTripOperation, err)
		}
	case MethodEnclaveH1, MethodProvisioningH1, MethodProvisioningH2:
		var (
			up  *upgradeResult
			err error
		)
		if target.Method == MethodProvisioningH2 {
			up, err = websocketH2(ctx, connectCtx, tr.conn, hostHeader, path, ua)
		} else {
			up, err = websocketH1(connectCtx, tr.conn, hostHeader, path, ua)
		}
		if up != nil {
			res.Status, res.SignalTimestamp, res.WSExtensions = int64(up.status), up.signalTimestamp, up.extensions
			res.Response = up.response
		}
		if target.Method == MethodEnclaveH1 {
			// no credentials: 401 or 404 from the enclave front end is the expected answer
			if errors.Is(err, errUpgradeRefused) && (res.Status == 401 || res.Status == 404) {
				return
			}
			if err == nil { // accepted without credentials: unexpected, but Signal answered
				_ = up.closer.Close()
				return
			}
			fail("websocket_upgrade", err)
			return
		}
		if err != nil {
			fail("websocket_upgrade", err)
			return
		}
		defer up.closer.Close()
		if checkpoint == nil {
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

// connectBudget bounds DNS, TCP, TLS and the upgrade or GET of one target (a variable so tests
// can shorten it).
var connectBudget = 30 * time.Second

// maxIdleSeconds bounds IdleSeconds. Signal-Server closes provisioning sockets 90 s after they
// open (ProvisioningConnectListener's timeout); a longer idle would record Signal's own close as
// a failure. 60 s leaves room for the address, the keepalives and the ping.
const maxIdleSeconds = 60

// idleDuration converts IdleSeconds, clamped to [0, maxIdleSeconds].
func idleDuration(seconds int64) time.Duration {
	return time.Duration(min(max(seconds, 0), maxIdleSeconds)) * time.Second
}

// targetTimeout is the time budget for one target: the connect budget, plus, on provisioning
// sockets, every step's own timeout and the idle time, so that a slow but working server is never
// cut off by the overall budget before a step's own timeout would fire.
func targetTimeout(method Method, idle time.Duration) time.Duration {
	if method != MethodProvisioningH1 && method != MethodProvisioningH2 {
		return connectBudget
	}
	steps := addressTimeout + time.Duration(len(keepalivePaddings))*keepaliveTimeout + keepaliveTimeout // + ping
	return connectBudget + steps + idle
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
