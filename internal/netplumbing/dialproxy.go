package netplumbing

import (
	"bufio"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"

	"golang.org/x/net/proxy"
)

// dialProxy is the entry point for dialing using a proxy.
func (txp *Transport) dialProxy(
	ctx context.Context, network string, address string,
	proxyURL *url.URL) (net.Conn, error) {
	log := txp.logger(ctx)
	log.Debugf("proxy: dialing with: %s", proxyURL)
	switch scheme := proxyURL.Scheme; scheme {
	case "socks5", "socks5h":
		return txp.dialProxySOCKS5(ctx, network, address, proxyURL)
	case "http":
		return txp.dialProxyHTTP(ctx, network, address, proxyURL)
	default:
		return nil, fmt.Errorf("%w: %s", ErrProxyNotImplemented, scheme)
	}
}

// ErrProxyNotImplemented indicates that we don't support connecting via proxy.
var ErrProxyNotImplemented = errors.New("netplumbing: proxy not implemented")

// dialProxySOCKS5 dials using a socks5 proxy.
func (txp *Transport) dialProxySOCKS5(
	ctx context.Context, network string, address string,
	proxyURL *url.URL) (net.Conn, error) {
	var auth *proxy.Auth
	if user := proxyURL.User; user != nil {
		password, _ := user.Password()
		auth = &proxy.Auth{
			User:     user.Username(),
			Password: password,
		}
	}
	// the code at proxy/socks5.go never fails; see https://git.io/JfJ4g
	socks5, _ := proxy.SOCKS5(network, proxyURL.Host, auth, &proxyAdapter{txp})
	contextDialer := socks5.(proxy.ContextDialer)
	return contextDialer.DialContext(ctx, network, address)
}

// proxyAdapter uses txp.connect as a child dial function
type proxyAdapter struct {
	txp *Transport
}

// DialContext implements proxy.ContextDialer.DialContext.
func (pc *proxyAdapter) DialContext(
	ctx context.Context, network, address string) (net.Conn, error) {
	return pc.txp.dialContextEmitLogs(ctx, network, address)
}

// Dial implements proxy.Dialer.Dial.
func (pc *proxyAdapter) Dial(network, address string) (net.Conn, error) {
	panic("netplumbing: this function should not be called")
}

// dialProxyHTTP dials using an HTTP proxy.
func (txp *Transport) dialProxyHTTP(
	ctx context.Context, network string, address string,
	proxyURL *url.URL) (net.Conn, error) {
	req, err := http.NewRequestWithContext(ctx, "CONNECT", address, nil)
	if err != nil {
		return nil, err
	}
	req.URL = &url.URL{Host: address} // fixup the real URL
	req.Header.Set("Host", address)
	if auth := proxyAuth(proxyURL); auth != "" {
		req.Header.Set("Proxy-Authorization", auth)
	}
	// TODO(bassosimone): allow a TLS proxy
	txp.dialProxyHTTPLogRequest(ctx, req)
	conn, err := txp.dialContextEmitLogs(ctx, "tcp", proxyURL.Host)
	if err != nil {
		return nil, err
	}
	pc := &proxyConnHTTP{r: bufio.NewReader(conn), Conn: conn}
	if err := txp.dialProxyHTTPWithConn(ctx, pc, req); err != nil {
		pc.Close()
		return nil, err
	}
	return pc, nil
}

// dialProxyHTTPLogRequest logs the outgoing HTTP request.
func (txp *Transport) dialProxyHTTPLogRequest(ctx context.Context, req *http.Request) {
	log := txp.logger(ctx)
	log.Debugf("> %s %s", req.Method, req.URL.Host) // fixup emitted URL
	for key, values := range req.Header {
		for _, value := range values {
			log.Debugf("> %s: %s", key, value)
		}
	}
	log.Debug(">")
}

// dialProxyHTTPWithConn finishes dialing with the given conn
func (txp *Transport) dialProxyHTTPWithConn(
	ctx context.Context, pc *proxyConnHTTP, req *http.Request) error {
	if err := pc.writeConnect(ctx, req); err != nil {
		return err
	}
	resp, err := pc.readResponse(ctx, req)
	if err != nil {
		return err
	}
	txp.dialProxyHTTPLogResponse(ctx, resp)
	if resp.StatusCode != 200 {
		return fmt.Errorf("proxy: request failed: code=%d", resp.StatusCode)
	}
	return nil
}

// dialProxyHTTPLogResponse logs the incoming HTTP response.
func (txp *Transport) dialProxyHTTPLogResponse(ctx context.Context, resp *http.Response) {
	log := txp.logger(ctx)
	log.Debugf("< %d", resp.StatusCode)
	for key, values := range resp.Header {
		for _, value := range values {
			log.Debugf("< %s: %s", key, value)
		}
	}
	log.Debug("<")
}

// proxyConnHTTP is a connection to an HTTP proxy.
type proxyConnHTTP struct {
	// r is used to read the proxy response
	r *bufio.Reader

	// Conn is the embedded conn
	net.Conn
}

// Read reads from the underlying connection.
func (c *proxyConnHTTP) Read(b []byte) (int, error) {
	return c.r.Read(b)
}

// writeConnect writes the CONNECT request.
func (c *proxyConnHTTP) writeConnect(ctx context.Context, req *http.Request) error {
	errch := make(chan error, 1)
	go func() { errch <- req.Write(c.Conn) }()
	select {
	case err := <-errch:
		return err
	case <-ctx.Done():
		return ctx.Err()
	}
}

// readResponse reads the response to the CONNECT request.
func (c *proxyConnHTTP) readResponse(
	ctx context.Context, req *http.Request) (*http.Response, error) {
	respch, errch := make(chan *http.Response, 1), make(chan error, 1)
	go func() {
		resp, err := http.ReadResponse(c.r, req)
		if err != nil {
			errch <- err
			return
		}
		respch <- resp
	}()
	select {
	case resp := <-respch:
		return resp, nil
	case err := <-errch:
		return nil, err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// From src/net/http/client.go in the standard library.
//
// License: 3-clause BSD + patent grant.
func proxyAuth(proxyURL *url.URL) string {
	if u := proxyURL.User; u != nil {
		username := u.Username()
		password, _ := u.Password()
		return "Basic " + proxyBasicAuth(username, password)
	}
	return ""
}

// From src/net/http/client.go in the standard library.
//
// License: 3-clause BSD + patent grant.
//
// See 2 (end of page 4) https://www.ietf.org/rfc/rfc2617.txt
// "To receive authorization, the client sends the userid and password,
// separated by a single colon (":") character, within a base64
// encoded string in the credentials."
// It is not meant to be urlencoded.
func proxyBasicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}
