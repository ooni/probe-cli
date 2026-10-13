package signalclient

//
// The simulated network of the QA tests: fake Signal front ends and CDN fronts in a netemx
// environment, and the ways they can misbehave.
//

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"slices"
	"strings"
	"sync"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/netem"
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

// qaBlockpageDomain is the name on a middlebox's blockpage certificate.
const qaBlockpageDomain = "blockpage.example"

func qaSelfSignedCertificate(t *testing.T, name string) tls.Certificate {
	certPEM, keyPEM := mustSelfSigned(t, name)
	return runtimex.Try1(tls.X509KeyPair(certPEM, keyPEM))
}

// qaInterceptorAddr is a host that exists only when faults name it: where a forged DNS answer
// sends a target, to a server that is not Signal.
const qaInterceptorAddr = "203.0.113.10"

// Addresses of the simulated fronts.
const (
	qaFastlyFrontAddr = "151.101.1.194" // github.githubassets.com, pinterest.com, www.redditstatic.com
	qaGoogleFrontAddr = "142.250.180.4" // www.google.com and the other Google fronts
)

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

// qaEnv is a simulated network, and the User-Agents its fake servers saw.
type qaEnv struct {
	*netemx.QAEnv
	uas *qaUserAgents
}

// qaGoogleFronts are the names the simulated Google front serves.
var qaGoogleFronts = append(slices.Clone(googleFronts), "www.google.com.pk", "www.google.com.eg")

// newQAEnv returns a network where every Signal target resolves to a fake front end, the
// qaStaticAddrs serve their hosts, 1.1.1.1 answers DNS over UDP and HTTPS (from the non-ISP
// resolvers' records), and the circumvention fronts relay to the fake servers. faults says how
// the server at each address misbehaves.
func newQAEnv(t *testing.T, faults map[string]fault) *qaEnv {
	uas := &qaUserAgents{byUA: map[string][]string{}}
	front := func(addr string, names ...string) netemx.QAEnvOption {
		return netemx.QAEnvOptionNetStack(addr, &qaSignalServerFactory{t: t, names: names, fault: faults[addr], uas: uas})
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
		netemx.QAEnvOptionNetStack(qaFastlyFrontAddr, &qaFrontFactory{t: t, fault: faults[qaFastlyFrontAddr],
			names: fastlyFronts, uas: uas}),
		netemx.QAEnvOptionNetStack(qaGoogleFrontAddr, &qaFrontFactory{t: t, fault: faults[qaGoogleFrontAddr],
			names: qaGoogleFronts, uas: uas}),
		netemx.QAEnvOptionLogger(log.Log),
	}
	for addr, hosts := range qaStaticAddrs {
		options = append(options, front(addr, hosts...))
	}
	if _, ok := faults[qaInterceptorAddr]; ok { // only when a scenario sends traffic there
		options = append(options, front(qaInterceptorAddr, "chat.signal.org"))
	}
	env := netemx.MustNewQAEnv(options...)
	for _, name := range fastlyFronts {
		env.AddRecordToAllResolvers(name, "", qaFastlyFrontAddr)
	}
	for _, name := range qaGoogleFronts {
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
	return &qaEnv{QAEnv: env, uas: uas}
}

// qaUpstreams maps the hosts a reflector may be asked for (X-Signal-Host) to the fake servers.
var qaUpstreams = map[string]string{"grpc.chat.signal.org": qaGRPCChatAddr, "cdsi.signal.org": qaEnclaveAddr}

// qaStaticAddrs are the static addresses that get a fake server (the second address of each chat
// host, which the controls' scenarios check). Every simulated host costs memory and goroutines
// that netem does not release when the env closes (gvisorStack.Close does not stop the gVisor
// stack), so the other static addresses stay unreachable, and runQA shortens the controls'
// handshakes to them.
var qaStaticAddrs = map[string][]string{
	"76.223.66.180": {"grpc.chat.signal.org"},
	"76.223.92.165": {"chat.signal.org"},
}

// qaSignalServerFactory is a netemx server factory for a fake Signal front end.
type qaSignalServerFactory struct {
	t     *testing.T
	names []string
	fault fault
	uas   *qaUserAgents
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
	switch s.factory.fault {
	case faultImpostorCertificate:
		config.Certificates = []tls.Certificate{qaSelfSignedCertificate(s.factory.t, s.factory.names[0])}
	case faultBlockpageCertificate:
		config.Certificates = []tls.Certificate{qaSelfSignedCertificate(s.factory.t, qaBlockpageDomain)}
	}
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
	return fakeServer{fault: s.factory.fault, compress: s.factory.fault == deflate}
}

func (s *qaSignalServer) timestampHeader() string {
	if s.factory.fault == faultNoTimestamp {
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
	case upgrade && s.factory.fault == faultRefuseUpgrade: // a CDN's geoblock page, which echoes the client's IP
		fmt.Fprintf(conn, "HTTP/1.1 403 Forbidden\r\nServer: cloudflare\r\nContent-Type: text/html\r\nContent-Length: %d\r\n"+
			"Connection: close\r\n\r\n%s", len(qaGeoblockPage), qaGeoblockPage)
	case upgrade && req.URL.Path == "/v1/websocket/provisioning/":
		accept := websocketAccept(req.Header.Get("Sec-WebSocket-Key"))
		if s.factory.fault == faultBadAccept {
			accept = websocketAccept("not the client's key")
		}
		ext := ""
		if s.factory.fault == deflate {
			ext = "Sec-WebSocket-Extensions: permessage-deflate\r\n"
		}
		fmt.Fprintf(conn, "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n"+
			"Sec-WebSocket-Accept: %s\r\n%s%s\r\n", accept, s.timestampHeader(), ext)
		s.provisioning().serve(s.factory.t, br, conn)
	case upgrade && s.factory.fault == faultEnclave401:
		fmt.Fprintf(conn, "HTTP/1.1 401 Unauthorized\r\nContent-Length: 0\r\nConnection: close\r\n\r\n")
	case upgrade && s.factory.fault == faultEnclaveAccepts:
		fmt.Fprintf(conn, "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\n"+
			"Sec-WebSocket-Accept: %s\r\n\r\n", websocketAccept(req.Header.Get("Sec-WebSocket-Key")))
	case upgrade: // an enclave path without credentials
		fmt.Fprintf(conn, "HTTP/1.1 404 Not Found\r\nContent-Length: 0\r\nConnection: close\r\n\r\n")
	case s.factory.fault == faultResetRequest:
		return // the caller closes the connection
	case s.factory.fault == faultTruncateBody:
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
	if s.factory.fault != faultNoConnectProtocol {
		settings = append(settings, http2.Setting{ID: http2.SettingEnableConnectProtocol, Val: 1})
	}
	locked(func() error { return fr.WriteSettings(settings...) })
	respond := func(stream uint32, status int, endStream bool) error {
		var buf bytes.Buffer
		enc := hpack.NewEncoder(&buf)
		enc.WriteField(hpack.HeaderField{Name: ":status", Value: fmt.Sprint(status)})
		if s.factory.fault != faultNoTimestamp {
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
				if s.factory.fault == faultRefuseUpgrade {
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
					switch s.factory.fault {
					case faultResetAfterAddress:
						locked(func() error { return fr.WriteRSTStream(f.StreamID, http2.ErrCodeCancel) })
					case faultKillAfterAddress:
						conn.Close()
					default:
						locked(func() error { return fr.WriteData(f.StreamID, true, nil) })
					}
				}()
				continue
			}
			if s.factory.fault == faultResetRequest {
				conn.Close()
				return
			}
			if s.factory.fault == faultTruncateBody {
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

// qaFrontFactory is a CDN front: libsignal's reflector tunnel (/tls-tunnel with X-Signal-Host)
// and Signal-Android's fronted requests (Host header and path prefix), relayed to the fake
// Signal servers.
type qaFrontFactory struct {
	t     *testing.T
	names []string
	fault fault
	uas   *qaUserAgents
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

func (s *qaFront) serve(conn net.Conn) {
	defer conn.Close()
	br := bufio.NewReader(conn)
	req, err := http.ReadRequest(br)
	if err != nil {
		return
	}
	if strings.EqualFold(req.Header.Get("Upgrade"), "websocket") && req.URL.Path == reflectorPath && reflectorHosts[req.Host] {
		s.factory.uas.add(req.UserAgent(), "tunnel:"+req.Host)
		upstream, ok := qaUpstreams[req.Header.Get("X-Signal-Host")]
		if s.factory.fault == faultRefuseUpgrade || !ok {
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
	signal := &qaSignalServer{factory: &qaSignalServerFactory{t: s.factory.t, names: []string{host}, fault: s.factory.fault,
		uas: s.factory.uas}}
	signal.serveH1Request(conn, br, req)
}

// frontedServices maps a front's Host header (and path prefix) to the Signal host it reaches.
var frontedServices = map[string]string{
	"chat-signal.global.ssl.fastly.net":         "chat.signal.org",
	"storage.signal.org.global.prod.fastly.net": "storage.signal.org",
	googleReflectorHost + " /service":           "chat.signal.org",
	googleReflectorHost + " /storage":           "storage.signal.org",
}

var reflectorHosts = map[string]bool{"reflector-signal.global.ssl.fastly.net": true, googleReflectorHost: true}

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
				if _, err := conn.Write(serverFrame(true, false, opBinary, buf[:n])); err != nil {
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
