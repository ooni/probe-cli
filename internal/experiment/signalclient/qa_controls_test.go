package signalclient

//
// QA scenarios for how the controls run (when, which, shared, cut by the clock) and for the
// SignalCA and StaticAddresses options. The blocking methods are in qa_conclusions_test.go.
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
