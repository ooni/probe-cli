package signalclient

//
// Integration tests in a simulated network (netemx): fake Signal front ends, DNS, and DPI
// rules for each way the experiment should detect interference.
//

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
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
	refuseUpgrade     bool // answer the provisioning upgrade with 403, as a CDN's geoblock would
	enclaveAccepts    bool // accept enclave upgrades without credentials (101)
	enclave401        bool // answer enclave upgrades 401, as for a real enclave id without credentials
	deflate           bool // accept permessage-deflate (HTTP/1.1) and compress every message
	resetAfterAddress bool // HTTP/2: RST_STREAM right after sending the address
	killAfterAddress  bool // close the TCP connection right after sending the address
	truncateBody      bool // GET responses: promise 1000 bytes, send 5, close the connection
	badAccept         bool // answer the HTTP/1.1 provisioning upgrade 101 with a wrong Sec-WebSocket-Accept
}

// qaGeoblockPage is the body of the fake refusal: a CDN-style page that echoes the client's address.
const qaGeoblockPage = "<html><body>Access denied: your country (203.0.113.7) is not allowed</body></html>"

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
	return fakeServer{sendAddress: !b.noAddress, closeAtStart: b.closeAfterUpgrade, noTimestamp: b.noTimestamp,
		compress: b.deflate, stopAfterAddress: b.resetAfterAddress || b.killAfterAddress}
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
	s.serveH1Request(conn, br, req)
}

// serveH1Request answers one HTTP/1.1 request already read from br.
func (s *qaSignalServer) serveH1Request(conn net.Conn, br *bufio.Reader, req *http.Request) {
	s.factory.uas.add(req.UserAgent(), req.Host)
	upgrade := strings.EqualFold(req.Header.Get("Upgrade"), "websocket")
	switch {
	case upgrade && s.factory.behavior.refuseUpgrade: // a CDN's geoblock page, which echoes the client's IP
		fmt.Fprintf(conn, "HTTP/1.1 403 Forbidden\r\nServer: cloudflare\r\nContent-Type: text/html\r\nContent-Length: %d\r\n"+
			"Connection: close\r\n\r\n%s", len(qaGeoblockPage), qaGeoblockPage)
	case upgrade && req.URL.Path == "/v1/websocket/provisioning/":
		accept := websocketAccept(req.Header.Get("Sec-WebSocket-Key"))
		if s.factory.behavior.badAccept {
			accept = websocketAccept("not the client's key")
		}
		ext := ""
		if s.factory.behavior.deflate {
			ext = "Sec-WebSocket-Extensions: permessage-deflate\r\n"
		}
		fmt.Fprintf(conn, "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n"+
			"Sec-WebSocket-Accept: %s\r\n%s%s\r\n", accept, s.timestampHeader(), ext)
		s.provisioning().serve(s.factory.t, br, conn)
	case upgrade && s.factory.behavior.enclave401:
		fmt.Fprintf(conn, "HTTP/1.1 401 Unauthorized\r\nContent-Length: 0\r\nConnection: close\r\n\r\n")
	case upgrade && s.factory.behavior.enclaveAccepts:
		fmt.Fprintf(conn, "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n"+
			"Sec-WebSocket-Accept: %s\r\n\r\n", websocketAccept(req.Header.Get("Sec-WebSocket-Key")))
	case upgrade: // an enclave path without credentials
		fmt.Fprintf(conn, "HTTP/1.1 404 Not Found\r\nContent-Length: 0\r\nConnection: close\r\n\r\n")
	case s.factory.behavior.truncateBody:
		fmt.Fprintf(conn, "HTTP/1.1 200 OK\r\nContent-Length: 1000\r\n\r\nshort")
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
					// close pr when done, or the main loop's pw.Write blocks on the stream's next DATA
					defer pr.Close()
					s.provisioning().serve(s.factory.t, pr, w)
					switch {
					case s.factory.behavior.resetAfterAddress:
						locked(func() error { return fr.WriteRSTStream(f.StreamID, http2.ErrCodeCancel) })
					case s.factory.behavior.killAfterAddress:
						conn.Close()
					default:
						locked(func() error { return fr.WriteData(f.StreamID, true, nil) })
					}
				}()
				continue
			}
			if s.factory.behavior.truncateBody {
				respond(f.StreamID, 200, false)
				locked(func() error { return fr.WriteData(f.StreamID, false, []byte("short")) })
				conn.Close()
				return
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

// Addresses of the simulated fronts.
const (
	qaFastlyFrontAddr = "151.101.1.194" // github.githubassets.com, pinterest.com, www.redditstatic.com
	qaGoogleFrontAddr = "142.250.180.4" // www.google.com and the other Google fronts
)

// qaUpstreams maps the hosts a reflector may be asked for (X-Signal-Host) to the fake servers.
var qaUpstreams = map[string]string{"grpc.chat.signal.org": qaGRPCChatAddr, "cdsi.signal.org": qaEnclaveAddr}

// qaStaticAddrs are the static addresses that get a fake server (the second address of each chat
// host, which the controls' scenarios check). Every simulated host costs memory and goroutines
// that netem does not release when the env closes (gvisorStack.Close does not stop the gVisor
// stack), so the other static addresses stay unreachable: controls to them fail fast (TestMain).
var qaStaticAddrs = map[string][]string{
	"76.223.66.180": {"grpc.chat.signal.org"},
	"76.223.92.165": {"chat.signal.org"},
}

// qaEnv is a network where every Signal target resolves to a fake front end, the qaStaticAddrs
// serve their hosts, 1.1.1.1 answers DNS over UDP and HTTPS (from the non-ISP resolvers'
// records), and the circumvention fronts relay to the fake servers.
func newQAEnv(t *testing.T, behaviors map[string]qaBehavior) *netemx.QAEnv {
	qaSeenUserAgents = &qaUserAgents{byUA: map[string][]string{}}
	front := func(addr string, names ...string) netemx.QAEnvOption {
		return netemx.QAEnvOptionNetStack(addr, &qaSignalServerFactory{t: t, names: names, behavior: behaviors[addr], uas: qaSeenUserAgents})
	}
	options := []netemx.QAEnvOption{
		front(qaGRPCChatAddr, "grpc.chat.signal.org"),
		front(qaChatAddr, "chat.signal.org"),
		front(qaEnclaveAddr, "cdsi.signal.org", "svr2.signal.org", "svrb.signal.org"),
		front(qaCDNAddr, "storage.signal.org", "cdn.signal.org", "cdn2.signal.org", "cdn3.signal.org",
			"updates2.signal.org", "sfu.voip.signal.org"),
		netemx.QAEnvOptionNetStack(netemx.AddressCloudflareDNSCom1111,
			&netemx.HTTPSecureServerFactory{Factory: &netemx.DNSOverHTTPSHandlerFactory{}, Ports: []int{443},
				ServerNameMain: netemx.AddressCloudflareDNSCom1111},
			&netemx.DNSOverUDPServerFactory{}),
		netemx.QAEnvOptionNetStack(qaFastlyFrontAddr, &qaFrontFactory{t: t, behavior: behaviors[qaFastlyFrontAddr],
			names: fastlyFronts}),
		netemx.QAEnvOptionNetStack(qaGoogleFrontAddr, &qaFrontFactory{t: t, behavior: behaviors[qaGoogleFrontAddr],
			names: append(append([]string(nil), googleFronts...), "www.google.com.pk", "www.google.com.eg")}),
		netemx.QAEnvOptionLogger(log.Log),
	}
	for addr, hosts := range qaStaticAddrs {
		options = append(options, front(addr, hosts...))
	}
	env := netemx.MustNewQAEnv(options...)
	for _, name := range fastlyFronts {
		env.AddRecordToAllResolvers(name, "", qaFastlyFrontAddr)
	}
	for _, name := range append(append([]string(nil), googleFronts...), "www.google.com.pk", "www.google.com.eg") {
		env.AddRecordToAllResolvers(name, "", qaGoogleFrontAddr)
	}
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
	return runQATargets(t, env, config, DefaultTargets)
}

// runQATargets is runQAWith with a target list.
func runQATargets(t *testing.T, env *netemx.QAEnv, config Config, targets []Target) (*TestKeys, map[string]*TargetResult) {
	t.Helper()
	var tk *TestKeys
	env.Do(func() {
		m := NewExperimentMeasurer(config).(*Measurer)
		m.targets = targets
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
	tk, byName := runQA(t, env)
	expectFailure(t, byName["chat_libsignal"], "tls_handshake", "connection_reset")
	expectOK(t, byName, "chat_android", "cdsi", "storage")
	if !tk.MeasurementSummaryKeys().Anomaly() {
		t.Error("no anomaly with chat_libsignal reset")
	}
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
	if tk.MeasurementSummaryKeys().Anomaly() {
		t.Error("anomaly while Signal reports an outage")
	}
}

func TestQAUpgradeRefused(t *testing.T) {
	env := newQAEnv(t, map[string]qaBehavior{qaGRPCChatAddr: {refuseUpgrade: true}, qaChatAddr: {refuseUpgrade: true}})
	defer env.Close()
	tk, byName := runQA(t, env)
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
	env := newQAEnv(t, map[string]qaBehavior{qaChatAddr: {badAccept: true}})
	defer env.Close()
	_, byName := runQA(t, env)
	expectFailure(t, byName["chat_android"], "websocket_upgrade", FailureWebSocketUpgradeInvalid)
	if byName["chat_android"].Status != 101 {
		t.Errorf("status %d", byName["chat_android"].Status)
	}
	expectOK(t, byName, "chat_libsignal")
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
	env := newQAEnv(t, map[string]qaBehavior{qaEnclaveAddr: {enclaveAccepts: true}})
	defer env.Close()
	_, byName := runQA(t, env)
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
	_, byName := runQATargets(t, env, Config{}, []Target{{"broken", "storage.signal.org", broken, MethodHTTPSGet, "/"}})
	r := byName["broken"]
	if r.FailedOperation != "tls_handshake" || r.Failure == nil {
		t.Fatalf("expected a tls_handshake failure, got op %q failure %q", r.FailedOperation, deref(r.Failure))
	}
}

func TestQAEnclave401IsReached(t *testing.T) {
	env := newQAEnv(t, map[string]qaBehavior{qaEnclaveAddr: {enclave401: true}})
	defer env.Close()
	_, byName := runQA(t, env)
	for _, name := range []string{"cdsi", "svr2", "svrb"} {
		if r := byName[name]; r.Failure != nil || r.Status != 401 {
			t.Errorf("%s: status %d failure %q", name, r.Status, deref(r.Failure))
		}
	}
}

func TestQAPermessageDeflateNegotiated(t *testing.T) {
	env := newQAEnv(t, map[string]qaBehavior{qaChatAddr: {deflate: true}})
	defer env.Close()
	_, byName := runQA(t, env)
	r := byName["chat_android"]
	if r.Failure != nil || r.WSExtensions != "permessage-deflate" || r.Provisioning == nil || !r.Provisioning.AddressReceived ||
		len(r.Provisioning.Keepalives) != 3 {
		t.Fatalf("deflate flow: %+v failure=%q", r, deref(r.Failure))
	}
}

func TestQAStreamResetAfterAddress(t *testing.T) {
	env := newQAEnv(t, map[string]qaBehavior{qaGRPCChatAddr: {resetAfterAddress: true}})
	defer env.Close()
	_, byName := runQA(t, env)
	r := byName["chat_libsignal"]
	if r.Failure == nil || r.Provisioning == nil || !r.Provisioning.AddressReceived {
		t.Fatalf("expected a failure after the address, got %+v", r)
	}
	if deref(r.Failure) != FailureHTTP2StreamReset {
		t.Errorf("stream reset recorded as %q at %q", deref(r.Failure), r.FailedOperation)
	}
}

func TestQAConnectionKilledAfterAddress(t *testing.T) {
	env := newQAEnv(t, map[string]qaBehavior{qaGRPCChatAddr: {killAfterAddress: true}, qaChatAddr: {killAfterAddress: true}})
	defer env.Close()
	_, byName := runQA(t, env)
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
	env := newQAEnv(t, map[string]qaBehavior{qaCDNAddr: {truncateBody: true}})
	defer env.Close()
	_, byName := runQA(t, env)
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

// A TLS blackhole costs the connect budget, not the whole provisioning budget.
func TestQAConnectBudgetIsEnforced(t *testing.T) {
	defer func(d, c time.Duration) { connectBudget, controlHandshakeTimeout = d, c }(connectBudget, controlHandshakeTimeout)
	connectBudget, controlHandshakeTimeout = 2*time.Second, time.Second
	env := newQAEnv(t, nil)
	defer env.Close()
	env.DPIEngine().AddRule(&netem.DPIDropTrafficForTLSSNI{Logger: log.Log, SNI: "chat.signal.org"})
	_, byName := runQA(t, env)
	r := byName["chat_android"]
	expectFailure(t, r, "tls_handshake", "generic_timeout_error")
	if took := r.T - r.T0; took > connectBudget.Seconds()+1 {
		t.Errorf("chat_android took %.1f s with a %v connect budget", took, connectBudget)
	}
	expectOK(t, byName, "chat_libsignal")
}

func TestQATLSRecordsAreTaggedWithTheTarget(t *testing.T) {
	env := newQAEnv(t, nil)
	defer env.Close()
	tk, _ := runQA(t, env)
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
