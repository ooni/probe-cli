package signalclient

import (
	"context"
	"crypto/x509"
	"errors"
	"net"
	"sync"
	"time"

	"github.com/ooni/probe-cli/v3/internal/logx"
	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/targetloading"
)

const (
	testName    = "signal_client" // provisional name, see doc.go
	testVersion = "0.2.0"
)

// Measurer performs the measurement.
type Measurer struct {
	config  Config // from the richer-input target, set by Run
	targets []Target

	limits limits

	// trust anchors (tests replace these)
	roots        func() ([]*x509.Certificate, error) // Signal's
	googleRoots  func() (*x509.CertPool, error)      // the Google fronts'
	mozillaRoots func() (*x509.CertPool, error)      // the Fastly fronts'
}

// NewExperimentMeasurer creates a new ExperimentMeasurer. Its configuration comes with each
// richer-input target (see richerinput.go).
func NewExperimentMeasurer() model.ExperimentMeasurer {
	return &Measurer{targets: DefaultTargets, limits: defaultLimits, roots: signalRoots,
		googleRoots: googleRoots, mozillaRoots: mozillaRoots}
}

// ExperimentName implements model.ExperimentMeasurer.
func (m *Measurer) ExperimentName() string { return testName }

// ExperimentVersion implements model.ExperimentMeasurer.
func (m *Measurer) ExperimentVersion() string { return testVersion }

var (
	// ErrInputRequired indicates that no richer-input target was provided.
	ErrInputRequired = targetloading.ErrInputRequired

	// ErrInvalidInputType indicates that the richer-input target has the wrong type.
	ErrInvalidInputType = targetloading.ErrInvalidInputType
)

// Run implements model.ExperimentMeasurer: it measures with the richer-input target's config.
func (m *Measurer) Run(ctx context.Context, args *model.ExperimentArgs) error {
	if args.Target == nil {
		return ErrInputRequired
	}
	target, ok := args.Target.(*RicherInputTarget)
	if !ok || target.Config == nil {
		return ErrInvalidInputType
	}
	configured := *m
	configured.config = *target.Config
	return configured.run(ctx, args)
}

func (m *Measurer) run(ctx context.Context, args *model.ExperimentArgs) error {
	logger := args.Session.Logger()
	zero := args.Measurement.MeasurementStartTimeSaved
	tk := &TestKeys{}
	args.Measurement.TestKeys = tk
	opts, err := m.parseOptions(args.Measurement.ProbeCC)
	if err != nil {
		return err
	}
	if m.config.Circumvention != "" && len(opts.routes) == 0 {
		logger.Infof("signal_client: not measuring the circumvention routes: the probe's country (%q) is not in CircumventionEnabledIn",
			args.Measurement.ProbeCC)
	}
	anchors, err := m.signalAnchors()
	if err != nil {
		return err
	}
	tk.SignalRoots = anchors.sha256
	roots := anchors.pool

	// One clock for the whole test (limits.test). Whatever it cuts short is recorded as
	// interrupted (markInterrupted), never as a network timeout.
	clockEnds := (time.Since(zero) + m.limits.test).Seconds()
	runCtx, stop := context.WithTimeoutCause(ctx, m.limits.test, errTestClock)
	defer stop()
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
			index := routeIndexBase + routeIndexStep*int64(i)
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
	clockRanOut := errors.Is(context.Cause(runCtx), errTestClock)
	stop()
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
	if clockRanOut {
		tk.markInterrupted(clockEnds)
	}
	return nil
}

// measureBeacon resolves Signal's outage beacon, uptime.signal.org, with the system resolver:
// 127.0.0.1 when the service is up, 127.0.0.2 during an outage (ts-041 Service status control).
func (m *Measurer) measureBeacon(ctx context.Context, logger model.Logger, zero time.Time, tk *TestKeys) {
	ctx, cancel := context.WithTimeout(ctx, m.limits.op)
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

// measureTarget measures target into res, recording its network events in tk. Each operation
// (DNS, TCP and TLS, the upgrade or GET, each provisioning step) has its limit; the test's clock
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

	resolveCtx, cancelResolve := context.WithTimeout(ctx, m.limits.op)
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

	dialCtx, cancelDial := context.WithTimeout(ctx, m.limits.op)
	tr, err := dialTLS(dialCtx, trace, logger, address, target.Hostname, target.Hello, roots)
	cancelDial()
	tk.recordTLS(trace, tr)
	defer func() { tk.NetworkEvents = append(tk.NetworkEvents, trace.NetworkEvents()...) }()
	if err != nil {
		fail(failedOperation(err), err)
		return
	}
	res.ALPN, res.LeafKeyType = tr.alpn, tr.leafKey
	res.UserAgent = m.config.userAgent(target.Client)
	// the first checkpoint follows the upgrade, before the provisioning flow: connected then
	drain := func() {
		connected()
		tk.NetworkEvents = append(tk.NetworkEvents, trace.NetworkEvents()...) // the buffer holds 64
	}
	opCtx, cancelOp := context.WithTimeout(ctx, m.limits.op)
	defer cancelOp()
	m.exercise(ctx, opCtx, tr, target.request(), zero, res, drain)
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
		a, err := httpsGet(opCtx, tr.conn, tr.alpn, req.hostHeader, req.path, ua)
		res.setAnswer(a)
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
			res.setAnswer(up.answer)
			res.WSExtensions = up.extensions
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
		res.Provisioning = runProvisioning(ctx, up.ws, zero, m.provisioningFlow(), checkpoint)
		if res.Provisioning.Failure != nil {
			res.Failure, res.FailedOperation = res.Provisioning.Failure, res.Provisioning.FailedStep
		}
	default:
		fail(netxlite.TopLevelOperation, errors.New("signal_client: unknown method"))
	}
}

// errTestClock is the cause of the test clock's end.
var errTestClock = errors.New("signal_client: the test's clock ran out")

// Trace indexes: the beacon is 0, the targets 1 to n, the controls follow; each route r takes
// routeIndexBase + r*routeIndexStep, and its inner targets the indexes after it.
const (
	routeIndexBase = 10000
	routeIndexStep = 10
)

// limits are the test's time limits (ts-041 Time limits) and how many controls run at once.
type limits struct {
	// test is the whole test's clock.
	test time.Duration

	// op bounds each network operation of a target or route: DNS, TCP and TLS, the upgrade or GET.
	op time.Duration

	// address and keepalive bound the provisioning steps: waiting for the server's address
	// (as ProvisioningSocket.kt does), and for each keepalive's response and the PONG.
	address, keepalive time.Duration

	// The controls run until the test's clock runs out; each has a short limit so that many fit
	// after the targets.
	doh, udp, controlHandshake time.Duration
	controlsParallelism        int
}

var defaultLimits = limits{
	test:                60 * time.Second,
	op:                  10 * time.Second,
	address:             10 * time.Second,
	keepalive:           10 * time.Second,
	doh:                 5 * time.Second,
	udp:                 5 * time.Second,
	controlHandshake:    5 * time.Second,
	controlsParallelism: 16,
}

// provisioningFlow is the provisioning flow this measurement runs.
func (m *Measurer) provisioningFlow() provisioningFlow {
	return provisioningFlow{paddings: keepalivePaddings, idle: idleDuration(m.config.IdleSeconds),
		addressTimeout: m.limits.address, keepaliveTimeout: m.limits.keepalive}
}

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
