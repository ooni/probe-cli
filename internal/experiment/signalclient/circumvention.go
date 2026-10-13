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

// routeInners returns what a route measures.
func routeInners(route string) []routeInner {
	chatLibsignal, cdsi := DefaultTargets[0], DefaultTargets[2]
	chatAndroid, storage := DefaultTargets[1], DefaultTargets[5]
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
// front, and measures the target through it.
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
	tag := "circumvention:" + route
	ol := logx.NewOperationLogger(logger, "signal_client: %s via %s", route, res.Front)
	defer func() { ol.Stop(errorOrNil(res.Failure)) }()
	routeFail := func(op string, failure *string) {
		if res.Failure == nil {
			res.Failure, res.FailedOperation = failure, op
		}
	}

	var frontRoots *x509.CertPool
	var err error
	if route == RouteReflectorG || route == RouteAndroidG {
		frontRoots, err = m.googleRoots()
	} else {
		frontRoots, err = m.mozillaRoots()
	}
	if err != nil {
		routeFail(netxlite.TLSHandshakeOperation, measurexlite.NewFailure(err))
		return res
	}

	// resolve returns the address of a front's first IPv4 address, with the system resolver
	resolve := func(front string, tags ...string) (string, error) {
		resolveCtx, cancel := context.WithTimeout(ctx, opTimeout)
		defer cancel()
		trace := measurexlite.NewTrace(index, zero, tags...)
		addrs, err := trace.NewStdlibResolver(logger).LookupHost(resolveCtx, front)
		tk.Queries = append(tk.Queries, trace.DNSLookupsFromRoundTrip()...)
		if err == nil && firstIPv4(addrs) == "" {
			err = errNoAddress
		}
		if err != nil {
			return "", err
		}
		return net.JoinHostPort(firstIPv4(addrs), "443"), nil
	}
	frontAddress, err := resolve(res.Front, tag)
	if err != nil {
		routeFail(netxlite.ResolveOperation, measurexlite.NewFailure(err))
		return res
	}
	res.FrontAddress = frontAddress

	for i, inner := range inners {
		in := &routeInnerRun{m: m, logger: logger, zero: zero, index: index + int64(i+1), route: route,
			tags: []string{tag, inner.target.Name}, frontAddress: res.FrontAddress, frontRoots: frontRoots,
			signalRoots: signalRoots, tk: tk}
		var r *TargetResult
		if isReflector {
			var status int64
			r, status = in.reflector(ctx, inner, res.Front, fc.hostHeader)
			if i == 0 { // the first tunnel's
				res.TunnelStatus = &status
			}
		} else {
			sfc := opts.fronts[route+"."+inner.service]
			front := res.Front
			if !contains(sfc.snis, front) { // a service whose fronts were replaced separately: its own front
				front = pickFront(route, opts.country, sfc.snis)
				address, err := resolve(front, in.tags...)
				if err != nil {
					r = &TargetResult{Name: inner.target.Name, Hostname: inner.target.Hostname, ClientHello: frontHello(route, inner.service, front).Name,
						Method: inner.target.Method, FailedOperation: netxlite.ResolveOperation,
						Failure: measurexlite.NewFailure(err), T0: time.Since(zero).Seconds(), T: time.Since(zero).Seconds()}
					res.Targets = append(res.Targets, r)
					continue
				}
				in.frontAddress = address
			}
			r = in.android(ctx, inner, front, sfc)
		}
		res.Targets = append(res.Targets, r)
		switch r.FailedOperation {
		case netxlite.ConnectOperation, tunnelUpgradeOperation:
			routeFail(r.FailedOperation, r.Failure)
		case netxlite.TLSHandshakeOperation:
			if !isReflector || in.frontFailed { // on reflectors, only the front's handshake
				routeFail(r.FailedOperation, r.Failure)
			}
		}
	}
	return res
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
	frontFailed  bool // the failure, if any, was the front's (not the inner target's)
}

// start creates the inner target's result and trace, and connects to the front: TCP and TLS
// within opTimeout. It returns a nil *tlsResult when that failed. Each later step (the tunnel's
// upgrade, the inner handshake, the upgrade or GET) has its own opTimeout.
func (in *routeInnerRun) start(ctx context.Context, target Target, hello Hello, front string) (
	res *TargetResult, trace *measurexlite.Trace, tr *tlsResult) {
	res = &TargetResult{Name: target.Name, Hostname: target.Hostname, ClientHello: hello.Name, Method: target.Method,
		T0: time.Since(in.zero).Seconds()}
	trace = measurexlite.NewTrace(in.index, in.zero, in.tags...)
	dialCtx, cancel := context.WithTimeout(ctx, opTimeout)
	defer cancel()
	tr, err := dialTLS(dialCtx, trace, in.logger, in.zero, in.frontAddress, front, hello, in.frontRoots, in.tags...)
	in.tk.TCPConnect = append(in.tk.TCPConnect, trace.TCPConnects()...)
	if tr != nil && tr.handshake != nil {
		in.tk.TLSHandshakes = append(in.tk.TLSHandshakes, tr.handshake)
	}
	if err != nil {
		in.frontFailed = true
		res.FailedOperation, res.Failure = netxlite.TLSHandshakeOperation, measurexlite.NewFailure(err)
		if tr == nil {
			res.FailedOperation = netxlite.ConnectOperation
		}
		return res, trace, nil
	}
	return res, trace, tr
}

// drainer returns a checkpoint that moves the trace's network events to the test keys.
func (in *routeInnerRun) drainer(trace *measurexlite.Trace) func() {
	return func() { in.tk.NetworkEvents = append(in.tk.NetworkEvents, trace.NetworkEvents()...) }
}

// reflector measures target through a reflector tunnel: the libsignal hello inside the tunnel,
// verified against Signal's roots, then the target's method. It also returns the tunnel's status.
func (in *routeInnerRun) reflector(ctx context.Context, inner routeInner, front, hostHeader string) (*TargetResult, int64) {
	target := inner.target
	res, trace, tr := in.start(ctx, target, HelloLibsignalReflector, front)
	defer func() { res.T = time.Since(in.zero).Seconds() }()
	defer in.drainer(trace)()
	res.ClientHello = target.Hello.Name
	res.UserAgent = in.m.config.userAgent(target.Hello)
	if tr == nil {
		return res, 0
	}
	stop := context.AfterFunc(ctx, func() { _ = tr.conn.Close() })
	defer stop()
	// libsignal sends its connection manager's User-Agent (the app's, plus " libsignal/<version>") on
	// the tunnel upgrade too (rust/net/infra/src/route/proxy.rs, ConnectionProxyConfig::Reflector)
	tunnelUA := in.m.config.userAgent(HelloLibsignalReflector)
	upgradeCtx, cancelUpgrade := context.WithTimeout(ctx, opTimeout)
	defer cancelUpgrade()
	up, err := tunnelUpgrade(upgradeCtx, tr.conn, hostHeader, target.Hostname, tunnelUA)
	var status int64
	if up != nil {
		status = int64(up.status)
	}
	if err != nil {
		_ = tr.conn.Close()
		in.frontFailed = true
		res.FailedOperation, res.Failure = tunnelUpgradeOperation, measurexlite.NewFailure(err)
		return res, status
	}
	tunnel := newTunnelConn(up, tr.conn)
	innerCtx, cancelInner := context.WithTimeout(ctx, opTimeout)
	defer cancelInner()
	itr, err := handshakeTLS(innerCtx, trace, in.zero, tunnel, "", target.Hostname, target.Hello, in.signalRoots, in.tags...)
	if itr != nil && itr.handshake != nil {
		in.tk.TLSHandshakes = append(in.tk.TLSHandshakes, itr.handshake)
	}
	if err != nil {
		res.FailedOperation, res.Failure = netxlite.TLSHandshakeOperation, measurexlite.NewFailure(err)
		return res, status
	}
	res.ALPN, res.LeafKeyType = itr.alpn, itr.leafKey
	exerciseCtx, cancelExercise := context.WithTimeout(ctx, opTimeout)
	defer cancelExercise()
	in.m.exerciseInto(ctx, exerciseCtx, itr, target, target.Hostname, target.Path, res, in.zero, in.drainer(trace))
	return res, status
}

// android measures target through an Android front: the front connection carries the request
// itself, with the service's Host header and path prefix.
func (in *routeInnerRun) android(ctx context.Context, inner routeInner, front string, fc frontConfig) *TargetResult {
	target := inner.target
	hello := frontHello(in.route, inner.service, front)
	res, trace, tr := in.start(ctx, target, hello, front)
	defer func() { res.T = time.Since(in.zero).Seconds() }()
	defer in.drainer(trace)()
	address := in.frontAddress
	res.Address = &address
	res.UserAgent = in.m.config.userAgent(hello)
	if tr == nil {
		return res
	}
	res.ALPN, res.LeafKeyType = tr.alpn, tr.leafKey
	exerciseCtx, cancelExercise := context.WithTimeout(ctx, opTimeout)
	defer cancelExercise()
	in.m.exerciseInto(ctx, exerciseCtx, tr, target, fc.hostHeader, fc.pathPrefix+inner.path, res, in.zero, in.drainer(trace))
	return res
}
