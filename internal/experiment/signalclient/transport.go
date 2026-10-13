package signalclient

//
// TCP + TLS with a reproduced Signal ClientHello, then the client's application protocol:
// an HTTP/1.1 WebSocket upgrade, an RFC 8441 extended CONNECT over HTTP/2, or an HTTPS GET.
//

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha1" // #nosec G505 -- RFC 6455 defines Sec-WebSocket-Accept with SHA-1
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	utls "github.com/refraction-networking/utls"
	"golang.org/x/net/http/httpguts"
)

// tlsResult is what dialTLS observed.
type tlsResult struct {
	conn      net.Conn
	alpn      string
	leafKey   string
	handshake *model.ArchivalTLSOrQUICHandshakeResult
}

// dialTLS connects to address and handshakes with the given hello, verifying the chain against
// roots for sni (nil roots: the chain is recorded but not verified). Events go to trace, and
// recordTLS keeps them. On failure, failedOperation(err) says which step failed; the result,
// when not nil, holds the failed handshake's record.
func dialTLS(ctx context.Context, trace *measurexlite.Trace, logger model.Logger, address, sni string, hello Hello,
	roots *x509.CertPool) (*tlsResult, error) {
	conn, err := trace.NewDialerWithoutResolver(logger).DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, err // wrapped by netxlite with the connect operation
	}
	return handshakeTLS(ctx, trace, conn, address, sni, hello, roots)
}

// recordTLS moves the trace's TCP connects, and the handshake when there was one, to tk.
func (tk *TestKeys) recordTLS(trace *measurexlite.Trace, tr *tlsResult) {
	tk.TCPConnect = append(tk.TCPConnect, trace.TCPConnects()...)
	if tr != nil && tr.handshake != nil {
		tk.TLSHandshakes = append(tk.TLSHandshakes, tr.handshake)
	}
}

// handshakeTLS runs the TLS handshake of hello on conn (a TCP connection, or a tunnel whose
// address is recorded as given). It closes conn on failure. It does not use measurexlite's
// handshaker: netxlite's uTLS takes a ClientHelloID, not the ClientHelloSpec a captured hello
// needs (echcheck makes the same choice), so the handshake is recorded here.
func handshakeTLS(ctx context.Context, trace *measurexlite.Trace, conn net.Conn, address, sni string, hello Hello,
	roots *x509.CertPool) (*tlsResult, error) {
	zero := trace.ZeroTime()
	tlsFailure := func(err error) (*tlsResult, error) {
		_ = conn.Close()
		return &tlsResult{}, netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.TLSHandshakeOperation,
			fmt.Errorf("signal_client: %s: %w", hello.Name, err))
	}
	var (
		tconn interface {
			net.Conn
			HandshakeContext(context.Context) error
		}
		connState func() tls.ConnectionState
	)
	if hello.stdlibTLS12 {
		c := tls.Client(conn, &tls.Config{ServerName: sni, RootCAs: roots, InsecureSkipVerify: roots == nil, // #nosec G402 -- recorded, not trusted
			MinVersion: tls.VersionTLS12, MaxVersion: tls.VersionTLS12, NextProtos: hello.alpn})
		tconn, connState = c, c.ConnectionState
	} else {
		config := &utls.Config{ServerName: sni, RootCAs: roots, InsecureSkipVerify: roots == nil} // #nosec G402 -- recorded, not trusted
		uconn := utls.UClient(conn, config, utls.HelloCustom)
		if err := hello.apply(uconn); err != nil {
			return tlsFailure(err)
		}
		tconn, connState = uconn, func() tls.ConnectionState { return stdlibState(uconn.ConnectionState()) }
	}
	started := time.Since(zero)
	err := tconn.HandshakeContext(ctx)
	finished := time.Since(zero)
	if err != nil {
		err = netxlite.NewErrWrapper(netxlite.ClassifyTLSHandshakeError, netxlite.TLSHandshakeOperation, err)
	}
	state := connState()
	hs := measurexlite.NewArchivalTLSOrQUICHandshakeResult(trace.Index(), started, "tcp", address,
		&tls.Config{ServerName: sni, NextProtos: hello.ALPN(), InsecureSkipVerify: roots == nil}, // #nosec G402 -- archival only
		state, err, finished, trace.Tags()...)
	if err != nil {
		_ = conn.Close()
		return &tlsResult{handshake: hs}, err
	}
	return &tlsResult{conn: tconn, alpn: state.NegotiatedProtocol, leafKey: leafKeyType(state), handshake: hs}, nil
}

// stdlibState converts uTLS's connection state for the archival helpers.
func stdlibState(s utls.ConnectionState) tls.ConnectionState {
	return tls.ConnectionState{
		Version:            s.Version,
		HandshakeComplete:  s.HandshakeComplete,
		CipherSuite:        s.CipherSuite,
		NegotiatedProtocol: s.NegotiatedProtocol,
		ServerName:         s.ServerName,
		PeerCertificates:   s.PeerCertificates,
	}
}

func leafKeyType(s tls.ConnectionState) string {
	if len(s.PeerCertificates) == 0 {
		return ""
	}
	return s.PeerCertificates[0].PublicKeyAlgorithm.String()
}

// answer is what the server answered an HTTP request with.
type answer struct {
	status          int
	signalTimestamp bool            // X-Signal-Timestamp was present: Signal's server answered
	response        *ResponseSample // an https_get's response, or a refused upgrade's
}

func newAnswer(resp *http.Response) answer {
	return answer{status: resp.StatusCode, signalTimestamp: resp.Header.Get("X-Signal-Timestamp") != ""}
}

// upgradeResult is the server's answer to a WebSocket upgrade or extended CONNECT, and the
// WebSocket when it was accepted.
type upgradeResult struct {
	answer
	extensions string // Sec-WebSocket-Extensions the server accepted
	ws         *wsConn
	closer     io.Closer
}

// sampledHeaders are the response headers recorded: who answered (Signal, or which CDN) and why.
var sampledHeaders = []string{"Server", "Via", "Content-Type", "Content-Length", "Content-Encoding",
	"Retry-After", "X-Signal-Timestamp", "CF-Ray", "X-Cache", "X-Served-By", "X-Amz-Cf-Pop"}

// bodySampleSize is how much of a response body is recorded.
const bodySampleSize = 1024

// sampleResponse records resp's status, sampledHeaders and up to bodySampleSize bytes of its
// body, read within ctx. The error is the body's read error, if any.
func sampleResponse(ctx context.Context, resp *http.Response) (*ResponseSample, error) {
	s := &ResponseSample{Status: int64(resp.StatusCode), Headers: map[string]model.ArchivalScrubbedMaybeBinaryString{}}
	for _, h := range sampledHeaders {
		if v := resp.Header.Values(h); len(v) > 0 {
			s.Headers[h] = model.ArchivalScrubbedMaybeBinaryString(strings.Join(v, ", "))
		}
	}
	var buf bytes.Buffer
	_, err := netxlite.CopyContext(ctx, &buf, io.LimitReader(resp.Body, bodySampleSize+1))
	body := buf.Bytes()
	if len(body) > bodySampleSize {
		body, s.BodyIsTruncated = body[:bodySampleSize], true
	}
	s.Body = model.ArchivalScrubbedMaybeBinaryString(body)
	return s, err
}

// deflateAccepted is whether a Sec-WebSocket-Extensions answer accepts permessage-deflate.
func deflateAccepted(extensions string) bool {
	for _, ext := range strings.Split(extensions, ",") {
		name, _, _ := strings.Cut(ext, ";")
		if strings.EqualFold(strings.TrimSpace(name), "permessage-deflate") {
			return true
		}
	}
	return false
}

// websocketAccept is the Sec-WebSocket-Accept value a server must return for key.
func websocketAccept(key string) string {
	sum := sha1.Sum([]byte(key + "258EAFA5-E914-47DA-95CA-C5AB0DC85B11")) // #nosec G401 -- RFC 6455 section 4.2.2
	return base64.StdEncoding.EncodeToString(sum[:])
}

// newWebSocketKey returns a random Sec-WebSocket-Key.
func newWebSocketKey() (string, error) {
	var key [16]byte
	if _, err := rand.Read(key[:]); err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(key[:]), nil
}

// websocketH1 performs Android OkHttp's HTTP/1.1 WebSocket upgrade on an established TLS conn.
func websocketH1(ctx context.Context, conn net.Conn, host, path, userAgent string) (*upgradeResult, error) {
	wsKey, err := newWebSocketKey()
	if err != nil {
		return nil, err
	}
	// OkHttp 5.3.2's order: RealWebSocket.connect's headers, then the app's User-Agent
	// interceptor (Signal's StandardUserAgentInterceptor), then BridgeInterceptor's Host and
	// Accept-Encoding. All of it is inside TLS; only the encrypted record sizes are visible.
	req := "GET " + path + " HTTP/1.1\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Key: " + wsKey + "\r\n" +
		"Sec-WebSocket-Version: 13\r\n" +
		"Sec-WebSocket-Extensions: permessage-deflate\r\n" +
		"User-Agent: " + userAgent + "\r\n" +
		"Host: " + host + "\r\n" +
		"Accept-Encoding: gzip\r\n\r\n"
	return upgradeH1(ctx, conn, req, wsKey)
}

// upgradeH1 sends an HTTP/1.1 WebSocket upgrade request and checks the answer.
func upgradeH1(ctx context.Context, conn net.Conn, req, wsKey string) (*upgradeResult, error) {
	if deadline, ok := ctx.Deadline(); ok {
		_ = conn.SetDeadline(deadline)
		defer func() { _ = conn.SetDeadline(time.Time{}) }()
	}
	if _, err := io.WriteString(conn, req); err != nil {
		return nil, netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.WriteOperation, err)
	}
	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, nil)
	if err != nil {
		return nil, netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.ReadOperation, err)
	}
	out := &upgradeResult{answer: newAnswer(resp), extensions: resp.Header.Get("Sec-WebSocket-Extensions")}
	if resp.StatusCode != http.StatusSwitchingProtocols {
		out.response, _ = sampleResponse(ctx, resp) // bounded, within the operation's limit
		_ = conn.Close()                            // first: Body.Close would otherwise drain an endless body until the deadline
		_ = resp.Body.Close()
		return out, upgradeRefused()
	}
	// as OkHttp and tungstenite do, accept the upgrade only with the headers RFC 6455 requires
	if !httpguts.HeaderValuesContainsToken(resp.Header["Upgrade"], "websocket") ||
		!httpguts.HeaderValuesContainsToken(resp.Header["Connection"], "upgrade") ||
		resp.Header.Get("Sec-WebSocket-Accept") != websocketAccept(wsKey) {
		_ = conn.Close()
		return out, newFailure(FailureWebSocketUpgradeInvalid, netxlite.HTTPRoundTripOperation, errUpgradeInvalid)
	}
	out.ws = newWSConn(br, conn)
	out.ws.deflate = deflateAccepted(out.extensions)
	out.closer = conn
	return out, nil
}

type closerFunc func() error

func (f closerFunc) Close() error { return f() }

// httpsGet sends one GET over an established TLS conn, using HTTP/2 when it was negotiated,
// as OkHttp does, and returns the answer. It reads at most 64 KiB of the body; an error while
// reading it is returned with the answer. It consumes conn: the conn is closed before returning.
func httpsGet(ctx context.Context, conn net.Conn, alpn, host, path, userAgent string) (answer, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://"+host+path, nil)
	if err != nil {
		return answer{}, netxlite.NewTopLevelGenericErrWrapper(err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept-Encoding", "gzip")
	var resp *http.Response
	if alpn == "h2" {
		cc, err := newH2ClientConn(conn, nil)
		if err != nil {
			return answer{}, netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.HTTPRoundTripOperation, err)
		}
		defer func() { _ = cc.Close() }()
		resp, err = cc.RoundTrip(req)
		if err != nil {
			return answer{}, netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.HTTPRoundTripOperation, err)
		}
	} else {
		if deadline, ok := ctx.Deadline(); ok {
			_ = conn.SetDeadline(deadline)
		}
		req.Header.Set("Connection", "close")
		if err := req.Write(conn); err != nil {
			return answer{}, netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.WriteOperation, err)
		}
		resp, err = http.ReadResponse(bufio.NewReader(conn), req)
		if err != nil {
			return answer{}, netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.ReadOperation, err)
		}
	}
	a := newAnswer(resp)
	var bodyErr error
	a.response, bodyErr = sampleResponse(ctx, resp)
	if bodyErr == nil { // the rest of the body, up to 64 KiB, is read and discarded
		_, bodyErr = netxlite.CopyContext(ctx, io.Discard, io.LimitReader(resp.Body, 1<<16))
	}
	_ = conn.Close() // first: Body.Close would otherwise drain an endless body until the deadline
	_ = resp.Body.Close()
	if bodyErr != nil {
		return a, netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.ReadOperation, bodyErr)
	}
	return a, nil
}
