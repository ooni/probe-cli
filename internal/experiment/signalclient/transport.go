package signalclient

//
// TCP + TLS with a reproduced Signal ClientHello, then the client's application protocol:
// an HTTP/1.1 WebSocket upgrade, an RFC 8441 extended CONNECT over HTTP/2, or an HTTPS GET.
//

import (
	"bufio"
	"context"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	utls "github.com/refraction-networking/utls"
	"golang.org/x/net/http2"
)

// libsignal 0.105.0's HTTP/2 client settings, from a decrypted capture: SETTINGS ENABLE_PUSH 0,
// INITIAL_WINDOW_SIZE 2097152, MAX_FRAME_SIZE 16384, MAX_HEADER_LIST_SIZE 16384, then a
// connection WINDOW_UPDATE of 5177345. x/net sends its settings in the same order.
const (
	libsignalStreamWindow     = 2097152
	libsignalConnWindowUpdate = 5177345
	libsignalMaxFrameSize     = 16384
	libsignalMaxHeaderList    = 16384
)

// tlsResult is what dialTLS observed.
type tlsResult struct {
	conn      *utls.UConn
	alpn      string
	leafKey   string
	handshake *model.ArchivalTLSOrQUICHandshakeResult
}

// dialTLS connects to address and handshakes with the given hello, verifying the chain against
// roots for sni. Events go to trace.
func dialTLS(ctx context.Context, trace *measurexlite.Trace, logger model.Logger, zero time.Time,
	address, sni string, hello Hello, roots *x509.CertPool) (*tlsResult, error) {
	dialer := trace.NewDialerWithoutResolver(logger)
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, err // already wrapped by netxlite (operation "connect")
	}
	spec, err := hello.Spec()
	if err != nil {
		conn.Close()
		return nil, netxlite.NewTopLevelGenericErrWrapper(err)
	}
	config := &utls.Config{ServerName: sni, RootCAs: roots}
	uconn := utls.UClient(conn, config, utls.HelloCustom)
	if err := uconn.ApplyPreset(spec); err != nil {
		conn.Close()
		return nil, netxlite.NewTopLevelGenericErrWrapper(fmt.Errorf("signal_client: %s: %w", hello.Name, err))
	}
	started := time.Since(zero)
	err = uconn.HandshakeContext(ctx)
	finished := time.Since(zero)
	if err != nil {
		err = netxlite.NewErrWrapper(netxlite.ClassifyTLSHandshakeError, netxlite.TLSHandshakeOperation, err)
	}
	state := stdlibState(uconn.ConnectionState())
	hs := measurexlite.NewArchivalTLSOrQUICHandshakeResult(trace.Index(), started, "tcp", address,
		&tls.Config{ServerName: sni, NextProtos: hello.ALPN()}, state, err, finished, hello.Name)
	if err != nil {
		conn.Close()
		return &tlsResult{handshake: hs}, err
	}
	return &tlsResult{conn: uconn, alpn: state.NegotiatedProtocol, leafKey: leafKeyType(state), handshake: hs}, nil
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

// upgradeResult is the server's answer to a WebSocket upgrade or extended CONNECT.
type upgradeResult struct {
	status          int
	signalTimestamp bool
	extensions      string // Sec-WebSocket-Extensions the server accepted
	ws              *wsConn
	closer          io.Closer
}

// errUpgradeRefused means the server answered the upgrade with something other than success.
var errUpgradeRefused = errors.New("websocket_upgrade_refused")

// websocketH1 performs Android OkHttp's HTTP/1.1 WebSocket upgrade on an established TLS conn.
func websocketH1(ctx context.Context, conn net.Conn, host, path, userAgent string) (*upgradeResult, error) {
	var key [16]byte
	if _, err := rand.Read(key[:]); err != nil {
		return nil, err
	}
	// OkHttp 5.3.2's order: RealWebSocket.connect's headers, then the app's User-Agent
	// interceptor (Signal's StandardUserAgentInterceptor), then BridgeInterceptor's Host and
	// Accept-Encoding. All of it is inside TLS; only the encrypted record sizes are visible.
	req := "GET " + path + " HTTP/1.1\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Key: " + base64.StdEncoding.EncodeToString(key[:]) + "\r\n" +
		"Sec-WebSocket-Version: 13\r\n" +
		"Sec-WebSocket-Extensions: permessage-deflate\r\n" +
		"User-Agent: " + userAgent + "\r\n" +
		"Host: " + host + "\r\n" +
		"Accept-Encoding: gzip\r\n\r\n"
	if deadline, ok := ctx.Deadline(); ok {
		conn.SetDeadline(deadline)
		defer conn.SetDeadline(time.Time{})
	}
	if _, err := io.WriteString(conn, req); err != nil {
		return nil, netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.WriteOperation, err)
	}
	br := bufio.NewReader(conn)
	resp, err := http.ReadResponse(br, nil)
	if err != nil {
		return nil, netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.ReadOperation, err)
	}
	out := &upgradeResult{status: resp.StatusCode, signalTimestamp: resp.Header.Get("X-Signal-Timestamp") != "",
		extensions: resp.Header.Get("Sec-WebSocket-Extensions")}
	if resp.StatusCode != http.StatusSwitchingProtocols {
		resp.Body.Close()
		return out, errUpgradeRefused
	}
	out.ws = newWSConn(br, conn)
	out.closer = conn
	return out, nil
}

// websocketH2 performs libsignal's RFC 8441 extended CONNECT on an established TLS conn.
func websocketH2(ctx context.Context, conn net.Conn, host, path, userAgent string) (*upgradeResult, error) {
	t1 := &http.Transport{HTTP2: &http.HTTP2Config{
		MaxReceiveBufferPerStream:     libsignalStreamWindow,
		MaxReceiveBufferPerConnection: libsignalConnWindowUpdate,
		MaxReadFrameSize:              libsignalMaxFrameSize,
	}}
	t2, err := http2.ConfigureTransports(t1)
	if err != nil {
		return nil, netxlite.NewTopLevelGenericErrWrapper(err)
	}
	t2.MaxHeaderListSize = libsignalMaxHeaderList
	cc, err := t2.NewClientConn(conn)
	if err != nil {
		return nil, netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.HTTPRoundTripOperation, err)
	}
	pr, pw := io.Pipe()
	req, err := http.NewRequestWithContext(ctx, http.MethodConnect, "https://"+host+path, pr)
	if err != nil {
		return nil, netxlite.NewTopLevelGenericErrWrapper(err)
	}
	req.Header[":protocol"] = []string{"websocket"}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Sec-WebSocket-Version", "13")
	resp, err := cc.RoundTrip(req)
	if err != nil {
		pw.Close()
		cc.Close()
		return nil, netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.HTTPRoundTripOperation, err)
	}
	out := &upgradeResult{status: resp.StatusCode, signalTimestamp: resp.Header.Get("X-Signal-Timestamp") != ""}
	if resp.StatusCode/100 != 2 { // libsignal accepts any 2xx
		resp.Body.Close()
		pw.Close()
		cc.Close()
		return out, errUpgradeRefused
	}
	out.ws = newWSConn(resp.Body, pw)
	out.closer = closerFunc(func() error { pw.Close(); resp.Body.Close(); return cc.Close() })
	return out, nil
}

type closerFunc func() error

func (f closerFunc) Close() error { return f() }

// httpsGet sends one GET over an established TLS conn, using HTTP/2 when it was negotiated,
// as OkHttp does, and returns the status and whether Signal's timestamp header was present.
func httpsGet(ctx context.Context, conn net.Conn, alpn, host, path, userAgent string) (int, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://"+host+path, nil)
	if err != nil {
		return 0, false, netxlite.NewTopLevelGenericErrWrapper(err)
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept-Encoding", "gzip")
	var resp *http.Response
	if alpn == "h2" {
		cc, err := (&http2.Transport{}).NewClientConn(conn)
		if err != nil {
			return 0, false, netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.HTTPRoundTripOperation, err)
		}
		defer cc.Close()
		resp, err = cc.RoundTrip(req)
		if err != nil {
			return 0, false, netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.HTTPRoundTripOperation, err)
		}
	} else {
		if deadline, ok := ctx.Deadline(); ok {
			conn.SetDeadline(deadline)
		}
		req.Header.Set("Connection", "close")
		if err := req.Write(conn); err != nil {
			return 0, false, netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.WriteOperation, err)
		}
		resp, err = http.ReadResponse(bufio.NewReader(conn), req)
		if err != nil {
			return 0, false, netxlite.NewErrWrapper(netxlite.ClassifyGenericError, netxlite.ReadOperation, err)
		}
	}
	io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<16))
	resp.Body.Close()
	return resp.StatusCode, resp.Header.Get("X-Signal-Timestamp") != "", nil
}
