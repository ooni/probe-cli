package signalclient

//
// Signal's circumvention routes (ts-041 Circumvention routes): domain fronts that the Android
// and iOS apps fall back to when the direct routes fail. Results go to signal_circumvention and
// never change what the targets say: a failing route says that Signal's circumvention would not
// help on that network, a different fact from Signal being blocked.
//
// Each front connection sends the hello the real client sends to that front: libsignal's
// reflector hello, or Signal-Android's hello with the front's ConnectionSpec (frontHello).
//

import (
	"context"
	"crypto/x509"
	"net"
	"time"

	"github.com/ooni/probe-cli/v3/internal/logx"
	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// Circumvention route names.
const (
	RouteReflectorF = "reflector_f"
	RouteReflectorG = "reflector_g"
	RouteAndroidG   = "android_g"
	RouteAndroidF   = "android_f"
)

// allRoutes is every route, in the order they run.
var allRoutes = []string{RouteReflectorF, RouteReflectorG, RouteAndroidG, RouteAndroidF}

// RouteResult is the outcome of one circumvention route.
type RouteResult struct {
	Route            string          `json:"route"`
	Front            string          `json:"front"`
	FrontAddress     string          `json:"front_address"`
	FrontClientHello string          `json:"front_client_hello"`
	HostHeader       string          `json:"host_header"`
	Country          string          `json:"country,omitempty"`       // android_g, android_f
	TunnelStatus     *int64          `json:"tunnel_status,omitempty"` // reflectors
	Targets          []*TargetResult `json:"targets"`
	Failure          *string         `json:"failure"`
	FailedOperation  string          `json:"failed_operation,omitempty"`
	T0               float64         `json:"t0"`
	T                float64         `json:"t"`
}

// The failed operations of a WebSocket upgrade (or extended CONNECT) and of a reflector tunnel's upgrade.
const (
	websocketUpgradeOperation = "websocket_upgrade"
	tunnelUpgradeOperation    = "tunnel_upgrade"
)

// routeInner is one target measured through a route.
type routeInner struct {
	target  Target // the target as measured directly (its hello is the inner one on reflectors)
	service string // Android routes: "chat" or "storage"
	path    string // Android routes: the path before the service prefix
}

// defaultTarget is the DefaultTargets entry with the given name.
func defaultTarget(name string) Target {
	for _, t := range DefaultTargets {
		if t.Name == name {
			return t
		}
	}
	panic("signal_client: no default target " + name)
}

// routeInners returns what a route measures.
func routeInners(route string) []routeInner {
	chatLibsignal, cdsi := defaultTarget("chat_libsignal"), defaultTarget("cdsi")
	chatAndroid, storage := defaultTarget("chat_android"), defaultTarget("storage")
	switch route {
	case RouteReflectorF, RouteReflectorG:
		return []routeInner{{target: chatLibsignal}, {target: cdsi}}
	default: // android_g, android_f
		return []routeInner{
			{target: chatAndroid, service: "chat", path: "/v1/websocket/provisioning/"},
			{target: storage, service: "storage", path: "/ping"},
		}
	}
}

// measureRoute measures one route: it resolves the front with the system resolver, then, for
// each inner target, connects to the front's first IPv4 address, handshakes with SNI set to the
// front, and measures the target through it. The route fails with the first failure that was
// the front's (TCP, TLS with the front, or a reflector's tunnel).
func (m *Measurer) measureRoute(ctx context.Context, logger model.Logger, zero time.Time, index int64, route string,
	opts *runOptions, signalRoots *x509.CertPool, tk *TestKeys) *RouteResult {
	isReflector := route == RouteReflectorF || route == RouteReflectorG
	inners := routeInners(route)
	frontKey := route
	if !isReflector {
		frontKey = route + "." + inners[0].service
	}
	fc := opts.fronts[frontKey]
	res := &RouteResult{Route: route, HostHeader: fc.hostHeader, Front: pickFront(route, opts.country, fc.snis),
		Targets: []*TargetResult{}, T0: time.Since(zero).Seconds()}
	defer func() { res.T = time.Since(zero).Seconds() }()
	res.FrontClientHello = frontHello(route, inners[0].service, res.Front).Name
	if isReflector {
		noTunnel := int64(0) // until a tunnel is attempted
		res.TunnelStatus = &noTunnel
	} else {
		res.Country = opts.country
	}
	ol := logx.NewOperationLogger(logger, "signal_client: %s via %s", route, res.Front)
	defer func() { ol.Stop(errorOrNil(res.Failure)) }()

	frontRoots, err := m.frontRoots(route)
	if err != nil {
		res.Failure, res.FailedOperation = measurexlite.NewFailure(err), netxlite.TLSHandshakeOperation
		return res
	}
	in := &routeInnerRun{m: m, logger: logger, zero: zero, route: route, frontRoots: frontRoots,
		signalRoots: signalRoots, tk: tk}
	tag := "circumvention:" + route
	if res.FrontAddress, err = in.resolveFront(ctx, index, res.Front, tag); err != nil {
		res.Failure, res.FailedOperation = measurexlite.NewFailure(err), netxlite.ResolveOperation
		return res
	}

	for i, inner := range inners {
		in.index, in.tags, in.frontAddress = index+int64(i+1), []string{tag, inner.target.Name}, res.FrontAddress
		var r *TargetResult
		var frontFailed bool
		if isReflector {
			var status int64
			r, status, frontFailed = in.reflector(ctx, inner, res.Front, fc.hostHeader)
			if i == 0 { // the first tunnel's
				res.TunnelStatus = &status
			}
		} else {
			r, frontFailed = in.android(ctx, inner, res.Front, opts)
		}
		res.Targets = append(res.Targets, r)
		if frontFailed && res.Failure == nil {
			res.Failure, res.FailedOperation = r.Failure, r.FailedOperation
		}
	}
	return res
}

// frontRoots are the roots a route's fronts are verified against: Google's or Mozilla's.
func (m *Measurer) frontRoots(route string) (*x509.CertPool, error) {
	if route == RouteReflectorG || route == RouteAndroidG {
		return m.googleRoots()
	}
	return m.mozillaRoots()
}

// routeInnerRun measures one inner target of a route.
type routeInnerRun struct {
	m            *Measurer
	logger       model.Logger
	zero         time.Time
	index        int64
	route        string
	tags         []string
	frontAddress string
	frontRoots   *x509.CertPool
	signalRoots  *x509.CertPool
	tk           *TestKeys
}

// resolveFront returns the address of a front's first IPv4 address, with the system resolver.
func (in *routeInnerRun) resolveFront(ctx context.Context, index int64, front string, tags ...string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, opTimeout)
	defer cancel()
	trace := measurexlite.NewTrace(index, in.zero, tags...)
	addrs, err := trace.NewStdlibResolver(in.logger).LookupHost(ctx, front)
	in.tk.Queries = append(in.tk.Queries, trace.DNSLookupsFromRoundTrip()...)
	if err == nil && firstIPv4(addrs) == "" {
		err = errNoAddress
	}
	if err != nil {
		return "", err
	}
	return net.JoinHostPort(firstIPv4(addrs), "443"), nil
}

// newResult is an inner target's result before it is measured.
func (in *routeInnerRun) newResult(target Target, hello Hello) *TargetResult {
	return &TargetResult{Name: target.Name, Hostname: target.Hostname, ClientHello: hello.Name, Method: target.Method,
		UserAgent: in.m.config.userAgent(target.Client), T0: time.Since(in.zero).Seconds()}
}

// connectFront records the inner target's evidence in a new trace and connects to the front: TCP
// and TLS within opTimeout. On failure, it records the failure in res and returns a nil
// *tlsResult. Each later step (the tunnel's upgrade, the inner handshake, the upgrade or GET) has
// its own opTimeout.
func (in *routeInnerRun) connectFront(ctx context.Context, res *TargetResult, hello Hello, front string) (
	*measurexlite.Trace, *tlsResult) {
	trace := measurexlite.NewTrace(in.index, in.zero, in.tags...)
	ctx, cancel := context.WithTimeout(ctx, opTimeout)
	defer cancel()
	tr, err := dialTLS(ctx, trace, in.logger, in.frontAddress, front, hello, in.frontRoots)
	in.tk.recordTLS(trace, tr)
	if err != nil {
		res.FailedOperation, res.Failure = failedOperation(err), measurexlite.NewFailure(err)
		return trace, nil
	}
	return trace, tr
}

// drainer returns a checkpoint that moves the trace's network events to the test keys.
func (in *routeInnerRun) drainer(trace *measurexlite.Trace) func() {
	return func() { in.tk.NetworkEvents = append(in.tk.NetworkEvents, trace.NetworkEvents()...) }
}

// reflector measures target through a reflector tunnel: the libsignal hello inside the tunnel,
// verified against Signal's roots, then the target's method. It also returns the tunnel's
// status, and whether a failure was the front's (TCP, TLS or the tunnel's upgrade).
func (in *routeInnerRun) reflector(ctx context.Context, inner routeInner, front, hostHeader string) (
	res *TargetResult, tunnelStatus int64, frontFailed bool) {
	target := inner.target
	res = in.newResult(target, target.Hello)
	defer func() { res.T = time.Since(in.zero).Seconds() }()
	trace, tr := in.connectFront(ctx, res, HelloLibsignalReflector, front)
	defer in.drainer(trace)()
	if tr == nil {
		return res, 0, true
	}
	stop := context.AfterFunc(ctx, func() { _ = tr.conn.Close() })
	defer stop()
	// libsignal sends its connection manager's User-Agent (the app's, plus " libsignal/<version>") on
	// the tunnel upgrade too (rust/net/infra/src/route/proxy.rs, ConnectionProxyConfig::Reflector)
	upgradeCtx, cancelUpgrade := context.WithTimeout(ctx, opTimeout)
	defer cancelUpgrade()
	up, err := tunnelUpgrade(upgradeCtx, tr.conn, hostHeader, target.Hostname, in.m.config.userAgent(ClientLibsignal))
	if up != nil {
		tunnelStatus = int64(up.status)
	}
	if err != nil {
		_ = tr.conn.Close()
		res.FailedOperation, res.Failure = tunnelUpgradeOperation, measurexlite.NewFailure(err)
		return res, tunnelStatus, true
	}
	innerCtx, cancelInner := context.WithTimeout(ctx, opTimeout)
	defer cancelInner()
	itr, err := handshakeTLS(innerCtx, trace, newTunnelConn(up, tr.conn), "", target.Hostname, target.Hello, in.signalRoots)
	in.tk.recordTLS(trace, itr)
	if err != nil {
		res.FailedOperation, res.Failure = failedOperation(err), measurexlite.NewFailure(err)
		return res, tunnelStatus, false
	}
	res.ALPN, res.LeafKeyType = itr.alpn, itr.leafKey
	opCtx, cancelOp := context.WithTimeout(ctx, opTimeout)
	defer cancelOp()
	in.m.exercise(ctx, opCtx, itr, target.request(), in.zero, res, in.drainer(trace))
	return res, tunnelStatus, false
}

// android measures target through an Android front: the front connection carries the request
// itself, with the service's Host header and path prefix. A service whose fronts the Fronts
// option replaced separately is reached through its own front. It also returns whether a failure
// was the front's (TCP or TLS).
func (in *routeInnerRun) android(ctx context.Context, inner routeInner, front string, opts *runOptions) (
	res *TargetResult, frontFailed bool) {
	target := inner.target
	fc := opts.fronts[in.route+"."+inner.service]
	routeFront := front
	if !contains(fc.snis, front) {
		front = pickFront(in.route, opts.country, fc.snis)
	}
	hello := frontHello(in.route, inner.service, front)
	res = in.newResult(target, hello)
	defer func() { res.T = time.Since(in.zero).Seconds() }()
	if front != routeFront {
		address, err := in.resolveFront(ctx, in.index, front, in.tags...)
		if err != nil {
			res.FailedOperation, res.Failure = netxlite.ResolveOperation, measurexlite.NewFailure(err)
			return res, false
		}
		in.frontAddress = address
	}
	address := in.frontAddress
	res.Address = &address
	trace, tr := in.connectFront(ctx, res, hello, front)
	defer in.drainer(trace)()
	if tr == nil {
		return res, true
	}
	res.ALPN, res.LeafKeyType = tr.alpn, tr.leafKey
	opCtx, cancelOp := context.WithTimeout(ctx, opTimeout)
	defer cancelOp()
	req := request{method: target.Method, hostHeader: fc.hostHeader, path: fc.pathPrefix + inner.path, provision: true}
	in.m.exercise(ctx, opCtx, tr, req, in.zero, res, in.drainer(trace))
	return res, false
}
