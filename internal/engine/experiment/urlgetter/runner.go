package urlgetter

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/engine/httpheader"
	"github.com/ooni/probe-cli/v3/internal/engine/netx"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/errorx"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

const httpRequestFailed = "http_request_failed"

// ErrHTTPRequestFailed indicates that the HTTP request failed.
var ErrHTTPRequestFailed = &errorx.ErrWrapper{
	Failure:    httpRequestFailed,
	Operation:  errorx.TopLevelOperation,
	WrappedErr: errors.New(httpRequestFailed),
}

// The Runner job is to run a single measurement
type Runner struct {
	Config     Config
	HTTPConfig netx.Config
	Target     string
}

// Run runs a measurement and returns the measurement result
func (r Runner) Run(ctx context.Context) error {
	targetURL, err := url.Parse(r.Target)
	if err != nil {
		return fmt.Errorf("urlgetter: invalid target URL: %w", err)
	}
	switch targetURL.Scheme {
	case "http", "https":
		return r.httpGet(ctx, r.Target)
	case "dnslookup":
		return r.dnsLookup(ctx, targetURL.Hostname())
	case "tlshandshake":
		return r.tlsHandshake(ctx, targetURL.Host)
	case "tcpconnect":
		return r.tcpConnect(ctx, targetURL.Host)
	default:
		return errors.New("unknown targetURL scheme")
	}
}

// MaybeUserAgent returns ua if ua is not empty. Otherwise it
// returns httpheader.RandomUserAgent().
func MaybeUserAgent(ua string) string {
	if ua == "" {
		ua = httpheader.UserAgent()
	}
	return ua
}

func (r Runner) httpGet(ctx context.Context, url string) error {
	// Implementation note: empty Method implies using the GET method
	req, err := http.NewRequest(r.Config.Method, url, nil)
	runtimex.PanicOnError(err, "http.NewRequest failed")
	req = req.WithContext(ctx)
	req.Header.Set("Accept", httpheader.Accept())
	req.Header.Set("Accept-Language", httpheader.AcceptLanguage())
	req.Header.Set("User-Agent", MaybeUserAgent(r.Config.UserAgent))
	if r.Config.HTTPHost != "" {
		req.Host = r.Config.HTTPHost
	}
	// Implementation note: the following cookiejar accepts all cookies
	// from all domains. As such, would not be safe for usage where cookies
	// matter, but it's totally fine for performing measurements.
	jar, err := cookiejar.New(nil)
	runtimex.PanicOnError(err, "cookiejar.New failed")
	httpClient := &http.Client{
		Jar:       jar,
		Transport: netx.NewHTTPTransport(r.HTTPConfig),
	}
	if r.Config.NoFollowRedirects {
		httpClient.CheckRedirect = func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}
	defer httpClient.CloseIdleConnections()
	resp, err := httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if _, err = io.Copy(ioutil.Discard, resp.Body); err != nil {
		return err
	}
	// Implementation note: we shall check for this error once we have read the
	// whole body. Even though we discard the body, we want to know whether we
	// see any error when reading the body before inspecting the HTTP status code.
	if resp.StatusCode >= 400 && r.Config.FailOnHTTPError {
		return ErrHTTPRequestFailed
	}
	return nil
}

func (r Runner) dnsLookup(ctx context.Context, hostname string) error {
	resolver := netx.NewResolver(r.HTTPConfig)
	_, err := resolver.LookupHost(ctx, hostname)
	return err
}

func (r Runner) tlsHandshake(ctx context.Context, address string) error {
	tlsDialer := netx.NewTLSDialer(r.HTTPConfig)
	conn, err := tlsDialer.DialTLSContext(ctx, "tcp", address)
	if conn != nil {
		conn.Close()
	}
	return err
}

func (r Runner) tcpConnect(ctx context.Context, address string) error {
	dialer := netx.NewDialer(r.HTTPConfig)
	conn, err := dialer.DialContext(ctx, "tcp", address)
	if conn != nil {
		conn.Close()
	}
	return err
}
