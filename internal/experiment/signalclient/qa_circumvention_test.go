package signalclient

//
// Simulated domain fronts and QA scenarios for the circumvention routes.
//

import (
	"bufio"
	"context"
	"crypto/tls"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/netemx"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// qaFrontFactory is a CDN front: libsignal's reflector tunnel (/tls-tunnel with X-Signal-Host)
// and Signal-Android's fronted requests (Host header and path prefix), relayed to the fake
// Signal servers.
type qaFrontFactory struct {
	t        *testing.T
	names    []string
	behavior qaBehavior
}

var _ netemx.NetStackServerFactory = &qaFrontFactory{}

func (f *qaFrontFactory) MustNewServer(_ netemx.NetStackServerFactoryEnv, stack *netem.UNetStack) netemx.NetStackServer {
	return &qaFront{factory: f, stack: stack}
}

type qaFront struct {
	factory  *qaFrontFactory
	stack    *netem.UNetStack
	mu       sync.Mutex
	listener net.Listener
}

func (s *qaFront) MustStart() {
	s.mu.Lock()
	defer s.mu.Unlock()
	ln := runtimex.Try1(s.stack.ListenTCP("tcp", &net.TCPAddr{IP: net.ParseIP(s.stack.IPAddress()), Port: 443}))
	config := s.stack.MustNewServerTLSConfig(s.factory.names[0], s.factory.names[1:]...)
	config.NextProtos = []string{"http/1.1"}
	s.listener = tls.NewListener(ln, config)
	go func() {
		for {
			conn, err := s.listener.Accept()
			if err != nil {
				return
			}
			go s.serve(conn)
		}
	}()
}

func (s *qaFront) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listener != nil {
		s.listener.Close()
	}
	return nil
}

// frontedServices maps a front's Host header (and path prefix) to the Signal host it reaches.
var frontedServices = map[string]string{
	"chat-signal.global.ssl.fastly.net":         "chat.signal.org",
	"storage.signal.org.global.prod.fastly.net": "storage.signal.org",
	googleReflectorHost + " /service":           "chat.signal.org",
	googleReflectorHost + " /storage":           "storage.signal.org",
}

var reflectorHosts = map[string]bool{"reflector-signal.global.ssl.fastly.net": true, googleReflectorHost: true}

func (s *qaFront) serve(conn net.Conn) {
	defer conn.Close()
	br := bufio.NewReader(conn)
	req, err := http.ReadRequest(br)
	if err != nil {
		return
	}
	b := s.factory.behavior
	if strings.EqualFold(req.Header.Get("Upgrade"), "websocket") && req.URL.Path == reflectorPath && reflectorHosts[req.Host] {
		qaSeenUserAgents.add(req.UserAgent(), "tunnel:"+req.Host)
		upstream, ok := qaUpstreams[req.Header.Get("X-Signal-Host")]
		if b.refuseUpgrade || !ok {
			fmt.Fprintf(conn, "HTTP/1.1 403 Forbidden\r\nContent-Length: 0\r\nConnection: close\r\n\r\n")
			return
		}
		up, err := s.stack.DialContext(context.Background(), "tcp", net.JoinHostPort(upstream, "443"))
		if err != nil {
			fmt.Fprintf(conn, "HTTP/1.1 502 Bad Gateway\r\nContent-Length: 0\r\nConnection: close\r\n\r\n")
			return
		}
		defer up.Close()
		fmt.Fprintf(conn, "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n"+
			"Sec-WebSocket-Accept: %s\r\n\r\n", websocketAccept(req.Header.Get("Sec-WebSocket-Key")))
		qaRelay(br, conn, up)
		return
	}
	// Signal-Android's fronted requests
	host := frontedServices[req.Host]
	if host == "" {
		for _, prefix := range []string{"/service", "/storage"} {
			if strings.HasPrefix(req.URL.Path, prefix+"/") {
				host = frontedServices[req.Host+" "+prefix]
				req.URL.Path = strings.TrimPrefix(req.URL.Path, prefix)
			}
		}
	}
	if host == "" {
		fmt.Fprintf(conn, "HTTP/1.1 404 Not Found\r\nContent-Length: 0\r\nConnection: close\r\n\r\n")
		return
	}
	req.Host = host
	signal := &qaSignalServer{factory: &qaSignalServerFactory{t: s.factory.t, names: []string{host}, behavior: b,
		uas: qaSeenUserAgents}}
	signal.serveH1Request(conn, br, req)
}

// qaRelay carries a reflector tunnel: the client's (masked) binary frames to up as bytes, and
// up's bytes back as unmasked binary frames.
func qaRelay(br *bufio.Reader, conn net.Conn, up net.Conn) {
	done := make(chan struct{}, 2)
	go func() {
		defer func() { done <- struct{}{} }()
		ws := &wsConn{r: br}
		for {
			op, payload, err := ws.readFrame()
			if err != nil || op == opClose {
				up.Close()
				return
			}
			if op == opBinary {
				if _, err := up.Write(payload); err != nil {
					return
				}
			}
		}
	}()
	go func() {
		defer func() { done <- struct{}{} }()
		buf := make([]byte, 70000) // more than a frame, to exercise the 64 KiB split on the way back
		for {
			n, err := up.Read(buf)
			if n > 0 {
				if err := qaWriteServerFrame(conn, opBinary, buf[:n]); err != nil {
					return
				}
			}
			if err != nil {
				conn.Close()
				return
			}
		}
	}()
	<-done
}

// qaWriteServerFrame writes an unmasked frame, as servers do.
func qaWriteServerFrame(w io.Writer, op byte, p []byte) error {
	hdr := []byte{0x80 | op}
	switch n := len(p); {
	case n < 126:
		hdr = append(hdr, byte(n))
	case n < 65536:
		hdr = binary.BigEndian.AppendUint16(append(hdr, 126), uint16(n))
	default:
		hdr = binary.BigEndian.AppendUint64(append(hdr, 127), uint64(n))
	}
	_, err := w.Write(append(hdr, p...))
	return err
}

// byRoute indexes the circumvention results.
func byRoute(tk *TestKeys) map[string]*RouteResult {
	out := map[string]*RouteResult{}
	for _, r := range tk.Circumvention {
		out[r.Route] = r
	}
	return out
}

func innerByName(r *RouteResult) map[string]*TargetResult {
	out := map[string]*TargetResult{}
	for _, t := range r.Targets {
		out[t.Name] = t
	}
	return out
}

func TestQACircumventionOffByDefault(t *testing.T) {
	env := newQAEnv(t, nil)
	defer env.Close()
	tk, _ := runQA(t, env)
	data, err := json.Marshal(tk)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"signal_circumvention":null`) {
		t.Errorf("signal_circumvention is not null: %s", data)
	}
}

func TestQACircumventionAllRoutes(t *testing.T) {
	env := newQAEnv(t, nil)
	defer env.Close()
	tk, byName := runQAWith(t, env, Config{Circumvention: "all", CircumventionCountry: "pk", Controls: ControlsOff})
	expectOK(t, byName, allTargets...)
	routes := byRoute(tk)
	if len(tk.Circumvention) != 4 {
		t.Fatalf("expected 4 routes, got %d", len(tk.Circumvention))
	}
	for _, name := range []string{RouteReflectorF, RouteReflectorG} {
		r := routes[name]
		if r.Failure != nil || r.TunnelStatus == nil || *r.TunnelStatus != 101 || r.FrontClientHello != HelloLibsignalReflector.Name {
			t.Errorf("%s: %+v failure=%q", name, r, deref(r.Failure))
			continue
		}
		inner := innerByName(r)
		chat, cdsi := inner["chat_libsignal"], inner["cdsi"]
		if chat == nil || chat.Failure != nil || chat.Address != nil || chat.ClientHello != HelloLibsignal.Name ||
			chat.LeafKeyType == "" || chat.Provisioning == nil || chat.Provisioning.PingEchoed == nil || !*chat.Provisioning.PingEchoed {
			t.Errorf("%s: chat_libsignal through the tunnel: %+v", name, chat)
		}
		if cdsi == nil || cdsi.Failure != nil || cdsi.Status != 404 {
			t.Errorf("%s: cdsi through the tunnel: %+v", name, cdsi)
		}
	}
	if f := routes[RouteReflectorF].Front; !contains(fastlyFronts, f) {
		t.Errorf("reflector_f front %q", f)
	}
	g := routes[RouteAndroidG]
	if g.Front != "www.google.com.pk" || g.Country != "PK" || g.HostHeader != googleReflectorHost ||
		g.FrontClientHello != HelloAndroidGmailWebSocket.Name {
		t.Errorf("android_g: %+v", g)
	}
	if f := routes[RouteAndroidF]; f.FrontClientHello != HelloAndroidModernWebSocket.Name {
		t.Errorf("android_f: %+v", f)
	}
	for _, name := range []string{RouteAndroidG, RouteAndroidF} {
		inner := innerByName(routes[name])
		chat, storage := inner["chat_android"], inner["storage"]
		if chat == nil || chat.Failure != nil || !chat.SignalTimestamp || chat.Status != 101 || chat.Provisioning == nil ||
			len(chat.Provisioning.Keepalives) != 3 || chat.Address == nil || *chat.Address != net.JoinHostPort(addrOf(name), "443") {
			t.Errorf("%s: chat_android: %+v", name, chat)
		}
		if storage == nil || storage.Failure != nil || storage.Status != 200 || !storage.SignalTimestamp {
			t.Errorf("%s: storage: %+v", name, storage)
		}
		if chat != nil && storage != nil && (chat.ClientHello != frontHello(name, "chat", routes[name].Front).Name ||
			storage.ClientHello != frontHello(name, "storage", routes[name].Front).Name) {
			t.Errorf("%s: hellos %s, %s", name, chat.ClientHello, storage.ClientHello)
		}
	}
	if f := routes[RouteAndroidF]; f.Front != fastlyFronts[0] || f.HostHeader != "chat-signal.global.ssl.fastly.net" {
		t.Errorf("android_f outside IR and CU should measure its first front: %+v", f)
	}
	// android_g's front handshakes are TLS 1.2; every route's records are tagged
	for _, hs := range tk.TLSHandshakes {
		if len(hs.Tags) > 0 && hs.Tags[0] == "circumvention:"+RouteAndroidG && hs.TLSVersion != "TLSv1.2" {
			t.Errorf("android_g handshake with %s: %+v", hs.TLSVersion, hs)
		}
	}
	tagged := map[string]bool{}
	for _, hs := range tk.TLSHandshakes {
		tagged[strings.Join(hs.Tags, ",")] = true
	}
	for _, want := range []string{"circumvention:reflector_f,chat_libsignal", "circumvention:reflector_g,cdsi",
		"circumvention:android_f,storage"} {
		if !tagged[want] {
			t.Errorf("no TLS record tagged %q: %v", want, tagged)
		}
	}
}

func addrOf(route string) string {
	if route == RouteAndroidG || route == RouteReflectorG {
		return qaGoogleFrontAddr
	}
	return qaFastlyFrontAddr
}

// A blocked front fails its route at the front's handshake; the direct targets are unaffected.
func TestQACircumventionFrontSNIBlocked(t *testing.T) {
	env := newQAEnv(t, nil)
	defer env.Close()
	env.DPIEngine().AddRule(&netem.DPIResetTrafficForTLSSNI{Logger: log.Log, SNI: "pinterest.com"})
	tk, byName := runQAWith(t, env, Config{Circumvention: "reflector_f,reflector_g", Controls: ControlsOff,
		Fronts: "reflector_f=reflector-signal.global.ssl.fastly.net@pinterest.com"})
	expectOK(t, byName, allTargets...)
	routes := byRoute(tk)
	f := routes[RouteReflectorF]
	if f.Front != "pinterest.com" || f.FailedOperation != "tls_handshake" || deref(f.Failure) != "connection_reset" ||
		f.TunnelStatus == nil || *f.TunnelStatus != 0 {
		t.Errorf("reflector_f: %+v failure=%q", f, deref(f.Failure))
	}
	if g := routes[RouteReflectorG]; g.Failure != nil {
		t.Errorf("reflector_g failed: %q", deref(g.Failure))
	}
}

func TestQACircumventionTunnelRefused(t *testing.T) {
	env := newQAEnv(t, map[string]qaBehavior{qaFastlyFrontAddr: {refuseUpgrade: true}})
	defer env.Close()
	tk, _ := runQAWith(t, env, Config{Circumvention: RouteReflectorF, Controls: ControlsOff})
	r := byRoute(tk)[RouteReflectorF]
	if r.FailedOperation != tunnelUpgradeOperation || deref(r.Failure) != FailureWebSocketUpgradeRefused ||
		r.TunnelStatus == nil || *r.TunnelStatus != 403 {
		t.Fatalf("reflector_f: %+v failure=%q", r, deref(r.Failure))
	}
	for _, inner := range r.Targets {
		if inner.FailedOperation != tunnelUpgradeOperation {
			t.Errorf("%s: %q", inner.Name, inner.FailedOperation)
		}
	}
}

// Through the tunnel, the inner handshake is still verified against Signal's roots: a relay that
// terminates TLS itself is caught.
func TestQACircumventionInnerTLSIsVerified(t *testing.T) {
	env := newQAEnv(t, nil)
	defer env.Close()
	other := mustSelfSignedPEM(t)
	tk, _ := runQAWith(t, env, Config{Circumvention: RouteReflectorG, Controls: ControlsOff, SignalCA: other})
	r := byRoute(tk)[RouteReflectorG]
	if r.Failure != nil || r.TunnelStatus == nil || *r.TunnelStatus != 101 {
		t.Fatalf("the front itself is verified with Google's roots, not SignalCA: %+v failure=%q", r, deref(r.Failure))
	}
	for _, inner := range r.Targets {
		if inner.FailedOperation != "tls_handshake" || deref(inner.Failure) != "ssl_unknown_authority" {
			t.Errorf("%s: %q %q", inner.Name, inner.FailedOperation, deref(inner.Failure))
		}
	}
}

func TestQACircumventionAndroidFrontsByCountry(t *testing.T) {
	env := newQAEnv(t, nil)
	defer env.Close()
	for country, want := range map[string]string{"EG": "www.google.com.eg", "BR": "www.google.com", "": "www.google.com"} {
		tk, _ := runQAWith(t, env, Config{Circumvention: RouteAndroidG, CircumventionCountry: country, Controls: ControlsOff})
		r := byRoute(tk)[RouteAndroidG]
		if r.Front != want || r.Failure != nil {
			t.Errorf("%q: front %q failure %q", country, r.Front, deref(r.Failure))
		}
	}
}

// A service whose fronts were replaced separately resolves and dials its own front.
func TestQACircumventionPerServiceFront(t *testing.T) {
	defer func(d time.Duration) { connectBudget = d }(connectBudget)
	connectBudget = time.Second
	env := newQAEnv(t, nil)
	defer env.Close()
	env.AddRecordToAllResolvers("front.storage.example", "", "10.10.34.36") // nothing there
	tk, _ := runQAWith(t, env, Config{Circumvention: RouteAndroidF, Controls: ControlsOff,
		Fronts: "android_f.storage=storage.signal.org.global.prod.fastly.net@front.storage.example"})
	r := byRoute(tk)[RouteAndroidF]
	inner := innerByName(r)
	if chat := inner["chat_android"]; chat.Failure != nil || *chat.Address != net.JoinHostPort(qaFastlyFrontAddr, "443") {
		t.Errorf("chat_android: %+v failure=%q", chat, deref(chat.Failure))
	}
	if st := inner["storage"]; st.FailedOperation != "connect" || st.Address == nil || *st.Address != "10.10.34.36:443" {
		t.Errorf("storage did not dial its own front: %+v", st)
	}
}

// A front that does not resolve fails the route at resolve; a reflector's tunnel_status is 0.
func TestQACircumventionFrontDoesNotResolve(t *testing.T) {
	env := newQAEnv(t, nil)
	defer env.Close()
	tk, _ := runQAWith(t, env, Config{Circumvention: RouteReflectorF, Controls: ControlsOff,
		Fronts: "reflector_f=reflector-signal.global.ssl.fastly.net@nx.front.example"})
	r := byRoute(tk)[RouteReflectorF]
	if r.FailedOperation != "resolve" || r.Failure == nil || r.TunnelStatus == nil || *r.TunnelStatus != 0 || len(r.Targets) != 0 {
		t.Fatalf("%+v failure=%q", r, deref(r.Failure))
	}
	data, _ := json.Marshal(r)
	if !strings.Contains(string(data), `"tunnel_status":0`) {
		t.Errorf("tunnel_status missing: %s", data)
	}
}

// Each target of a route has its own connect budget, as a direct target does: a blackholed
// front costs each of a reflector route's tunnels one budget, and no more.
func TestQACircumventionConnectBudgetIsPerTarget(t *testing.T) {
	defer func(d time.Duration) { connectBudget = d }(connectBudget)
	connectBudget = 2 * time.Second
	env := newQAEnv(t, nil)
	defer env.Close()
	env.DPIEngine().AddRule(&netem.DPIDropTrafficForTLSSNI{Logger: log.Log, SNI: "pinterest.com"})
	tk, _ := runQAWith(t, env, Config{Circumvention: RouteReflectorF, Controls: ControlsOff,
		Fronts: "reflector_f=reflector-signal.global.ssl.fastly.net@pinterest.com"})
	r := byRoute(tk)[RouteReflectorF]
	for _, inner := range r.Targets {
		took := inner.T - inner.T0
		if inner.FailedOperation != "tls_handshake" || deref(inner.Failure) != "generic_timeout_error" ||
			took < connectBudget.Seconds()-0.5 || took > connectBudget.Seconds()+1 {
			t.Errorf("%s: %q %q after %.2f s", inner.Name, inner.FailedOperation, deref(inner.Failure), took)
		}
	}
	if len(r.Targets) != 2 || r.FailedOperation != "tls_handshake" {
		t.Errorf("route: %d targets, %q %q", len(r.Targets), r.FailedOperation, deref(r.Failure))
	}
}

// The reflector tunnel's upgrade carries libsignal's User-Agent (the app's, plus libsignal's
// token): the OONI one by default, the app's with AppUserAgent.
func TestQACircumventionTunnelUserAgent(t *testing.T) {
	for _, c := range []struct {
		config Config
		want   string
	}{
		{Config{Circumvention: RouteReflectorF, Controls: ControlsOff}, ooniUserAgent + " libsignal/0.105.0"},
		{Config{Circumvention: RouteReflectorF, Controls: ControlsOff, AppUserAgent: true},
			"Signal-Android/8.31.1 Android/34 libsignal/0.105.0"},
	} {
		env := newQAEnv(t, nil)
		runQAWith(t, env, c.config)
		seen := qaSeenUserAgents.seen()
		env.Close()
		if hosts := seen[c.want]; !contains(hosts, "tunnel:reflector-signal.global.ssl.fastly.net") {
			t.Errorf("AppUserAgent=%v: tunnel upgrade not sent with %q (saw %v)", c.config.AppUserAgent, c.want, seen)
		}
	}
}
