package webconnectivity

//
// SecureFlow
//
// Generated by `boilerplate' using the https template.
//

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/ooni/probe-cli/v3/internal/atomicx"
	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// Measures HTTPS endpoints.
//
// The zero value of this structure IS NOT valid and you MUST initialize
// all the fields marked as MANDATORY before using this structure.
type SecureFlow struct {
	// Address is the MANDATORY address to connect to.
	Address string

	// DNSCache is the MANDATORY DNS cache.
	DNSCache *DNSCache

	// IDGenerator is the MANDATORY atomic int64 to generate task IDs.
	IDGenerator *atomicx.Int64

	// Logger is the MANDATORY logger to use.
	Logger model.Logger

	// NumRedirects it the MANDATORY counter of the number of redirects.
	NumRedirects *NumRedirects

	// TestKeys is MANDATORY and contains the TestKeys.
	TestKeys *TestKeys

	// ZeroTime is the MANDATORY measurement's zero time.
	ZeroTime time.Time

	// WaitGroup is the MANDATORY wait group this task belongs to.
	WaitGroup *sync.WaitGroup

	// ALPN is the OPTIONAL ALPN to use.
	ALPN []string

	// CookieJar contains the OPTIONAL cookie jar, used for redirects.
	CookieJar http.CookieJar

	// FollowRedirects is OPTIONAL and instructs this flow
	// to follow HTTP redirects (if any).
	FollowRedirects bool

	// HostHeader is the OPTIONAL host header to use.
	HostHeader string

	// PrioSelector is the OPTIONAL priority selector to use to determine
	// whether this flow is allowed to fetch the webpage.
	PrioSelector *prioritySelector

	// Referer contains the OPTIONAL referer, used for redirects.
	Referer string

	// SNI is the OPTIONAL SNI to use.
	SNI string

	// UDPAddress is the OPTIONAL address of the UDP resolver to use. If this
	// field is not set we use a default one (e.g., `8.8.8.8:53`).
	UDPAddress string

	// URLPath is the OPTIONAL URL path.
	URLPath string

	// URLRawQuery is the OPTIONAL URL raw query.
	URLRawQuery string
}

// Start starts this task in a background goroutine.
func (t *SecureFlow) Start(ctx context.Context) {
	t.WaitGroup.Add(1)
	index := t.IDGenerator.Add(1)
	go func() {
		defer t.WaitGroup.Done() // synchronize with the parent
		t.Run(ctx, index)
	}()
}

// Run runs this task in the current goroutine.
func (t *SecureFlow) Run(parentCtx context.Context, index int64) {
	// create trace
	trace := measurexlite.NewTrace(index, t.ZeroTime)

	// start the operation logger
	ol := measurexlite.NewOperationLogger(
		t.Logger, "[#%d] GET https://%s using %s", index, t.HostHeader, t.Address,
	)

	// perform the TCP connect
	const tcpTimeout = 10 * time.Second
	tcpCtx, tcpCancel := context.WithTimeout(parentCtx, tcpTimeout)
	defer tcpCancel()
	tcpDialer := trace.NewDialerWithoutResolver(t.Logger)
	tcpConn, err := tcpDialer.DialContext(tcpCtx, "tcp", t.Address)
	t.TestKeys.AppendTCPConnectResults(trace.TCPConnects()...)
	if err != nil {
		ol.Stop(err)
		return
	}
	defer func() {
		t.TestKeys.AppendNetworkEvents(trace.NetworkEvents()...)
		tcpConn.Close()
	}()

	// perform TLS handshake
	tlsSNI, err := t.sni()
	if err != nil {
		t.TestKeys.SetFundamentalFailure(err)
		ol.Stop(err)
		return
	}
	tlsHandshaker := trace.NewTLSHandshakerStdlib(t.Logger)
	tlsConfig := &tls.Config{
		NextProtos: t.alpn(),
		RootCAs:    netxlite.NewDefaultCertPool(),
		ServerName: tlsSNI,
	}
	const tlsTimeout = 10 * time.Second
	tlsCtx, tlsCancel := context.WithTimeout(parentCtx, tlsTimeout)
	defer tlsCancel()
	tlsConn, tlsConnState, err := tlsHandshaker.Handshake(tlsCtx, tcpConn, tlsConfig)
	t.TestKeys.AppendTLSHandshakes(trace.TLSHandshakes()...)
	if err != nil {
		ol.Stop(err)
		return
	}
	defer tlsConn.Close()

	alpn := tlsConnState.NegotiatedProtocol

	// Determine whether we're allowed to fetch the webpage
	if t.PrioSelector == nil || !t.PrioSelector.permissionToFetch(t.Address) {
		ol.Stop("stop after TLS handshake")
		return
	}

	// create HTTP transport
	httpTransport := netxlite.NewHTTPTransport(
		t.Logger,
		netxlite.NewNullDialer(),
		// note: netxlite guarantees that here tlsConn is a netxlite.TLSConn
		netxlite.NewSingleUseTLSDialer(tlsConn.(netxlite.TLSConn)),
	)

	// create HTTP request
	const httpTimeout = 10 * time.Second
	httpCtx, httpCancel := context.WithTimeout(parentCtx, httpTimeout)
	defer httpCancel()
	httpReq, err := t.newHTTPRequest(httpCtx)
	if err != nil {
		if t.Referer == "" {
			// when the referer is empty, the failing URL comes from our backend
			// or from the user, so it's a fundamental failure. After that, we
			// are dealing with websites provided URLs, so we should not flag a
			// fundamental failure, because we want to see the measurement submitted.
			t.TestKeys.SetFundamentalFailure(err)
		}
		ol.Stop(err)
		return
	}

	// perform HTTP transaction
	httpResp, httpRespBody, err := t.httpTransaction(
		httpCtx,
		"tcp",
		t.Address,
		alpn,
		httpTransport,
		httpReq,
		trace,
	)
	if err != nil {
		ol.Stop(err)
		return
	}

	// if enabled, follow possible redirects
	t.maybeFollowRedirects(parentCtx, httpResp)

	// TODO: insert here additional code if needed
	_ = httpRespBody

	// completed successfully
	ol.Stop(nil)
}

// alpn returns the user-configured ALPN or a reasonable default
func (t *SecureFlow) alpn() []string {
	if len(t.ALPN) > 0 {
		return t.ALPN
	}
	return []string{"h2", "http/1.1"}
}

// sni returns the user-configured SNI or a reasonable default
func (t *SecureFlow) sni() (string, error) {
	if t.SNI != "" {
		return t.SNI, nil
	}
	addr, _, err := net.SplitHostPort(t.Address)
	if err != nil {
		return "", err
	}
	return addr, nil
}

// urlHost computes the host to include into the URL
func (t *SecureFlow) urlHost(scheme string) (string, error) {
	addr, port, err := net.SplitHostPort(t.Address)
	if err != nil {
		t.Logger.Warnf("BUG: net.SplitHostPort failed for %s: %s", t.Address, err.Error())
		return "", err
	}
	urlHost := t.HostHeader
	if urlHost == "" {
		urlHost = addr
	}
	if port == "443" && scheme == "https" {
		return urlHost, nil
	}
	urlHost = net.JoinHostPort(urlHost, port)
	return urlHost, nil
}

// newHTTPRequest creates a new HTTP request.
func (t *SecureFlow) newHTTPRequest(ctx context.Context) (*http.Request, error) {
	const urlScheme = "https"
	urlHost, err := t.urlHost(urlScheme)
	if err != nil {
		return nil, err
	}
	httpURL := &url.URL{
		Scheme:   urlScheme,
		Host:     urlHost,
		Path:     t.URLPath,
		RawQuery: t.URLRawQuery,
	}
	httpReq, err := http.NewRequestWithContext(ctx, "GET", httpURL.String(), nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Host", t.HostHeader)
	httpReq.Header.Set("Accept", model.HTTPHeaderAccept)
	httpReq.Header.Set("Accept-Language", model.HTTPHeaderAcceptLanguage)
	httpReq.Header.Set("Referer", t.Referer)
	httpReq.Header.Set("User-Agent", model.HTTPHeaderUserAgent)
	httpReq.Host = t.HostHeader
	if t.CookieJar != nil {
		for _, cookie := range t.CookieJar.Cookies(httpURL) {
			httpReq.AddCookie(cookie)
		}
	}
	return httpReq, nil
}

// httpTransaction runs the HTTP transaction and saves the results.
func (t *SecureFlow) httpTransaction(ctx context.Context, network, address, alpn string,
	txp model.HTTPTransport, req *http.Request, trace *measurexlite.Trace) (*http.Response, []byte, error) {
	const maxbody = 1 << 19
	started := trace.TimeSince(trace.ZeroTime)
	t.TestKeys.AppendNetworkEvents(measurexlite.NewAnnotationArchivalNetworkEvent(
		trace.Index, started, "http_transaction_start",
	))
	resp, err := txp.RoundTrip(req)
	var body []byte
	if err == nil {
		defer resp.Body.Close()
		if cookies := resp.Cookies(); t.CookieJar != nil && len(cookies) > 0 {
			t.CookieJar.SetCookies(req.URL, cookies)
		}
		reader := io.LimitReader(resp.Body, maxbody)
		body, err = StreamAllContext(ctx, reader)
	}
	finished := trace.TimeSince(trace.ZeroTime)
	t.TestKeys.AppendNetworkEvents(measurexlite.NewAnnotationArchivalNetworkEvent(
		trace.Index, finished, "http_transaction_done",
	))
	ev := measurexlite.NewArchivalHTTPRequestResult(
		trace.Index,
		started,
		network,
		address,
		alpn,
		txp.Network(),
		req,
		resp,
		maxbody,
		body,
		err,
		finished,
	)
	t.TestKeys.AppendRequests(ev)
	return resp, body, err
}

// maybeFollowRedirects follows redirects if configured and needed
func (t *SecureFlow) maybeFollowRedirects(ctx context.Context, resp *http.Response) {
	if !t.FollowRedirects || !t.NumRedirects.CanFollowOneMoreRedirect() {
		return // not configured or too many redirects
	}
	switch resp.StatusCode {
	case 301, 302, 307, 308:
		location, err := resp.Location()
		if err != nil {
			return // broken response from server
		}
		t.Logger.Infof("redirect to: %s", location.String())
		resolvers := &DNSResolvers{
			CookieJar:    t.CookieJar,
			DNSCache:     t.DNSCache,
			Domain:       location.Hostname(),
			IDGenerator:  t.IDGenerator,
			Logger:       t.Logger,
			NumRedirects: t.NumRedirects,
			TestKeys:     t.TestKeys,
			URL:          location,
			ZeroTime:     t.ZeroTime,
			WaitGroup:    t.WaitGroup,
			Referer:      resp.Request.URL.String(),
			Session:      nil, // no need to issue another control request
			THAddr:       "",  // ditto
			UDPAddress:   t.UDPAddress,
		}
		resolvers.Start(ctx)
	default:
		// no redirect to follow
	}
}
