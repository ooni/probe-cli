package signalclient

//
// Integration tests in a simulated network (netemx): fake Signal front ends, DNS, and DPI
// rules for each way the experiment should detect interference.
//

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha1"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
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
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netemx"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/hpack"
)

// Addresses of the simulated Signal front ends (the real ones, for readability).
const (
	qaGRPCChatAddr = "15.197.251.99"  // grpc.chat.signal.org
	qaChatAddr     = "13.248.212.111" // chat.signal.org
	qaEnclaveAddr  = "40.122.45.194"  // cdsi, svr2, svrb
	qaCDNAddr      = "104.18.10.47"   // storage, cdn*, updates2, sfu
)

// qaBehavior configures how a fake front end misbehaves (all false: like Signal).
type qaBehavior struct {
	noTimestamp       bool // replies lack X-Signal-Timestamp, as injected replies would
	noAddress         bool // the provisioning address never arrives
	closeAfterUpgrade bool // the stream ends right after the upgrade
	noConnectProtocol bool // HTTP/2 SETTINGS without ENABLE_CONNECT_PROTOCOL
	refuseUpgrade     bool // answer the provisioning upgrade with 403, as a blockpage would
}

// qaUserAgents records the User-Agent of every request the fake front ends receive, by host.
type qaUserAgents struct {
	mu   sync.Mutex
	byUA map[string][]string // User-Agent -> hosts
}

func (u *qaUserAgents) add(ua, host string) {
	u.mu.Lock()
	defer u.mu.Unlock()
	u.byUA[ua] = append(u.byUA[ua], host)
}

func (u *qaUserAgents) seen() map[string][]string {
	u.mu.Lock()
	defer u.mu.Unlock()
	out := map[string][]string{}
	for k, v := range u.byUA {
		out[k] = append([]string(nil), v...)
	}
	return out
}

// qaSeenUserAgents belongs to the most recent newQAEnv (the QA tests do not run in parallel).
var qaSeenUserAgents *qaUserAgents

// qaSignalServerFactory is a netemx server factory for a fake Signal front end.
type qaSignalServerFactory struct {
	t        *testing.T
	names    []string
	behavior qaBehavior
	uas      *qaUserAgents
}

var _ netemx.NetStackServerFactory = &qaSignalServerFactory{}

func (f *qaSignalServerFactory) MustNewServer(_ netemx.NetStackServerFactoryEnv, stack *netem.UNetStack) netemx.NetStackServer {
	return &qaSignalServer{factory: f, stack: stack}
}

type qaSignalServer struct {
	factory  *qaSignalServerFactory
	stack    *netem.UNetStack
	mu       sync.Mutex
	listener net.Listener
}

func (s *qaSignalServer) MustStart() {
	s.mu.Lock()
	defer s.mu.Unlock()
	addr := &net.TCPAddr{IP: net.ParseIP(s.stack.IPAddress()), Port: 443}
	ln := runtimex.Try1(s.stack.ListenTCP("tcp", addr))
	config := s.stack.MustNewServerTLSConfig(s.factory.names[0], s.factory.names[1:]...)
	config.NextProtos = []string{"h2", "http/1.1"}
	s.listener = tls.NewListener(ln, config)
	go func() {
		for {
			conn, err := s.listener.Accept()
			if err != nil {
				return
			}
			go s.serve(conn.(*tls.Conn))
		}
	}()
}

func (s *qaSignalServer) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.listener != nil {
		s.listener.Close()
	}
	return nil
}

func (s *qaSignalServer) serve(conn *tls.Conn) {
	defer conn.Close()
	if err := conn.Handshake(); err != nil {
		return
	}
	if conn.ConnectionState().NegotiatedProtocol == "h2" {
		s.serveH2(conn)
		return
	}
	s.serveH1(conn)
}

func (s *qaSignalServer) provisioning() fakeServer {
	b := s.factory.behavior
	return fakeServer{sendAddress: !b.noAddress, closeAtStart: b.closeAfterUpgrade, noTimestamp: b.noTimestamp}
}

func (s *qaSignalServer) timestampHeader() string {
	if s.factory.behavior.noTimestamp {
		return ""
	}
	return "X-Signal-Timestamp: 1791563494213\r\n"
}

// serveH1 is chat.signal.org's HTTP/1.1 front end, the enclaves' and the CDNs'.
func (s *qaSignalServer) serveH1(conn net.Conn) {
	br := bufio.NewReader(conn)
	req, err := http.ReadRequest(br)
	if err != nil {
		return
	}
	s.factory.uas.add(req.UserAgent(), req.Host)
	upgrade := strings.EqualFold(req.Header.Get("Upgrade"), "websocket")
	switch {
	case upgrade && s.factory.behavior.refuseUpgrade:
		fmt.Fprintf(conn, "HTTP/1.1 403 Forbidden\r\nContent-Length: 0\r\nConnection: close\r\n\r\n")
	case upgrade && req.URL.Path == "/v1/websocket/provisioning/":
		sum := sha1.Sum([]byte(req.Header.Get("Sec-WebSocket-Key") + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"))
		fmt.Fprintf(conn, "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n"+
			"Sec-WebSocket-Accept: %s\r\n%s\r\n", base64.StdEncoding.EncodeToString(sum[:]), s.timestampHeader())
		s.provisioning().serve(s.factory.t, br, conn)
	case upgrade: // an enclave path without credentials
		fmt.Fprintf(conn, "HTTP/1.1 404 Not Found\r\nContent-Length: 0\r\nConnection: close\r\n\r\n")
	default:
		fmt.Fprintf(conn, "HTTP/1.1 200 OK\r\nContent-Length: 2\r\nConnection: close\r\n%s\r\nok", s.timestampHeader())
	}
}

// serveH2 is grpc.chat.signal.org's HTTP/2 front end: RFC 8441 extended CONNECT to the
// provisioning WebSocket, and plain requests for the CDNs that negotiate h2.
func (s *qaSignalServer) serveH2(conn net.Conn) {
	preface := make([]byte, len(http2.ClientPreface))
	if _, err := io.ReadFull(conn, preface); err != nil || string(preface) != http2.ClientPreface {
		return
	}
	fr := http2.NewFramer(conn, conn)
	fr.ReadMetaHeaders = hpack.NewDecoder(4096, nil)
	var wmu sync.Mutex
	locked := func(fn func() error) error { wmu.Lock(); defer wmu.Unlock(); return fn() }
	settings := []http2.Setting{{ID: http2.SettingMaxConcurrentStreams, Val: 100}, {ID: http2.SettingMaxHeaderListSize, Val: 8192}}
	if !s.factory.behavior.noConnectProtocol {
		settings = append(settings, http2.Setting{ID: http2.SettingEnableConnectProtocol, Val: 1})
	}
	locked(func() error { return fr.WriteSettings(settings...) })
	respond := func(stream uint32, status int, endStream bool) error {
		var buf bytes.Buffer
		enc := hpack.NewEncoder(&buf)
		enc.WriteField(hpack.HeaderField{Name: ":status", Value: fmt.Sprint(status)})
		if !s.factory.behavior.noTimestamp {
			enc.WriteField(hpack.HeaderField{Name: "x-signal-timestamp", Value: "1791563494213"})
		}
		return locked(func() error {
			return fr.WriteHeaders(http2.HeadersFrameParam{StreamID: stream, BlockFragment: buf.Bytes(), EndHeaders: true, EndStream: endStream})
		})
	}
	streams := map[uint32]*io.PipeWriter{}
	defer func() {
		for _, pw := range streams {
			pw.Close()
		}
	}()
	for {
		frame, err := fr.ReadFrame()
		if err != nil {
			return
		}
		switch f := frame.(type) {
		case *http2.SettingsFrame:
			if !f.IsAck() {
				locked(fr.WriteSettingsAck)
			}
		case *http2.MetaHeadersFrame:
			for _, hf := range f.RegularFields() {
				if hf.Name == "user-agent" {
					s.factory.uas.add(hf.Value, f.PseudoValue("authority"))
				}
			}
			if f.PseudoValue("method") == "CONNECT" && f.PseudoValue("protocol") == "websocket" {
				if s.factory.behavior.refuseUpgrade {
					respond(f.StreamID, 403, true)
					continue
				}
				respond(f.StreamID, 200, false)
				pr, pw := io.Pipe()
				streams[f.StreamID] = pw
				w := &qaStreamWriter{fr: fr, mu: &wmu, stream: f.StreamID}
				go func() {
					s.provisioning().serve(s.factory.t, pr, w)
					locked(func() error { return fr.WriteData(f.StreamID, true, nil) })
				}()
				continue
			}
			respond(f.StreamID, 404, true)
		case *http2.DataFrame:
			data := append([]byte(nil), f.Data()...)
			if n := uint32(len(data)); n > 0 {
				locked(func() error {
					fr.WriteWindowUpdate(0, n)
					return fr.WriteWindowUpdate(f.StreamID, n)
				})
			}
			if pw := streams[f.StreamID]; pw != nil {
				pw.Write(data)
				if f.StreamEnded() {
					pw.Close()
				}
			}
		case *http2.RSTStreamFrame:
			if pw := streams[f.StreamID]; pw != nil {
				pw.Close()
			}
		case *http2.GoAwayFrame:
			return
		}
	}
}

// qaStreamWriter writes server frames as HTTP/2 DATA frames on one stream.
type qaStreamWriter struct {
	fr     *http2.Framer
	mu     *sync.Mutex
	stream uint32
}

func (w *qaStreamWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()
	for off := 0; off < len(p); off += 16384 {
		end := min(off+16384, len(p))
		if err := w.fr.WriteData(w.stream, false, p[off:end]); err != nil {
			return off, err
		}
	}
	return len(p), nil
}

// qaEnv is a network where every Signal target resolves to a fake front end.
func newQAEnv(t *testing.T, behaviors map[string]qaBehavior) *netemx.QAEnv {
	qaSeenUserAgents = &qaUserAgents{byUA: map[string][]string{}}
	front := func(addr string, names ...string) netemx.QAEnvOption {
		return netemx.QAEnvOptionNetStack(addr, &qaSignalServerFactory{t: t, names: names, behavior: behaviors[addr], uas: qaSeenUserAgents})
	}
	env := netemx.MustNewQAEnv(
		front(qaGRPCChatAddr, "grpc.chat.signal.org"),
		front(qaChatAddr, "chat.signal.org"),
		front(qaEnclaveAddr, "cdsi.signal.org", "svr2.signal.org", "svrb.signal.org"),
		front(qaCDNAddr, "storage.signal.org", "cdn.signal.org", "cdn2.signal.org", "cdn3.signal.org",
			"updates2.signal.org", "sfu.voip.signal.org"),
		netemx.QAEnvOptionLogger(log.Log),
	)
	env.AddRecordToAllResolvers("grpc.chat.signal.org", "", qaGRPCChatAddr)
	env.AddRecordToAllResolvers("chat.signal.org", "", qaChatAddr)
	for _, h := range []string{"cdsi", "svr2", "svrb"} {
		env.AddRecordToAllResolvers(h+".signal.org", "", qaEnclaveAddr)
	}
	for _, h := range []string{"storage", "cdn", "cdn2", "cdn3", "updates2", "sfu.voip"} {
		env.AddRecordToAllResolvers(h+".signal.org", "", qaCDNAddr)
	}
	env.AddRecordToAllResolvers("uptime.signal.org", "", "127.0.0.1")
	return env
}

// runQA runs the experiment inside env and returns its test keys by target name.
func runQA(t *testing.T, env *netemx.QAEnv) (*TestKeys, map[string]*TargetResult) {
	t.Helper()
	return runQAWith(t, env, Config{})
}

// runQAWith is runQA with a config.
func runQAWith(t *testing.T, env *netemx.QAEnv, config Config) (*TestKeys, map[string]*TargetResult) {
	t.Helper()
	var tk *TestKeys
	env.Do(func() {
		m := NewExperimentMeasurer(config).(*Measurer)
		m.roots = func() (*x509.CertPool, error) { return env.ClientStack.DefaultCertPool(), nil }
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

var allTargets = []string{"chat_libsignal", "chat_android", "cdsi", "svr2", "svrb", "storage", "cdn", "cdn2", "cdn3",
	"updates2", "updates2_desktop", "sfu"}

func TestQAEverythingReachable(t *testing.T) {
	env := newQAEnv(t, nil)
	defer env.Close()
	tk, byName := runQA(t, env)
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
}

func TestQADNSBlocking(t *testing.T) {
	env := newQAEnv(t, nil)
	defer env.Close()
	env.DPIEngine().AddRule(&netem.DPISpoofDNSResponse{Logger: log.Log, Domain: "chat.signal.org"}) // no addresses: NXDOMAIN
	_, byName := runQA(t, env)
	expectFailure(t, byName["chat_android"], "resolve", "dns_nxdomain_error")
	expectOK(t, byName, "chat_libsignal", "cdsi", "storage")
}

func TestQAIPBlocking(t *testing.T) {
	env := newQAEnv(t, nil)
	defer env.Close()
	env.DPIEngine().AddRule(&netem.DPICloseConnectionForServerEndpoint{Logger: log.Log, ServerIPAddress: qaGRPCChatAddr, ServerPort: 443})
	_, byName := runQA(t, env)
	expectFailure(t, byName["chat_libsignal"], "connect", "connection_refused")
	expectOK(t, byName, "chat_android", "cdsi", "storage")
}

func TestQASNIBlocking(t *testing.T) {
	env := newQAEnv(t, nil)
	defer env.Close()
	env.DPIEngine().AddRule(&netem.DPIResetTrafficForTLSSNI{Logger: log.Log, SNI: "grpc.chat.signal.org"})
	_, byName := runQA(t, env)
	expectFailure(t, byName["chat_libsignal"], "tls_handshake", "connection_reset")
	expectOK(t, byName, "chat_android", "cdsi", "storage")
}

// TestQAFingerprintBlocking resets ClientHellos that list X25519MLKEM768 then x25519 in their
// groups (Signal-Android's and Desktop's, not libsignal's) on the chat front end: the two chat
// targets reach the same service with different hellos, so only chat_android fails.
func TestQAFingerprintBlocking(t *testing.T) {
	env := newQAEnv(t, nil)
	defer env.Close()
	env.DPIEngine().AddRule(&netem.DPIResetTrafficForString{Logger: log.Log, ServerIPAddress: qaChatAddr, ServerPort: 443,
		String: "\x11\xec\x00\x1d"})
	env.DPIEngine().AddRule(&netem.DPIResetTrafficForString{Logger: log.Log, ServerIPAddress: qaGRPCChatAddr, ServerPort: 443,
		String: "\x11\xec\x00\x1d"})
	_, byName := runQA(t, env)
	expectFailure(t, byName["chat_android"], "tls_handshake", "connection_reset")
	expectOK(t, byName, "chat_libsignal")
}

func TestQAInjectedReplyWithoutTimestamp(t *testing.T) {
	env := newQAEnv(t, map[string]qaBehavior{qaChatAddr: {noTimestamp: true}})
	defer env.Close()
	_, byName := runQA(t, env)
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
	defer func(d time.Duration) { addressTimeout = d }(addressTimeout)
	addressTimeout = 500 * time.Millisecond
	env := newQAEnv(t, map[string]qaBehavior{qaGRPCChatAddr: {noAddress: true}, qaChatAddr: {noAddress: true}})
	defer env.Close()
	_, byName := runQA(t, env)
	expectFailure(t, byName["chat_libsignal"], "provisioning_address", "generic_timeout_error")
	expectFailure(t, byName["chat_android"], "provisioning_address", "generic_timeout_error")
}

func TestQASessionKilledAfterUpgrade(t *testing.T) {
	env := newQAEnv(t, map[string]qaBehavior{qaGRPCChatAddr: {closeAfterUpgrade: true}, qaChatAddr: {closeAfterUpgrade: true}})
	defer env.Close()
	_, byName := runQA(t, env)
	expectFailure(t, byName["chat_libsignal"], "provisioning_address", "eof_error")
	expectFailure(t, byName["chat_android"], "provisioning_address", "eof_error")
}

func TestQANoExtendedConnect(t *testing.T) {
	env := newQAEnv(t, map[string]qaBehavior{qaGRPCChatAddr: {noConnectProtocol: true}})
	defer env.Close()
	_, byName := runQA(t, env)
	expectFailure(t, byName["chat_libsignal"], "websocket_upgrade", FailureExtendedConnectNotSupported)
	expectOK(t, byName, "chat_android")
}

func TestQASignalReportsOutage(t *testing.T) {
	env := newQAEnv(t, nil)
	defer env.Close()
	env.AddRecordToAllResolvers("uptime.signal.org", "", "127.0.0.2")
	tk, _ := runQA(t, env)
	if tk.SignalServiceStatus != "outage" {
		t.Fatalf("service status %q", tk.SignalServiceStatus)
	}
}

func TestQAUpgradeRefused(t *testing.T) {
	env := newQAEnv(t, map[string]qaBehavior{qaGRPCChatAddr: {refuseUpgrade: true}, qaChatAddr: {refuseUpgrade: true}})
	defer env.Close()
	_, byName := runQA(t, env)
	for _, name := range []string{"chat_libsignal", "chat_android"} {
		expectFailure(t, byName[name], "websocket_upgrade", FailureWebSocketUpgradeRefused)
		if byName[name].Status != 403 {
			t.Errorf("%s: status %d", name, byName[name].Status)
		}
	}
	expectOK(t, byName, "cdsi", "storage")
}

func TestQAUserAgentIdentifiesOONIByDefault(t *testing.T) {
	env := newQAEnv(t, nil)
	defer env.Close()
	_, byName := runQA(t, env)
	seen := qaSeenUserAgents.seen()
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
	_, byName := runQAWith(t, env, Config{AppUserAgent: true})
	seen := qaSeenUserAgents.seen()
	for ua, hosts := range seen {
		if strings.Contains(ua, "ooniprobe") {
			t.Errorf("AppUserAgent still identified as OONI to %v: %q", hosts, ua)
		}
	}
	for name, want := range map[string]string{
		"chat_libsignal":   "Signal-Android/8.29.3 Android/34 libsignal/0.105.0",
		"chat_android":     "Signal-Android/8.29.3 Android/34",
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
