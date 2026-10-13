package signalclient

//
// The rows of ts-041's Possible conclusions table, simulated: each blocking method, what the
// target records, and what its controls find.
//

import (
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/google/go-cmp/cmp"
	"github.com/ooni/netem"
	utls "github.com/refraction-networking/utls"
)

// qaConclusion is one row of the conclusions table.
type qaConclusion struct {
	name      string           // the conclusion
	faults    map[string]fault // how the fake servers misbehave
	configure func(*qaEnv)     // DPI rules and DNS records
	options   []qaOption

	// want is the outcome of each target the row is about; every other target must succeed,
	// without controls.
	want    map[string]qaOutcome
	anomaly bool

	// check, when set, checks details of the evidence beyond the outcomes.
	check func(t *testing.T, tk *TestKeys, byName map[string]*TargetResult)
}

// qaOutcome is how far a target got and what its controls found, each as "ok" or the failed
// operation and the failure. A static_address control is named with its address_source.
type qaOutcome struct {
	Target   string
	Controls []string
}

func outcomeOf(r *TargetResult) qaOutcome {
	out := qaOutcome{Target: result(r.FailedOperation, r.Failure)}
	for _, c := range r.Controls {
		kind := c.Kind
		if c.Kind == ControlStatic {
			kind += "(" + c.AddressSource + ")"
		}
		out.Controls = append(out.Controls, kind+": "+result(c.FailedOperation, c.Failure))
	}
	return out
}

func result(operation string, failure *string) string {
	if failure == nil {
		return "ok"
	}
	return operation + " " + *failure
}

// netem has no IPv6: the IPv6 static addresses fail at connect (and the doh control's fallback to
// Cloudflare's IPv6 resolver cannot be exercised).
const qaNoIPv6 = "connect host_unreachable"

// qaGRPCStaticAddr is grpc.chat.signal.org's static address that has a simulated host.
const qaGRPCStaticAddr = "76.223.66.180"

// TLS 1.3 hellos carry supported_versions: libsignal's offers TLS 1.3 only, Android's and
// Chrome's end with TLS 1.3 then 1.2. Go's TLS 1.2 hello (the tls12 control) has neither.
const (
	qaLibsignalSupportedVersions = "\x00\x2b\x00\x03\x02\x03\x04"
	qaTLS13Then12                = "\x03\x04\x03\x03"
)

// qaTLSRecordOfAHello starts the first TLS record of every ClientHello sent here.
const qaTLSRecordOfAHello = "\x16\x03\x01"

// targetNamed returns the default target with name.
func targetNamed(name string) Target {
	for _, target := range DefaultTargets {
		if target.Name == name {
			return target
		}
	}
	panic("no target " + name)
}

// qaHybridThenX25519 is X25519MLKEM768 followed by x25519 in a hello's supported groups, which
// Signal-Android's and Desktop's hellos list and libsignal's does not: a fingerprinting censor's
// pattern.
var qaHybridThenX25519 = string(binary.BigEndian.AppendUint16(binary.BigEndian.AppendUint16(nil,
	uint16(utls.X25519MLKEM768)), uint16(utls.X25519)))

// qaForgeChatToInterceptor makes the system resolver answer chat.signal.org with qaInterceptorAddr.
func qaForgeChatToInterceptor(env *qaEnv) {
	env.ISPResolverConfig().AddRecord("chat.signal.org", "", qaInterceptorAddr)
}

// qaInterceptedChat is chat_android's outcome when the interceptor's certificate fails with
// failure: the handshakes that verify fail like the target, those that do not complete, and the
// static addresses reach Signal.
func qaInterceptedChat(failure string) map[string]qaOutcome {
	return map[string]qaOutcome{"chat_android": {
		Target: "tls_handshake " + failure,
		Controls: []string{
			"doh: ok",
			"udp_1111: ok",
			"static_address(android): ok",
			"static_address(android): ok",
			"static_address(desktop): " + qaNoIPv6,
			"decoy_sni: ok", // not verified
			"alternate_hello: tls_handshake " + failure,
			"browser_hello: ok", // not verified
			"tls12: tls_handshake " + failure,
		}}}
}

// qaInterceptorCertificateRecorded checks that the interceptor's certificates are in the evidence.
func qaInterceptorCertificateRecorded(t *testing.T, tk *TestKeys, _ map[string]*TargetResult) {
	for _, hs := range tk.TLSHandshakes {
		if strings.Join(hs.Tags, ",") == "chat_android" && hs.Address == qaInterceptorAddr+":443" &&
			len(hs.PeerCertificates) > 0 {
			return
		}
	}
	t.Error("no chat_android handshake with the interceptor's certificates")
}

func TestQAConclusions(t *testing.T) {
	for _, c := range []qaConclusion{{
		// NXDOMAIN from the system resolver and from 1.1.1.1:53, which the DPI also sees
		name: "DNS blocking",
		configure: func(env *qaEnv) {
			env.DPIEngine().AddRule(&netem.DPISpoofDNSResponse{Logger: log.Log, Domain: "chat.signal.org"})
		},
		want: map[string]qaOutcome{"chat_android": {
			Target: "resolve dns_nxdomain_error",
			Controls: []string{
				"doh: ok",
				"udp_1111: resolve dns_nxdomain_error",
				"static_address(android): ok",
				"static_address(android): ok",
				"static_address(desktop): " + qaNoIPv6,
			}}},
		anomaly: true,
		check: func(t *testing.T, tk *TestKeys, byName map[string]*TargetResult) {
			c := controlsOf(byName["chat_android"])
			if doh := c[ControlDoH][0]; doh.Resolver != dohResolvers[0] || !contains(*doh.Addresses, qaChatAddr) {
				t.Errorf("doh: %+v", doh)
			}
			if udp := c[ControlUDP][0]; udp.Resolver != udpResolver {
				t.Errorf("udp_1111: %+v", udp)
			}
			for _, s := range c[ControlStatic][:2] { // Signal answered the upgrade
				if *s.Status != 101 || !*s.SignalTimestamp || !*s.Verified {
					t.Errorf("static %s: %+v", s.Address, s)
				}
			}
			for _, q := range tk.Queries { // the controls' evidence is tagged
				if strings.Join(q.Tags, ",") == "chat_android,control:udp_1111" && q.ResolverAddress == udpResolver {
					return
				}
			}
			t.Error("no tagged udp_1111 query")
		},
	}, {
		// the system resolver forges an address; DoH gives the real one, which is in no static
		// list, so it gets its own static_address control
		name: "DNS manipulation (forged answers)",
		configure: func(env *qaEnv) {
			env.ISPResolverConfig().AddRecord("storage.signal.org", "", "10.10.34.35") // nothing there
		},
		options: []qaOption{func(m *Measurer) { m.limits.op = time.Second }},
		want: map[string]qaOutcome{"storage": {
			Target: "connect generic_timeout_error",
			Controls: []string{
				"doh: ok",
				"udp_1111: ok",
				"static_address(android): connect generic_timeout_error", // static addresses without a simulated host
				"static_address(desktop): connect generic_timeout_error",
				"static_address(desktop): " + qaNoIPv6,
				"static_address(doh): ok",
			}}},
		anomaly: true,
		check: func(t *testing.T, _ *TestKeys, byName map[string]*TargetResult) {
			static := controlsOf(byName["storage"])[ControlStatic]
			if s := static[len(static)-1]; s.Address != qaCDNAddr+":443" || *s.Status == 0 { // the server answered
				t.Errorf("static_address from DoH: %+v", s)
			}
		},
	}, {
		// libsignal would connect to the other address
		name: "Partial address blocking",
		configure: func(env *qaEnv) {
			env.DPIEngine().AddRule(&netem.DPICloseConnectionForServerEndpoint{Logger: log.Log,
				ServerIPAddress: qaGRPCChatAddr, ServerPort: 443})
		},
		want: map[string]qaOutcome{"chat_libsignal": {
			Target: "connect connection_refused",
			Controls: []string{
				"doh: ok",
				"udp_1111: ok",
				"static_address(libsignal): ok",
				"static_address(libsignal): " + qaNoIPv6,
			}}},
		anomaly: true,
		check: func(t *testing.T, _ *TestKeys, byName map[string]*TargetResult) {
			s := controlsOf(byName["chat_libsignal"])[ControlStatic][0]
			if *s.Status != 200 || !*s.SignalTimestamp || s.ClientHello != HelloLibsignal.Name {
				t.Errorf("static: %+v", s)
			}
		},
	}, {
		// every address of the host fails at connect
		name: "Address (IP) blocking",
		configure: func(env *qaEnv) {
			for _, addr := range []string{qaGRPCChatAddr, qaGRPCStaticAddr} {
				env.DPIEngine().AddRule(&netem.DPICloseConnectionForServerEndpoint{Logger: log.Log,
					ServerIPAddress: addr, ServerPort: 443})
			}
		},
		want: map[string]qaOutcome{"chat_libsignal": {
			Target: "connect connection_refused",
			Controls: []string{
				"doh: ok",
				"udp_1111: ok",
				"static_address(libsignal): connect connection_refused",
				"static_address(libsignal): " + qaNoIPv6,
			}}},
		anomaly: true,
	}, {
		// the decoy completes on the same address; every real-SNI handshake fails
		name: "SNI-based blocking",
		configure: func(env *qaEnv) {
			env.DPIEngine().AddRule(&netem.DPIResetTrafficForTLSSNI{Logger: log.Log, SNI: "grpc.chat.signal.org"})
		},
		want: map[string]qaOutcome{"chat_libsignal": {
			Target: "tls_handshake connection_reset",
			Controls: []string{
				"doh: ok",
				"udp_1111: ok",
				"static_address(libsignal): tls_handshake connection_reset",
				"static_address(libsignal): " + qaNoIPv6,
				"decoy_sni: ok",
				"alternate_hello: tls_handshake connection_reset",
				"browser_hello: tls_handshake connection_reset",
				"tls12: tls_handshake connection_reset",
			}}},
		anomaly: true,
		check: func(t *testing.T, _ *TestKeys, byName map[string]*TargetResult) {
			c := controlsOf(byName["chat_libsignal"])
			decoy := c[ControlDecoySNI][0]
			if *decoy.Verified || decoy.AddressSource != "target" || decoy.Address != qaGRPCChatAddr+":443" ||
				len(decoy.SNI) != len("grpc.chat.signal.org") || !strings.HasSuffix(decoy.SNI, ".example.com") ||
				decoy.LeafKeyType == "" {
				t.Errorf("decoy: %+v", decoy)
			}
			got := map[string]string{}
			for _, kind := range []string{ControlAlternate, ControlBrowser, ControlTLS12} {
				x := c[kind][0]
				got[kind] = x.SNI + " " + x.ClientHello
			}
			want := map[string]string{
				ControlAlternate: "grpc.chat.signal.org " + HelloAndroidREST.Name,
				ControlBrowser:   "grpc.chat.signal.org " + HelloChrome.Name,
				ControlTLS12:     "grpc.chat.signal.org go-crypto-tls-1.2+h2",
			}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("SNI and hello of the handshake controls (-want +got):\n%s", diff)
			}
			if *c[ControlBrowser][0].Verified || !*c[ControlTLS12][0].Verified {
				t.Error("browser_hello must not verify, tls12 must")
			}
		},
	}, {
		// every hello to the address is reset, the decoy's too; the other address works
		name: "Blocking of TLS to the address",
		configure: func(env *qaEnv) {
			env.DPIEngine().AddRule(&netem.DPIResetTrafficForString{Logger: log.Log, ServerIPAddress: qaGRPCChatAddr,
				ServerPort: 443, String: qaTLSRecordOfAHello})
		},
		want: map[string]qaOutcome{"chat_libsignal": {
			Target: "tls_handshake connection_reset",
			Controls: []string{
				"doh: ok",
				"udp_1111: ok",
				"static_address(libsignal): ok",
				"static_address(libsignal): " + qaNoIPv6,
				"decoy_sni: tls_handshake connection_reset",
				"alternate_hello: tls_handshake connection_reset",
				"browser_hello: tls_handshake connection_reset",
				"tls12: tls_handshake connection_reset",
			}}},
		anomaly: true,
	}, {
		// chat_android's hello is reset on both chat front ends, libsignal's is not: alternate_hello
		// completes on the same address and SNI, and the decoy, which keeps the target's hello,
		// fails like it
		name: "Fingerprint-based blocking",
		configure: func(env *qaEnv) {
			for _, addr := range []string{qaChatAddr, qaGRPCChatAddr} {
				env.DPIEngine().AddRule(&netem.DPIResetTrafficForString{Logger: log.Log, ServerIPAddress: addr,
					ServerPort: 443, String: qaHybridThenX25519})
			}
		},
		want: map[string]qaOutcome{"chat_android": {
			Target: "tls_handshake connection_reset",
			Controls: []string{
				"doh: ok",
				"udp_1111: ok",
				"static_address(android): ok", // the rule is on the target's address only
				"static_address(desktop): " + qaNoIPv6,
				"decoy_sni: tls_handshake connection_reset",
				"alternate_hello: ok",
				"browser_hello: tls_handshake connection_reset", // Chrome also offers X25519MLKEM768 first
				"tls12: ok",                                     // no ML-KEM at TLS 1.2
			}}},
		anomaly: true,
		check: func(t *testing.T, _ *TestKeys, byName map[string]*TargetResult) {
			if alt := controlsOf(byName["chat_android"])[ControlAlternate][0]; alt.ClientHello != HelloLibsignal.Name ||
				alt.LeafKeyType == "" {
				t.Errorf("alternate_hello: %+v", alt)
			}
		},
	}, {
		// every TLS 1.3 hello to the host is reset; Go's TLS 1.2 hello completes
		name: "TLS 1.3-specific interference",
		configure: func(env *qaEnv) {
			for _, addr := range []string{qaGRPCChatAddr, qaGRPCStaticAddr} {
				for _, pattern := range []string{qaLibsignalSupportedVersions, qaTLS13Then12} {
					env.DPIEngine().AddRule(&netem.DPIResetTrafficForString{Logger: log.Log, ServerIPAddress: addr,
						ServerPort: 443, String: pattern})
				}
			}
		},
		want: map[string]qaOutcome{"chat_libsignal": {
			Target: "tls_handshake connection_reset",
			Controls: []string{
				"doh: ok",
				"udp_1111: ok",
				"static_address(libsignal): tls_handshake connection_reset",
				"static_address(libsignal): " + qaNoIPv6,
				"decoy_sni: tls_handshake connection_reset",
				"alternate_hello: tls_handshake connection_reset",
				"browser_hello: tls_handshake connection_reset",
				"tls12: ok", // the simulated server accepts TLS 1.2; the real one answers protocol_version
			}}},
		anomaly: true,
	}, {
		// the system resolver sends the target to a middlebox with its own certificate for the
		// name; DoH's answer is a static address, which reaches Signal
		name:      "Interception: a certificate for the name from another issuer",
		faults:    map[string]fault{qaInterceptorAddr: faultImpostorCertificate},
		configure: qaForgeChatToInterceptor,
		want:      qaInterceptedChat("ssl_unknown_authority"),
		anomaly:   true,
		check:     qaInterceptorCertificateRecorded,
	}, {
		// as above, with a certificate for a blockpage domain: Go checks the name before the
		// issuer, so the failure says hostname even though the certificate is also self-signed
		// (the certificates in tls_handshakes tell the two apart)
		name:      "Interception: a blockpage certificate",
		faults:    map[string]fault{qaInterceptorAddr: faultBlockpageCertificate},
		configure: qaForgeChatToInterceptor,
		want:      qaInterceptedChat("ssl_invalid_hostname"),
		anomaly:   true,
		check:     qaInterceptorCertificateRecorded,
	}, {
		// the GET is reset after the handshake: the controls run and complete their handshakes
		name:   "Possible interference after the handshake: http_round_trip",
		faults: map[string]fault{qaCDNAddr: faultResetRequest},
		options: []qaOption{withTargets(targetNamed("storage")),
			func(m *Measurer) { m.limits.controlHandshake = time.Second }},
		want: map[string]qaOutcome{"storage": {
			Target: "http_round_trip eof_error",
			Controls: []string{
				"doh: ok",
				"udp_1111: ok",
				"static_address(android): connect generic_timeout_error", // static addresses without a simulated host
				"static_address(desktop): connect generic_timeout_error",
				"static_address(desktop): " + qaNoIPv6,
				"decoy_sni: ok",
				"alternate_hello: ok",
			}}},
		anomaly: true,
	}, {
		// only the 16 KiB keepalive fails: the 1 KiB one and the unpadded one get their answers
		name:    "Interference after the handshake: only large messages fail",
		faults:  map[string]fault{qaChatAddr: faultKillLargeMessages},
		want:    map[string]qaOutcome{"chat_android": {Target: "keepalive eof_error"}},
		anomaly: true,
		check: func(t *testing.T, _ *TestKeys, byName map[string]*TargetResult) {
			var got []string
			for _, k := range byName["chat_android"].Provisioning.Keepalives {
				got = append(got, fmt.Sprintf("%d: %s", k.Padding, result("", k.Failure)))
			}
			want := []string{"0: ok", "1024: ok", "16384:  eof_error"}
			if diff := cmp.Diff(want, got); diff != "" {
				t.Errorf("keepalives (-want +got):\n%s", diff)
			}
		},
	}, {
		// after Signal answered the upgrade, a failure gets no controls by default
		name:    "Interference after the handshake: the address never arrives",
		faults:  map[string]fault{qaGRPCChatAddr: faultNoAddress, qaChatAddr: faultNoAddress},
		options: []qaOption{func(m *Measurer) { m.limits.address = 500 * time.Millisecond }},
		want: map[string]qaOutcome{
			"chat_libsignal": {Target: "provisioning_address generic_timeout_error"},
			"chat_android":   {Target: "provisioning_address generic_timeout_error"},
		},
		anomaly: true,
	}, {
		name:   "Interference after the handshake: the session ends after the upgrade",
		faults: map[string]fault{qaGRPCChatAddr: faultCloseAtStart, qaChatAddr: faultCloseAtStart},
		want: map[string]qaOutcome{
			"chat_libsignal": {Target: "provisioning_address eof_error"},
			"chat_android":   {Target: "provisioning_address eof_error"},
		},
		anomaly: true,
	}, {
		// TLS is verified against Signal's roots, so the refusal is the server's (or its CDN's):
		// no controls run, and the refusal is recorded, scrubbed of IP addresses
		name:   "Not censorship by itself: a refused upgrade",
		faults: map[string]fault{qaGRPCChatAddr: faultRefuseUpgrade, qaChatAddr: faultRefuseUpgrade},
		want: map[string]qaOutcome{
			"chat_libsignal": {Target: "websocket_upgrade " + FailureWebSocketUpgradeRefused},
			"chat_android":   {Target: "websocket_upgrade " + FailureWebSocketUpgradeRefused},
		},
		anomaly: false,
		check: func(t *testing.T, _ *TestKeys, byName map[string]*TargetResult) {
			if byName["chat_libsignal"].Status != 403 {
				t.Errorf("chat_libsignal: status %d", byName["chat_libsignal"].Status)
			}
			resp := byName["chat_android"].Response
			if resp == nil || resp.Status != 403 || resp.Headers["Server"] != "cloudflare" || resp.BodyIsTruncated ||
				!strings.Contains(string(resp.Body), "Access denied") {
				t.Fatalf("response %+v", resp)
			}
			data, err := json.Marshal(resp)
			if err != nil || strings.Contains(string(data), "203.0.113.7") {
				t.Errorf("response not scrubbed: %s %v", data, err)
			}
		},
	}} {
		t.Run(c.name, func(t *testing.T) {
			env := newQAEnv(t, c.faults)
			defer env.Close()
			if c.configure != nil {
				c.configure(env)
			}
			tk, byName := runQA(t, env, Config{}, c.options...)
			got := map[string]qaOutcome{}
			for _, r := range tk.Targets {
				if outcome := outcomeOf(r); c.want[r.Name].Target != "" || outcome.Target != "ok" || r.Controls != nil {
					got[r.Name] = outcome
				}
			}
			if diff := cmp.Diff(c.want, got); diff != "" {
				t.Errorf("outcomes (-want +got):\n%s", diff)
			}
			if anomaly := tk.MeasurementSummaryKeys().Anomaly(); anomaly != c.anomaly {
				t.Errorf("anomaly %v, want %v", anomaly, c.anomaly)
			}
			if tk.ControlsTruncated {
				t.Error("controls truncated")
			}
			if c.check != nil {
				c.check(t, tk, byName)
			}
		})
	}
}
