package signalclient

//
// QA scenarios for the targets, in the simulated network of qa_servers_test.go: DPI rules and
// misbehaving servers for each way the experiment should detect interference.
//

import (
	"context"
	"crypto/x509"
	"encoding/binary"
	"strings"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
	utls "github.com/refraction-networking/utls"
)

// qaOption changes the Measurer of a QA run (its targets, its limits).
type qaOption func(*Measurer)

// withTargets measures targets instead of DefaultTargets.
func withTargets(targets ...Target) qaOption {
	return func(m *Measurer) { m.targets = targets }
}

// runQA runs the experiment inside env with config and returns its test keys, and its targets by
// name. In the simulated network most static addresses have no host, and a SYN to them is
// dropped rather than refused, so the controls' handshakes get 2 s instead of 5 unless an option
// says otherwise.
func runQA(t *testing.T, env *qaEnv, config Config, options ...qaOption) (*TestKeys, map[string]*TargetResult) {
	t.Helper()
	var tk *TestKeys
	env.Do(func() {
		m := NewExperimentMeasurer(config).(*Measurer)
		m.limits.controlHandshake = 2 * time.Second
		for _, option := range options {
			option(m)
		}
		m.roots = func() ([]*x509.Certificate, error) { return []*x509.Certificate{env.ClientStack.CACert()}, nil }
		m.googleRoots = func() (*x509.CertPool, error) { return env.ClientStack.DefaultCertPool(), nil }
		meas := &model.Measurement{MeasurementStartTimeSaved: time.Now()}
		args := &model.ExperimentArgs{
			Callbacks:   model.NewPrinterCallbacks(model.DiscardLogger),
			Measurement: meas,
			Session:     &mocks.Session{MockLogger: func() model.Logger { return model.DiscardLogger }},
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		if err := m.Run(ctx, args); err != nil {
			t.Fatal(err)
		}
		tk = meas.TestKeys.(*TestKeys)
	})
	byName := map[string]*TargetResult{}
	for _, r := range tk.Targets {
		byName[r.Name] = r
	}
	return tk, byName
}

// expectOK fails unless every named target succeeded.
func expectOK(t *testing.T, byName map[string]*TargetResult, names ...string) {
	t.Helper()
	for _, n := range names {
		if r := byName[n]; r == nil || r.Failure != nil {
			t.Errorf("%s: expected success, got %+v failure=%q", n, r, deref(r.Failure))
		}
	}
}

// expectFailure fails unless the target failed at operation with failure.
func expectFailure(t *testing.T, r *TargetResult, operation, failure string) {
	t.Helper()
	if r.FailedOperation != operation || deref(r.Failure) != failure {
		t.Errorf("%s: expected %s/%s, got %q/%q", r.Name, operation, failure, r.FailedOperation, deref(r.Failure))
	}
}

// qaHybridThenX25519 is X25519MLKEM768 followed by x25519 in a hello's supported groups, which
// Signal-Android's and Desktop's hellos list and libsignal's does not: a fingerprinting censor's
// pattern.
var qaHybridThenX25519 = string(binary.BigEndian.AppendUint16(binary.BigEndian.AppendUint16(nil,
	uint16(utls.X25519MLKEM768)), uint16(utls.X25519)))

// allTargets are the names of DefaultTargets.
var allTargets = func() (names []string) {
	for _, target := range DefaultTargets {
		names = append(names, target.Name)
	}
	return names
}()

func TestQAEverythingReachable(t *testing.T) {
	env := newQAEnv(t, nil)
	defer env.Close()
	tk, byName := runQA(t, env, Config{})
	expectOK(t, byName, allTargets...)
	if tk.SignalServiceStatus != "ok" {
		t.Errorf("service status %q", tk.SignalServiceStatus)
	}
	if len(tk.TLSHandshakes) != len(allTargets) {
		t.Errorf("expected %d TLS handshakes, got %d", len(allTargets), len(tk.TLSHandshakes))
	}
	for _, name := range []string{"chat_libsignal", "chat_android"} {
		r := byName[name]
		p := r.Provisioning
		if !r.SignalTimestamp || p == nil || !p.AddressReceived || !p.Acked || len(p.Keepalives) != 3 || p.PingEchoed == nil || !*p.PingEchoed {
			t.Errorf("%s: incomplete provisioning: %+v %+v", name, r, p)
		}
	}
	if byName["chat_libsignal"].ALPN != "h2" || byName["chat_android"].ALPN != "http/1.1" {
		t.Errorf("unexpected ALPN: %q %q", byName["chat_libsignal"].ALPN, byName["chat_android"].ALPN)
	}
	if byName["cdsi"].Status != 404 {
		t.Errorf("enclave status %d", byName["cdsi"].Status)
	}
	if tk.MeasurementSummaryKeys().Anomaly() {
		t.Error("anomaly with everything reachable")
	}
	if s := byName["storage"]; s.Response == nil || s.Response.Status != s.Status {
		t.Errorf("https_get response not recorded: %+v", s.Response)
	}
	if byName["chat_android"].Response != nil {
		t.Error("a successful upgrade recorded a response")
	}
}

func TestQADNSBlocking(t *testing.T) {
	env := newQAEnv(t, nil)
	defer env.Close()
	env.DPIEngine().AddRule(&netem.DPISpoofDNSResponse{Logger: log.Log, Domain: "chat.signal.org"}) // no addresses: NXDOMAIN
	_, byName := runQA(t, env, Config{})
	expectFailure(t, byName["chat_android"], "resolve", "dns_nxdomain_error")
	expectOK(t, byName, "chat_libsignal", "cdsi", "storage")
}

func TestQAIPBlocking(t *testing.T) {
	env := newQAEnv(t, nil)
	defer env.Close()
	env.DPIEngine().AddRule(&netem.DPICloseConnectionForServerEndpoint{Logger: log.Log, ServerIPAddress: qaGRPCChatAddr, ServerPort: 443})
	_, byName := runQA(t, env, Config{})
	expectFailure(t, byName["chat_libsignal"], "connect", "connection_refused")
	expectOK(t, byName, "chat_android", "cdsi", "storage")
}

func TestQASNIBlocking(t *testing.T) {
	env := newQAEnv(t, nil)
	defer env.Close()
	env.DPIEngine().AddRule(&netem.DPIResetTrafficForTLSSNI{Logger: log.Log, SNI: "grpc.chat.signal.org"})
	tk, byName := runQA(t, env, Config{})
	expectFailure(t, byName["chat_libsignal"], "tls_handshake", "connection_reset")
	expectOK(t, byName, "chat_android", "cdsi", "storage")
	if !tk.MeasurementSummaryKeys().Anomaly() {
		t.Error("no anomaly with chat_libsignal reset")
	}
}

// TestQAFingerprintBlocking resets ClientHellos that list X25519MLKEM768 then x25519 in their
// groups (Signal-Android's and Desktop's, not libsignal's) on both chat front ends: the two chat
// targets reach the same service with different hellos, so only chat_android fails.
func TestQAFingerprintBlocking(t *testing.T) {
	env := newQAEnv(t, nil)
	defer env.Close()
	env.DPIEngine().AddRule(&netem.DPIResetTrafficForString{Logger: log.Log, ServerIPAddress: qaChatAddr, ServerPort: 443,
		String: qaHybridThenX25519})
	env.DPIEngine().AddRule(&netem.DPIResetTrafficForString{Logger: log.Log, ServerIPAddress: qaGRPCChatAddr, ServerPort: 443,
		String: qaHybridThenX25519})
	_, byName := runQA(t, env, Config{})
	expectFailure(t, byName["chat_android"], "tls_handshake", "connection_reset")
	expectOK(t, byName, "chat_libsignal")
}

func TestQAInjectedReplyWithoutTimestamp(t *testing.T) {
	env := newQAEnv(t, map[string]fault{qaChatAddr: faultNoTimestamp})
	defer env.Close()
	_, byName := runQA(t, env, Config{})
	r := byName["chat_android"]
	if r.Failure != nil || r.SignalTimestamp {
		t.Fatalf("expected success without timestamp, got %+v failure=%q", r, deref(r.Failure))
	}
	for _, k := range r.Provisioning.Keepalives {
		if k.SignalTimestamp {
			t.Errorf("keepalive reported a timestamp: %+v", k)
		}
	}
	if !byName["chat_libsignal"].SignalTimestamp {
		t.Error("chat_libsignal lost its timestamp")
	}
}

func TestQAProvisioningAddressNeverArrives(t *testing.T) {
	env := newQAEnv(t, map[string]fault{qaGRPCChatAddr: faultNoAddress, qaChatAddr: faultNoAddress})
	defer env.Close()
	_, byName := runQA(t, env, Config{}, func(m *Measurer) { m.limits.address = 500 * time.Millisecond })
	expectFailure(t, byName["chat_libsignal"], "provisioning_address", "generic_timeout_error")
	expectFailure(t, byName["chat_android"], "provisioning_address", "generic_timeout_error")
}

func TestQASessionKilledAfterUpgrade(t *testing.T) {
	env := newQAEnv(t, map[string]fault{qaGRPCChatAddr: faultCloseAtStart, qaChatAddr: faultCloseAtStart})
	defer env.Close()
	_, byName := runQA(t, env, Config{})
	expectFailure(t, byName["chat_libsignal"], "provisioning_address", "eof_error")
	expectFailure(t, byName["chat_android"], "provisioning_address", "eof_error")
}

func TestQANoExtendedConnect(t *testing.T) {
	env := newQAEnv(t, map[string]fault{qaGRPCChatAddr: faultNoConnectProtocol})
	defer env.Close()
	_, byName := runQA(t, env, Config{})
	expectFailure(t, byName["chat_libsignal"], "websocket_upgrade", FailureExtendedConnectNotSupported)
	expectOK(t, byName, "chat_android")
}

func TestQASignalReportsOutage(t *testing.T) {
	env := newQAEnv(t, nil)
	defer env.Close()
	env.AddRecordToAllResolvers("uptime.signal.org", "", "127.0.0.2")
	tk, _ := runQA(t, env, Config{})
	if tk.SignalServiceStatus != "outage" {
		t.Fatalf("service status %q", tk.SignalServiceStatus)
	}
	if tk.MeasurementSummaryKeys().Anomaly() { // the targets were reached: the beacon alone is no anomaly
		t.Error("anomaly from the beacon alone")
	}
}

func TestQAUpgradeRefused(t *testing.T) {
	env := newQAEnv(t, map[string]fault{qaGRPCChatAddr: faultRefuseUpgrade, qaChatAddr: faultRefuseUpgrade})
	defer env.Close()
	tk, byName := runQA(t, env, Config{})
	if tk.MeasurementSummaryKeys().Anomaly() { // a refusal inside Signal-verified TLS is the server's
		t.Error("anomaly for a refusal by the server")
	}
	for _, name := range []string{"chat_libsignal", "chat_android"} {
		expectFailure(t, byName[name], "websocket_upgrade", FailureWebSocketUpgradeRefused)
		if byName[name].Status != 403 {
			t.Errorf("%s: status %d", name, byName[name].Status)
		}
	}
	expectOK(t, byName, "cdsi", "storage")
}

// A 101 with the wrong Sec-WebSocket-Accept is not an upgrade: the apps' clients reject it.
func TestQAUpgradeWithWrongAccept(t *testing.T) {
	env := newQAEnv(t, map[string]fault{qaChatAddr: faultBadAccept})
	defer env.Close()
	_, byName := runQA(t, env, Config{})
	expectFailure(t, byName["chat_android"], "websocket_upgrade", FailureWebSocketUpgradeInvalid)
	if byName["chat_android"].Status != 101 {
		t.Errorf("status %d", byName["chat_android"].Status)
	}
	expectOK(t, byName, "chat_libsignal")
}

func TestQAUserAgentIdentifiesOONIByDefault(t *testing.T) {
	env := newQAEnv(t, nil)
	defer env.Close()
	_, byName := runQA(t, env, Config{})
	seen := env.uas.seen()
	for ua := range seen {
		if !strings.HasPrefix(ua, "ooniprobe-signal_client/") || !strings.Contains(ua, "(+https://ooni.org)") {
			t.Errorf("server saw a non-identifying User-Agent %q from %v", ua, seen[ua])
		}
	}
	if got := byName["chat_libsignal"].UserAgent; !strings.HasSuffix(got, " libsignal/0.105.0") {
		t.Errorf("chat_libsignal recorded %q", got)
	}
	if len(seen) != 2 { // with and without libsignal's token
		t.Errorf("expected 2 distinct User-Agents, got %v", seen)
	}
}

func TestQAAppUserAgent(t *testing.T) {
	env := newQAEnv(t, nil)
	defer env.Close()
	_, byName := runQA(t, env, Config{AppUserAgent: true})
	seen := env.uas.seen()
	for ua, hosts := range seen {
		if strings.Contains(ua, "ooniprobe") {
			t.Errorf("AppUserAgent still identified as OONI to %v: %q", hosts, ua)
		}
	}
	for name, want := range map[string]string{
		"chat_libsignal":   "Signal-Android/8.31.1 Android/34 libsignal/0.105.0",
		"chat_android":     "Signal-Android/8.31.1 Android/34",
		"updates2_desktop": "Signal-Desktop/8.33.0 Linux",
	} {
		if got := byName[name].UserAgent; got != want {
			t.Errorf("%s: recorded %q, want %q", name, got, want)
		}
		if _, ok := seen[want]; !ok {
			t.Errorf("%s: server never saw %q (saw %v)", name, want, seen)
		}
	}
}

// An enclave accepting the upgrade without credentials is unexpected, but Signal answered.
func TestQAEnclaveAcceptingUpgradeIsReached(t *testing.T) {
	env := newQAEnv(t, map[string]fault{qaEnclaveAddr: faultEnclaveAccepts})
	defer env.Close()
	_, byName := runQA(t, env, Config{})
	for _, name := range []string{"cdsi", "svr2", "svrb"} {
		r := byName[name]
		if r.Failure != nil || r.FailedOperation != "" || r.Status != 101 {
			t.Errorf("%s: expected reached with 101, got status %d op %q failure %q", name, r.Status, r.FailedOperation, deref(r.Failure))
		}
	}
}

// A hello that cannot be turned into a spec fails at the TLS step, after TCP connected.
func TestQABrokenHelloIsATLSFailure(t *testing.T) {
	env := newQAEnv(t, nil)
	defer env.Close()
	broken := Hello{Name: "broken", raw: []byte{0x01, 0x00, 0x00, 0x01, 0x00}}
	_, byName := runQA(t, env, Config{}, withTargets(Target{"broken", "storage.signal.org", ClientAndroid, broken, MethodHTTPSGet, "/"}))
	r := byName["broken"]
	if r.FailedOperation != "tls_handshake" || r.Failure == nil {
		t.Fatalf("expected a tls_handshake failure, got op %q failure %q", r.FailedOperation, deref(r.Failure))
	}
}

func TestQAEnclave401IsReached(t *testing.T) {
	env := newQAEnv(t, map[string]fault{qaEnclaveAddr: faultEnclave401})
	defer env.Close()
	_, byName := runQA(t, env, Config{})
	for _, name := range []string{"cdsi", "svr2", "svrb"} {
		if r := byName[name]; r.Failure != nil || r.Status != 401 {
			t.Errorf("%s: status %d failure %q", name, r.Status, deref(r.Failure))
		}
	}
}

func TestQAPermessageDeflateNegotiated(t *testing.T) {
	env := newQAEnv(t, map[string]fault{qaChatAddr: deflate})
	defer env.Close()
	_, byName := runQA(t, env, Config{})
	r := byName["chat_android"]
	if r.Failure != nil || r.WSExtensions != "permessage-deflate" || r.Provisioning == nil || !r.Provisioning.AddressReceived ||
		len(r.Provisioning.Keepalives) != 3 {
		t.Fatalf("deflate flow: %+v failure=%q", r, deref(r.Failure))
	}
}

func TestQAStreamResetAfterAddress(t *testing.T) {
	env := newQAEnv(t, map[string]fault{qaGRPCChatAddr: faultResetAfterAddress})
	defer env.Close()
	_, byName := runQA(t, env, Config{})
	r := byName["chat_libsignal"]
	if r.Failure == nil || r.Provisioning == nil || !r.Provisioning.AddressReceived {
		t.Fatalf("expected a failure after the address, got %+v", r)
	}
	if deref(r.Failure) != FailureHTTP2StreamReset {
		t.Errorf("stream reset recorded as %q at %q", deref(r.Failure), r.FailedOperation)
	}
}

func TestQAConnectionKilledAfterAddress(t *testing.T) {
	env := newQAEnv(t, map[string]fault{qaGRPCChatAddr: faultKillAfterAddress, qaChatAddr: faultKillAfterAddress})
	defer env.Close()
	_, byName := runQA(t, env, Config{})
	for _, name := range []string{"chat_libsignal", "chat_android"} {
		r := byName[name]
		if r.Failure == nil || !r.Provisioning.AddressReceived {
			t.Errorf("%s: expected a failure after the address, got %+v", name, r)
			continue
		}
		if f := deref(r.Failure); f != "eof_error" && f != "connection_reset" {
			t.Errorf("%s: kill recorded as %q at %q", name, f, r.FailedOperation)
		}
	}
}

func TestQATruncatedBodyIsAFailure(t *testing.T) {
	env := newQAEnv(t, map[string]fault{qaCDNAddr: faultTruncateBody})
	defer env.Close()
	_, byName := runQA(t, env, Config{})
	for name, alpn := range map[string]string{"storage": "h2", "updates2_desktop": ""} { // both protocols
		r := byName[name]
		if r.ALPN != alpn {
			t.Errorf("%s: negotiated ALPN %q, so the test no longer covers %q", name, r.ALPN, alpn)
		}
		if r.Failure == nil || r.Status != 200 {
			t.Errorf("%s: expected status 200 with a body failure, got status %d failure %q", name, r.Status, deref(r.Failure))
		}
	}
}

// A TLS blackhole costs the operation limit, not the whole provisioning budget.
func TestQAConnectBudgetIsEnforced(t *testing.T) {
	const op = 2 * time.Second
	env := newQAEnv(t, nil)
	defer env.Close()
	env.DPIEngine().AddRule(&netem.DPIDropTrafficForTLSSNI{Logger: log.Log, SNI: "chat.signal.org"})
	_, byName := runQA(t, env, Config{}, func(m *Measurer) { m.limits.op, m.limits.controlHandshake = op, time.Second })
	r := byName["chat_android"]
	expectFailure(t, r, "tls_handshake", "generic_timeout_error")
	if took := r.T - r.T0; took > op.Seconds()+1 {
		t.Errorf("chat_android took %.1f s with a %v operation limit", took, op)
	}
	expectOK(t, byName, "chat_libsignal")
}

func TestQATLSRecordsAreTaggedWithTheTarget(t *testing.T) {
	env := newQAEnv(t, nil)
	defer env.Close()
	tk, _ := runQA(t, env, Config{})
	seen := map[string]bool{}
	for _, hs := range tk.TLSHandshakes {
		if len(hs.Tags) != 1 {
			t.Fatalf("tags %v", hs.Tags)
		}
		seen[hs.Tags[0]] = true
	}
	for _, name := range allTargets {
		if !seen[name] {
			t.Errorf("no TLS record tagged %q (tags seen: %v)", name, seen)
		}
	}
}

// With every target's TLS blackholed, the targets run concurrently: the phase takes about one
// operation limit, not one per target (ts-041 Time limits).
func TestQAEverythingBlackholedRunsConcurrently(t *testing.T) {
	const op = 2 * time.Second
	env := newQAEnv(t, nil)
	defer env.Close()
	for _, target := range DefaultTargets {
		env.DPIEngine().AddRule(&netem.DPIDropTrafficForTLSSNI{Logger: log.Log, SNI: target.Hostname})
	}
	tk, _ := runQA(t, env, Config{Controls: ControlsOff}, func(m *Measurer) { m.limits.op = op })
	first, last := tk.Targets[0].T0, tk.Targets[0].T
	for _, r := range tk.Targets {
		if r.FailedOperation != "tls_handshake" || deref(r.Failure) != "generic_timeout_error" {
			t.Errorf("%s: %q %q", r.Name, r.FailedOperation, deref(r.Failure))
		}
		first, last = min(first, r.T0), max(last, r.T)
	}
	if took := last - first; took > 2*op.Seconds() {
		t.Errorf("targets took %.1f s with a %v operation limit: not concurrent", took, op)
	}
	if !tk.MeasurementSummaryKeys().Anomaly() {
		t.Error("no anomaly with every target blackholed")
	}
}

// A step cut short by the test's clock is interrupted, not a timeout of the network's.
func TestQAProvisioningCutByTheClock(t *testing.T) {
	const clock = 2 * time.Second // the address step would wait 10 s
	env := newQAEnv(t, map[string]fault{qaChatAddr: faultNoAddress})
	defer env.Close()
	_, byName := runQA(t, env, Config{Controls: ControlsOff}, func(m *Measurer) { m.limits.test = clock })
	expectFailure(t, byName["chat_android"], "provisioning_address", "interrupted")
	expectOK(t, byName, "chat_libsignal", "storage")
	if took := byName["chat_android"].T - byName["chat_android"].T0; took > clock.Seconds()+1 {
		t.Errorf("chat_android took %.1f s with a %v clock", took, clock)
	}
}
