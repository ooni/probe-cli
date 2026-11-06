package httpmiddlebox

//
// Iterative network tracing
//

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sort"
	"sync"
	"syscall"
	"time"

	"github.com/ooni/probe-cli/v3/internal/logx"
	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	utls "gitlab.com/yawning/utls.git"
)

// Measures HTTP endpoints.
//
// The zero value of this structure IS NOT valid and you MUST initialize
// all the fields marked as MANDATORY before using this structure.
type HTTPTraceConfig struct {
	// Logger is the MANDATORY logger to use.
	Logger model.Logger

	// CookieJar contains the OPTIONAL cookie jar, used for redirects.
	CookieJar http.CookieJar

	// HostHeader is the OPTIONAL host header to use.
	HostHeader string

	// Referer contains the OPTIONAL referer, used for redirects.
	Referer string

	// URLPath is the OPTIONAL URL path.
	URLPath string

	// URLRawQuery is the OPTIONAL URL raw query.
	URLRawQuery string
}

// ClientIDs to map configurable inputs to uTLS fingerprints
// We use a non-zero index to map to each ClientID
var ClientIDs = map[int]*utls.ClientHelloID{
	1: &utls.HelloGolang,
	2: &utls.HelloChrome_Auto,
	3: &utls.HelloFirefox_Auto,
	4: &utls.HelloIOS_Auto,
}

// HTTPTrace performs tracing using control and target SNI
func (m *Measurer) HTTPTrace(ctx context.Context, index int64, zeroTime time.Time, logger model.Logger,
	address string, targetSNI string, trace *CompleteTrace, config *Config) {
	// perform a TCP traceroute
	trace.TCPTraceroute = m.runTraceroute(index, zeroTime, logger, address, targetSNI, config)
	// perform an iterative trace with the control SNI
	trace.ControlTrace = m.startIterativeTrace(ctx, index, zeroTime, logger, address, config.snicontrol(), config)
	// perform an iterative trace with the target SNI
	trace.TargetTrace = m.startIterativeTrace(ctx, index, zeroTime, logger, address, targetSNI, config)
}

// runTraceroute conducts an unrooted TCP traceroute towards the target IP
func (m *Measurer) runTraceroute(index int64, zeroTime time.Time, logger model.Logger,
	address string, sni string, config *Config) (tr *IterativeTraceroute) {
	tr = &IterativeTraceroute{
		SNI:        sni,
		Iterations: []*ICMPIteration{},
	}
	maxTTL := config.maxttl()
	ticker := time.NewTicker(config.delay())
	wg := new(sync.WaitGroup)
	for i := int64(1); i <= maxTTL; i++ {
		wg.Add(1)
		icmpIteration, err := tracerouteTCP(index, zeroTime, address, int(i), 3000, wg, logger, config.privacymode())
		if err != nil {
			return
		}
		tr.addIterationsTraceroute(icmpIteration)

		<-ticker.C
	}
	wg.Wait()
	return
}

// startIterativeTrace creates a Trace and calls iterativeTrace
func (m *Measurer) startIterativeTrace(ctx context.Context, index int64, zeroTime time.Time, logger model.Logger,
	address string, sni string, config *Config) (tr *IterativeTrace) {
	tr = &IterativeTrace{
		SNI:        sni,
		Iterations: []*Iteration{},
	}
	maxTTL := config.maxttl()
	m.traceWithIncreasingTTLs(ctx, index, zeroTime, logger, address, sni, maxTTL, tr, config)
	tr.Iterations = alignIterations(tr.Iterations)
	return
}

// traceWithIncreasingTTLs performs iterative tracing with increasing TTL values
func (m *Measurer) traceWithIncreasingTTLs(ctx context.Context, index int64, zeroTime time.Time, logger model.Logger,
	address string, sni string, maxTTL int64, trace *IterativeTrace, config *Config) {
	ticker := time.NewTicker(config.delay())
	wg := new(sync.WaitGroup)
	for i := int64(1); i <= maxTTL; i++ {
		wg.Add(1)
		go m.handshakeWithTTL(ctx, index, zeroTime, logger, address, sni, int(i), trace, wg, config)
		<-ticker.C
	}
	wg.Wait()
}

// handshakeWithTTL performs the TLS Handshake using the passed ttl value
func (m *Measurer) handshakeWithTTL(ctx context.Context, index int64, zeroTime time.Time, logger model.Logger,
	address string, sni string, ttl int, tr *IterativeTrace, wg *sync.WaitGroup, config *Config) {
	defer wg.Done()
	trace := measurexlite.NewTrace(index, zeroTime)
	// 1. Connect to the target IP
	newHTTPTraceConfig := HTTPTraceConfig{}
	newHTTPTraceConfig.Logger = logger
	d := NewDialerTTLWrapper()
	ol := logx.NewOperationLogger(logger, "[#%d] GET https://%s using %s", index, sni, address)
	conn, err := d.DialContext(ctx, "tcp", address)
	if err != nil {
		iteration := newIterationFromArchivalHTTPRequestResult(ttl, err, nil, nil, sni)
		tr.addIterations(iteration)
		ol.Stop(err)
		return
	}
	defer conn.Close()
	// 2. Set the TTL to the passed value
	err = setConnTTL(conn, ttl)
	if err != nil {
		iteration := newIterationFromArchivalHTTPRequestResult(ttl, err, nil, nil, sni)
		tr.addIterations(iteration)
		ol.Stop(err)
		return
	}
	// 3. Perform the HTTP request and extract the SO_ERROR value (if any)
	httpTransport := netxlite.NewHTTPTransportWithOptions(
		logger,
		netxlite.NewSingleUseDialer(conn),
		netxlite.NewNullTLSDialer(),
	)

	const httpTimeout = 10 * time.Second
	httpCtx, httpCancel := context.WithTimeout(ctx, httpTimeout)
	defer httpCancel()
	httpReq, err := m.newHTTPRequest(httpCtx, address, newHTTPTraceConfig)
	if err != nil {
		if newHTTPTraceConfig.Referer == "" {
			// when the referer is empty, the failing URL comes from our backend
			// or from the user, so it's a fundamental failure. After that, we
			// are dealing with websites provided URLs, so we should not flag a
			// fundamental failure, because we want to see the measurement submitted.
			iteration := newIterationFromArchivalHTTPRequestResult(ttl, err, nil, nil, sni)
			tr.addIterations(iteration)
		}
		ol.Stop(err)
	}

	// perform HTTP transaction
	httpRequestResult, err := m.httpTransaction(
		httpCtx,
		"tcp",
		address,
		"",
		httpTransport,
		httpReq,
		trace,
		newHTTPTraceConfig,
	)
	if err != nil {
		ol.Stop(err)
	}

	fmt.Println(httpRequestResult.Response.Headers)

	ol.Stop(err)
	soErr := extractSoError(conn)
	// 4. reset the TTL value to ensure that conn closes successfully
	// Note: Do not check for errors here
	_ = setConnTTL(conn, 64)
	iteration := newIterationFromArchivalHTTPRequestResult(ttl, err, soErr, httpRequestResult, sni)
	tr.addIterations(iteration)
}

// urlHost computes the host to include into the URL
func (m *Measurer) urlHost(scheme string, address string, newHTTPTraceConfig HTTPTraceConfig) (string, error) {
	addr, port, err := net.SplitHostPort(address)
	if err != nil {
		newHTTPTraceConfig.Logger.Warnf("BUG: net.SplitHostPort failed for %s: %s", address, err.Error())
		return "", err
	}
	urlHost := newHTTPTraceConfig.HostHeader
	if urlHost == "" {
		urlHost = addr
	}
	if port == "80" && scheme == "http" {
		return urlHost, nil
	}
	urlHost = net.JoinHostPort(urlHost, port)
	return urlHost, nil
}

// newHTTPRequest creates a new HTTP request.
func (m *Measurer) newHTTPRequest(ctx context.Context, address string, newHTTPTraceConfig HTTPTraceConfig) (*http.Request, error) {

	const urlScheme = "http"
	urlHost, err := m.urlHost(urlScheme, address, newHTTPTraceConfig)
	if err != nil {
		return nil, err
	}
	httpURL := &url.URL{
		Scheme:   urlScheme,
		Host:     urlHost,
		Path:     newHTTPTraceConfig.URLPath,
		RawQuery: newHTTPTraceConfig.URLRawQuery,
	}
	httpReq, err := http.NewRequestWithContext(ctx, "GET", httpURL.String(), nil)
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Host", newHTTPTraceConfig.HostHeader)
	httpReq.Header.Set("Accept", model.HTTPHeaderAccept)
	httpReq.Header.Set("Accept-Language", model.HTTPHeaderAcceptLanguage)
	httpReq.Header.Set("Referer", newHTTPTraceConfig.Referer)
	httpReq.Header.Set("User-Agent", model.HTTPHeaderUserAgent)
	httpReq.Host = newHTTPTraceConfig.HostHeader
	if newHTTPTraceConfig.CookieJar != nil {
		for _, cookie := range newHTTPTraceConfig.CookieJar.Cookies(httpURL) {
			httpReq.AddCookie(cookie)
		}
	}
	return httpReq, nil
}

// httpTransaction runs the HTTP transaction and saves the results.
func (m *Measurer) httpTransaction(ctx context.Context, network, address, alpn string,
	txp model.HTTPTransport, req *http.Request, trace *measurexlite.Trace, newHTTPTraceConfig HTTPTraceConfig) (*model.ArchivalHTTPRequestResult, error) {
	const maxbody = 1 << 19
	started := trace.TimeSince(trace.ZeroTime())

	resp, err := txp.RoundTrip(req)
	var body []byte
	if err == nil {
		defer resp.Body.Close()
		if cookies := resp.Cookies(); newHTTPTraceConfig.CookieJar != nil && len(cookies) > 0 {
			newHTTPTraceConfig.CookieJar.SetCookies(req.URL, cookies)
		}
		reader := io.LimitReader(resp.Body, maxbody)
		body, err = netxlite.StreamAllContext(ctx, reader) //err can be non-nil even if a response was received
	}

	finished := trace.TimeSince(trace.ZeroTime())

	ev := measurexlite.NewArchivalHTTPRequestResult(
		trace.Index(),
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
		trace.Tags()...,
	)

	return ev, err
}

// extractSoError fetches the SO_ERROR value and returns a non-nil error if
// it qualifies as a valid ICMP soft error
// Note: The passed conn must be of type dialerTTLWrapperConn
func extractSoError(conn net.Conn) error {
	soErrno, err := getSoErr(conn)
	if err != nil || errors.Is(soErrno, syscall.Errno(0)) {
		return nil
	}
	soErr := netxlite.MaybeNewErrWrapper(netxlite.ClassifyGenericError, netxlite.TLSHandshakeOperation, soErrno)
	return soErr
}

// genTLSConfig generates tls.Config from a given SNI
func genTLSConfig(sni string) *tls.Config {
	// See https://github.com/ooni/probe/issues/2413 to understand
	// why we're using nil to force netxlite to use the cached
	// default Mozilla cert pool.
	return &tls.Config{ // #nosec G402 - we need to use a large TLS versions range for measuring
		RootCAs:            nil,
		ServerName:         sni,
		NextProtos:         []string{"h2", "http/1.1"},
		InsecureSkipVerify: true, // #nosec G402 - it's fine to skip verify in a nettest
	}
}

// alignIterEvents sorts the iterEvents according to increasing TTL
// and stops when we receive a nil or connection_reset
func alignIterations(in []*Iteration) (out []*Iteration) {
	out = []*Iteration{}
	sort.Slice(in, func(i int, j int) bool {
		return in[i].TTL < in[j].TTL
	})
	for _, iter := range in {
		out = append(out, iter)
		if iter.HTTPRequestResult.Failure == nil || *iter.HTTPRequestResult.Failure == netxlite.FailureConnectionReset {
			break
		}
	}
	return out
}
