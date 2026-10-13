package signalclient

//
// QA scenarios for the address controls, one per blocking method of the spec's conclusions
// table, and for SignalCA.
//

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"strings"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/google/gopacket/layers"
	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/netemx"
)

// controlsOf indexes a target's controls by kind (several static_address controls are listed
// under "static_address" in order).
func controlsOf(r *TargetResult) map[string][]*ControlResult {
	out := map[string][]*ControlResult{}
	for _, c := range r.Controls {
		out[c.Kind] = append(out[c.Kind], c)
	}
	return out
}

func kindsOf(r *TargetResult) []string {
	var out []string
	for _, c := range r.Controls {
		out = append(out, c.Kind)
	}
	return out
}

// expectNoControls fails if any named target has controls.
func expectNoControls(t *testing.T, byName map[string]*TargetResult, names ...string) {
	t.Helper()
	for _, n := range names {
		if r := byName[n]; r.Controls != nil {
			t.Errorf("%s: unexpected controls %v", n, kindsOf(r))
		}
	}
}

// DNS blocking: NXDOMAIN from the system resolver (and from 1.1.1.1:53, which the DPI also sees);
// DoH answers, and the static addresses reach Signal.
func TestQAControlsDNSBlocking(t *testing.T) {
	env := newQAEnv(t, nil)
	defer env.Close()
	env.DPIEngine().AddRule(&netem.DPISpoofDNSResponse{Logger: log.Log, Domain: "chat.signal.org"})
	tk, byName := runQA(t, env, Config{})
	r := byName["chat_android"]
	expectFailure(t, r, "resolve", "dns_nxdomain_error")
	if got := strings.Join(kindsOf(r), ","); got != "doh,udp_1111,static_address,static_address,static_address" {
		t.Fatalf("controls %s", got)
	}
	c := controlsOf(r)
	if doh := c[ControlDoH][0]; doh.Failure != nil || doh.Resolver != dohResolvers[0] || !contains(*doh.Addresses, qaChatAddr) {
		t.Errorf("doh: %+v failure=%q", doh, deref(doh.Failure))
	}
	if udp := c[ControlUDP][0]; deref(udp.Failure) != "dns_nxdomain_error" || udp.Resolver != udpResolver {
		t.Errorf("udp_1111: %+v failure=%q", udp, deref(udp.Failure))
	}
	static := c[ControlStatic]
	for i, want := range []string{"13.248.212.111:443", "76.223.92.165:443", "[2600:9000:a507:ab6d:4ce3:2f58:25d7:9cbf]:443"} {
		if static[i].Address != want {
			t.Errorf("static %d: %s, want %s", i, static[i].Address, want)
		}
	}
	for _, s := range static[:2] {
		if s.Failure != nil || s.AddressSource != "android" || *s.Status != 101 || !*s.SignalTimestamp || !*s.Verified {
			t.Errorf("static %s: %+v failure=%q", s.Address, s, deref(s.Failure))
		}
	}
	if static[2].Failure == nil || static[2].AddressSource != "desktop" { // netem has no IPv6
		t.Errorf("IPv6 static: %+v", static[2])
	}
	expectNoControls(t, byName, "chat_libsignal", "cdsi", "storage")
	if tk.ControlsTruncated {
		t.Error("truncated")
	}
	// the controls' evidence is tagged
	found := false
	for _, q := range tk.Queries {
		if strings.Join(q.Tags, ",") == "chat_android,control:udp_1111" && q.QueryType == "A" && q.ResolverAddress == udpResolver {
			found = true
		}
	}
	if !found {
		t.Error("no tagged udp_1111 query")
	}
}

// DNS manipulation: the system resolver forges an address; DoH gives the real one, which is in
// no static list, so it gets its own static_address control.
func TestQAControlsForgedAnswer(t *testing.T) {
	env := newQAEnv(t, nil)
	defer env.Close()
	env.ISPResolverConfig().AddRecord("storage.signal.org", "", "10.10.34.35") // nothing there
	_, byName := runQA(t, env, Config{}, func(m *Measurer) { m.limits.op = time.Second })
	r := byName["storage"]
	if r.FailedOperation != "connect" {
		t.Fatalf("storage: %q %q", r.FailedOperation, deref(r.Failure))
	}
	var fromDoH *ControlResult
	for _, s := range controlsOf(r)[ControlStatic] {
		if s.AddressSource == "doh" {
			fromDoH = s
		}
	}
	if fromDoH == nil || fromDoH.Address != qaCDNAddr+":443" || fromDoH.Failure != nil || *fromDoH.Status == 0 {
		t.Fatalf("no successful static_address from the DoH answer: %v", kindsOf(r))
	}
}

// IP blocking of one address: other static addresses reach Signal (libsignal would connect).
func TestQAControlsIPBlocking(t *testing.T) {
	env := newQAEnv(t, nil)
	defer env.Close()
	env.DPIEngine().AddRule(&netem.DPICloseConnectionForServerEndpoint{Logger: log.Log, ServerIPAddress: qaGRPCChatAddr, ServerPort: 443})
	_, byName := runQA(t, env, Config{})
	r := byName["chat_libsignal"]
	expectFailure(t, r, "connect", "connection_refused")
	c := controlsOf(r)
	if len(c[ControlDecoySNI]) != 0 || len(c[ControlAlternate]) != 0 || len(c[ControlBrowser]) != 0 || len(c[ControlTLS12]) != 0 {
		t.Errorf("TLS controls after a connect failure: %v", kindsOf(r))
	}
	s := c[ControlStatic][0]
	if s.Address != "76.223.66.180:443" || s.AddressSource != "libsignal" || s.Failure != nil || *s.Status != 200 ||
		!*s.SignalTimestamp || s.ClientHello != HelloLibsignal.Name {
		t.Errorf("static: %+v failure=%q", s, deref(s.Failure))
	}
}

// SNI blocking: the decoy completes on the same address, every real-SNI handshake fails.
func TestQAControlsSNIBlocking(t *testing.T) {
	env := newQAEnv(t, nil)
	defer env.Close()
	env.DPIEngine().AddRule(&netem.DPIResetTrafficForTLSSNI{Logger: log.Log, SNI: "grpc.chat.signal.org"})
	_, byName := runQA(t, env, Config{})
	r := byName["chat_libsignal"]
	expectFailure(t, r, "tls_handshake", "connection_reset")
	if got := strings.Join(kindsOf(r), ","); got != "doh,udp_1111,static_address,static_address,decoy_sni,alternate_hello,browser_hello,tls12" {
		t.Fatalf("controls %s", got)
	}
	c := controlsOf(r)
	decoy := c[ControlDecoySNI][0]
	if decoy.Failure != nil || *decoy.Verified || decoy.AddressSource != "target" || decoy.Address != qaGRPCChatAddr+":443" ||
		len(decoy.SNI) != len("grpc.chat.signal.org") || !strings.HasSuffix(decoy.SNI, ".example.com") || decoy.LeafKeyType == "" {
		t.Errorf("decoy: %+v failure=%q", decoy, deref(decoy.Failure))
	}
	for _, kind := range []string{ControlAlternate, ControlBrowser, ControlTLS12} {
		x := c[kind][0]
		if deref(x.Failure) != "connection_reset" || x.FailedOperation != "tls_handshake" || x.SNI != "grpc.chat.signal.org" {
			t.Errorf("%s: %+v failure=%q", kind, x, deref(x.Failure))
		}
	}
	if c[ControlAlternate][0].ClientHello != HelloAndroidREST.Name || c[ControlBrowser][0].ClientHello != HelloChrome.Name ||
		*c[ControlBrowser][0].Verified || c[ControlTLS12][0].ClientHello != "go-crypto-tls-1.2+h2" || !*c[ControlTLS12][0].Verified {
		t.Errorf("hellos: %s %s %s", c[ControlAlternate][0].ClientHello, c[ControlBrowser][0].ClientHello, c[ControlTLS12][0].ClientHello)
	}
	for _, s := range c[ControlStatic][:1] { // same SNI, other address: blocked too (the second is IPv6)
		if deref(s.Failure) != "connection_reset" {
			t.Errorf("static %s: %q", s.Address, deref(s.Failure))
		}
	}
}

// Fingerprint blocking: chat_android's hello is reset; libsignal's (alternate_hello) completes on
// the same address and SNI, and the decoy, which keeps the target's hello, fails like it.
func TestQAControlsFingerprintBlocking(t *testing.T) {
	env := newQAEnv(t, nil)
	defer env.Close()
	env.DPIEngine().AddRule(&netem.DPIResetTrafficForString{Logger: log.Log, ServerIPAddress: qaChatAddr, ServerPort: 443,
		String: qaHybridThenX25519})
	_, byName := runQA(t, env, Config{})
	r := byName["chat_android"]
	expectFailure(t, r, "tls_handshake", "connection_reset")
	c := controlsOf(r)
	if alt := c[ControlAlternate][0]; alt.Failure != nil || alt.ClientHello != HelloLibsignal.Name || alt.LeafKeyType == "" {
		t.Errorf("alternate_hello: %+v failure=%q", alt, deref(alt.Failure))
	}
	if decoy := c[ControlDecoySNI][0]; deref(decoy.Failure) != "connection_reset" {
		t.Errorf("decoy: %q", deref(decoy.Failure))
	}
	if tls12 := c[ControlTLS12][0]; tls12.Failure != nil { // Go offers no ML-KEM at TLS 1.2
		t.Errorf("tls12: %q", deref(tls12.Failure))
	}
}

// A refused upgrade is the server's answer (TLS is verified against Signal's roots): no controls
// run, and the refusal is recorded, scrubbed of IP addresses.
func TestQAControlsNotAfterUpgradeRefused(t *testing.T) {
	env := newQAEnv(t, map[string]qaBehavior{qaChatAddr: {refuseUpgrade: true}})
	defer env.Close()
	_, byName := runQA(t, env, Config{})
	r := byName["chat_android"]
	expectFailure(t, r, "websocket_upgrade", FailureWebSocketUpgradeRefused)
	if len(r.Controls) != 0 {
		t.Errorf("controls %s", strings.Join(kindsOf(r), ","))
	}
	resp := r.Response
	if resp == nil || resp.Status != 403 || resp.Headers["Server"] != "cloudflare" || resp.BodyIsTruncated ||
		!strings.Contains(string(resp.Body), "Access denied") {
		t.Fatalf("response %+v", resp)
	}
	data, err := json.Marshal(resp)
	if err != nil || strings.Contains(string(data), "203.0.113.7") {
		t.Errorf("response not scrubbed: %s %v", data, err)
	}
}

// A provisioning step that fails after Signal answered gets no controls by default.
func TestQAControlsNotAfterSignalAnswered(t *testing.T) {
	env := newQAEnv(t, map[string]qaBehavior{qaChatAddr: {noAddress: true}})
	defer env.Close()
	_, byName := runQA(t, env, Config{}, func(m *Measurer) { m.limits.address = 300 * time.Millisecond })
	expectFailure(t, byName["chat_android"], "provisioning_address", "generic_timeout_error")
	expectNoControls(t, byName, allTargets...)
}

func TestQAControlsOff(t *testing.T) {
	env := newQAEnv(t, nil)
	defer env.Close()
	env.DPIEngine().AddRule(&netem.DPIResetTrafficForTLSSNI{Logger: log.Log, SNI: "grpc.chat.signal.org"})
	tk, byName := runQA(t, env, Config{Controls: ControlsOff})
	expectNoControls(t, byName, allTargets...)
	data, _ := json.Marshal(tk.Targets[0])
	if !strings.Contains(string(data), `"controls":null`) {
		t.Errorf("controls not null: %s", data)
	}
}

// always: a baseline for every target; identical controls run once and are shared.
func TestQAControlsAlways(t *testing.T) {
	env := newQAEnv(t, nil)
	defer env.Close()
	tk, byName := runQA(t, env, Config{Controls: ControlsAlways})
	expectOK(t, byName, allTargets...)
	for _, name := range allTargets {
		c := controlsOf(byName[name])
		for _, kind := range []string{ControlDoH, ControlUDP, ControlDecoySNI, ControlAlternate, ControlBrowser, ControlTLS12} {
			if len(c[kind]) != 1 {
				t.Errorf("%s: %d %s controls", name, len(c[kind]), kind)
			}
		}
	}
	// updates2 and updates2_desktop share their DNS controls (same host)
	a, b := controlsOf(byName["updates2"]), controlsOf(byName["updates2_desktop"])
	if a[ControlDoH][0] != b[ControlDoH][0] || a[ControlUDP][0] != b[ControlUDP][0] {
		t.Error("DNS controls of updates2 and updates2_desktop were not shared")
	}
	if a[ControlBrowser][0] != b[ControlBrowser][0] { // same kind, address, SNI and hello
		t.Error("browser_hello of updates2 and updates2_desktop was not shared")
	}
	if a[ControlDecoySNI][0] == b[ControlDecoySNI][0] || a[ControlTLS12][0] == b[ControlTLS12][0] {
		t.Error("controls with different hellos (or ALPNs) were shared")
	}
	if a[ControlTLS12][0].ClientHello == b[ControlTLS12][0].ClientHello {
		t.Errorf("tls12 hellos with different ALPNs have one name: %s", a[ControlTLS12][0].ClientHello)
	}
	if tk.ControlsTruncated {
		t.Error("truncated")
	}
	// tls12 against the fake chat front (which accepts TLS 1.2) completes
	if x := controlsOf(byName["chat_libsignal"])[ControlTLS12][0]; x.Failure != nil {
		t.Errorf("tls12: %q", deref(x.Failure))
	}
}

// When the test's clock runs out during the controls, those not started are not recorded
// (signal_controls_truncated), and those cut short are interrupted, not timeouts.
func TestQAControlsCutByTheClock(t *testing.T) {
	env := newQAEnv(t, nil)
	defer env.Close()
	env.DPIEngine().AddRule(&netem.DPISpoofDNSResponse{Logger: log.Log, Domain: "chat.signal.org"}) // fails at once
	env.DPIEngine().AddRule(&netem.DPIDropTrafficForServerEndpoint{Logger: log.Log, ServerIPAddress: netemx.AddressCloudflareDNSCom1111,
		ServerPort: 443, ServerProtocol: layers.IPProtocolTCP}) // the first control (doh) is still running when the clock runs out
	tk, byName := runQA(t, env, Config{}, func(m *Measurer) {
		m.limits.test, m.limits.controlsParallelism = 1500*time.Millisecond, 1 // the doh control takes about 2 s here
	})
	r := byName["chat_android"]
	if !tk.ControlsTruncated {
		for _, c := range r.Controls {
			t.Logf("%s %s %q t0=%.2f t=%.2f", c.Kind, c.Resolver, deref(c.Failure), c.T0, c.T)
		}
		t.Fatalf("not truncated: chat_android %q %q", r.FailedOperation, deref(r.Failure))
	}
	if len(r.Controls) != 1 || r.Controls[0].Kind != ControlDoH || deref(r.Controls[0].Failure) != "interrupted" {
		t.Fatalf("controls %+v", r.Controls)
	}
}

// SignalCA replaces the roots: with the network's CA everything verifies and signal_roots says
// which root was used; with another root, Signal's hosts fail verification.
func TestQASignalCA(t *testing.T) {
	env := newQAEnv(t, nil)
	defer env.Close()
	ca := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: env.ClientStack.CACert().Raw}))
	tk, byName := runQA(t, env, Config{SignalCA: ca, Controls: ControlsOff})
	expectOK(t, byName, allTargets...)
	sum := sha256.Sum256(env.ClientStack.CACert().Raw)
	if len(tk.SignalRoots) != 1 || tk.SignalRoots[0] != hex.EncodeToString(sum[:]) {
		t.Errorf("signal_roots %v", tk.SignalRoots)
	}

	tk, byName = runQA(t, env, Config{SignalCA: otherRoot(t), Controls: ControlsOff})
	for _, name := range allTargets {
		expectFailure(t, byName[name], "tls_handshake", "ssl_unknown_authority")
	}
	if len(tk.SignalRoots) != 1 || tk.SignalRoots[0] == hex.EncodeToString(sum[:]) {
		t.Errorf("signal_roots %v", tk.SignalRoots)
	}
}

// StaticAddresses replaces a host's candidates, recorded with address_source "option".
func TestQAControlsStaticAddressesOption(t *testing.T) {
	env := newQAEnv(t, nil)
	defer env.Close()
	env.DPIEngine().AddRule(&netem.DPISpoofDNSResponse{Logger: log.Log, Domain: "chat.signal.org"})
	_, byName := runQA(t, env, Config{StaticAddresses: "chat.signal.org=76.223.92.165"})
	static := controlsOf(byName["chat_android"])[ControlStatic]
	// the option's address, then the DoH answer, which is no longer in any list
	if len(static) != 2 || static[0].Address != "76.223.92.165:443" || static[0].AddressSource != "option" ||
		static[0].Failure != nil || *static[0].Status != 101 ||
		static[1].Address != qaChatAddr+":443" || static[1].AddressSource != "doh" || static[1].Failure != nil {
		for _, s := range static {
			t.Logf("%+v failure=%q", *s, deref(s.Failure))
		}
		t.Fatal("unexpected static controls")
	}
}

// SignalCA also applies to the controls that verify; the unverified ones are unaffected.
func TestQAControlsUseSignalCA(t *testing.T) {
	env := newQAEnv(t, nil)
	defer env.Close()
	env.DPIEngine().AddRule(&netem.DPIResetTrafficForTLSSNI{Logger: log.Log, SNI: "grpc.chat.signal.org"})
	_, byName := runQA(t, env, Config{SignalCA: otherRoot(t)})
	c := controlsOf(byName["chat_android"]) // its target failed verification: tls_handshake
	if s := c[ControlStatic][0]; deref(s.Failure) != "ssl_unknown_authority" {
		t.Errorf("static_address: %q", deref(s.Failure))
	}
	if x := c[ControlAlternate][0]; deref(x.Failure) != "ssl_unknown_authority" {
		t.Errorf("alternate_hello: %q", deref(x.Failure))
	}
	if x := c[ControlDecoySNI][0]; x.Failure != nil {
		t.Errorf("decoy_sni (unverified): %q", deref(x.Failure))
	}
	if x := c[ControlBrowser][0]; x.Failure != nil {
		t.Errorf("browser_hello (unverified): %q", deref(x.Failure))
	}
}
