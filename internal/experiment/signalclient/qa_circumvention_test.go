package signalclient

//
// QA scenarios for the circumvention routes, through the fronts of qa_servers_test.go.
//

import (
	"encoding/json"
	"net"
	"strings"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/ooni/netem"
)

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
	tk, _ := runQA(t, env, Config{})
	data, err := json.Marshal(tk)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"signal_circumvention":null`) {
		t.Errorf("signal_circumvention is not null: %s", data)
	}
}

// Outside the countries CircumventionEnabledIn names (here, the simulated probe has no country),
// no route is measured and the direct targets are unaffected.
func TestQACircumventionNotEnabledInTheProbesCountry(t *testing.T) {
	env := newQAEnv(t, nil)
	defer env.Close()
	tk, byName := runQA(t, env, Config{Circumvention: "all", CircumventionEnabledIn: "IR,RU", Controls: ControlsOff})
	if tk.Circumvention != nil {
		t.Errorf("routes measured outside IR and RU: %d", len(tk.Circumvention))
	}
	expectOK(t, byName, allTargets...)
}

func TestQACircumventionAllRoutes(t *testing.T) {
	env := newQAEnv(t, nil)
	defer env.Close()
	tk, byName := runQA(t, env, Config{Circumvention: "all", CircumventionCountry: "pk", Controls: ControlsOff})
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
	tk, byName := runQA(t, env, Config{Circumvention: "reflector_f,reflector_g", Controls: ControlsOff,
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
	env := newQAEnv(t, map[string]fault{qaFastlyFrontAddr: faultRefuseUpgrade})
	defer env.Close()
	tk, _ := runQA(t, env, Config{Circumvention: RouteReflectorF, Controls: ControlsOff})
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
	other := otherRoot(t)
	tk, _ := runQA(t, env, Config{Circumvention: RouteReflectorG, Controls: ControlsOff, SignalCA: other})
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
		tk, _ := runQA(t, env, Config{Circumvention: RouteAndroidG, CircumventionCountry: country, Controls: ControlsOff})
		r := byRoute(tk)[RouteAndroidG]
		if r.Front != want || r.Failure != nil {
			t.Errorf("%q: front %q failure %q", country, r.Front, deref(r.Failure))
		}
	}
}

// A service whose fronts were replaced separately resolves and dials its own front.
func TestQACircumventionPerServiceFront(t *testing.T) {
	env := newQAEnv(t, nil)
	defer env.Close()
	env.AddRecordToAllResolvers("front.storage.example", "", "10.10.34.36") // nothing there
	tk, _ := runQA(t, env, Config{Circumvention: RouteAndroidF, Controls: ControlsOff,
		Fronts: "android_f.storage=storage.signal.org.global.prod.fastly.net@front.storage.example"},
		func(m *Measurer) { m.limits.op = time.Second })
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
	tk, _ := runQA(t, env, Config{Circumvention: RouteReflectorF, Controls: ControlsOff,
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

// Each target of a route has its own operation limit, as a direct target does: a blackholed
// front costs each of a reflector route's tunnels one budget, and no more.
func TestQACircumventionConnectBudgetIsPerTarget(t *testing.T) {
	const op = 2 * time.Second
	env := newQAEnv(t, nil)
	defer env.Close()
	env.DPIEngine().AddRule(&netem.DPIDropTrafficForTLSSNI{Logger: log.Log, SNI: "pinterest.com"})
	tk, _ := runQA(t, env, Config{Circumvention: RouteReflectorF, Controls: ControlsOff,
		Fronts: "reflector_f=reflector-signal.global.ssl.fastly.net@pinterest.com"},
		func(m *Measurer) { m.limits.op = op })
	r := byRoute(tk)[RouteReflectorF]
	for _, inner := range r.Targets {
		took := inner.T - inner.T0
		if inner.FailedOperation != "tls_handshake" || deref(inner.Failure) != "generic_timeout_error" ||
			took < op.Seconds()-0.5 || took > op.Seconds()+1 {
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
		runQA(t, env, c.config)
		seen := env.uas.seen()
		env.Close()
		if hosts := seen[c.want]; !contains(hosts, "tunnel:reflector-signal.global.ssl.fastly.net") {
			t.Errorf("AppUserAgent=%v: tunnel upgrade not sent with %q (saw %v)", c.config.AppUserAgent, c.want, seen)
		}
	}
}
