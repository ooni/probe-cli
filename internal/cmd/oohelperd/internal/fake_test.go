package internal

import (
	"context"
	"io"
	"net"
	"net/http"
	"time"

	"github.com/ooni/probe-cli/v3/internal/atomicx"
	"github.com/ooni/probe-cli/v3/internal/engine/netx"
	"github.com/ooni/probe-cli/v3/internal/iox"
)

type FakeResolver struct {
	NumFailures *atomicx.Int64
	Err         error
	Result      []string
}

func NewFakeResolverThatFails() FakeResolver {
	return FakeResolver{NumFailures: &atomicx.Int64{}, Err: ErrNotFound}
}

func NewFakeResolverWithResult(r []string) FakeResolver {
	return FakeResolver{NumFailures: &atomicx.Int64{}, Result: r}
}

var ErrNotFound = &net.DNSError{
	Err: "no such host",
}

func (c FakeResolver) LookupHost(ctx context.Context, hostname string) ([]string, error) {
	time.Sleep(10 * time.Microsecond)
	if c.Err != nil {
		if c.NumFailures != nil {
			c.NumFailures.Add(1)
		}
		return nil, c.Err
	}
	return c.Result, nil
}

func (c FakeResolver) Network() string {
	return "fake"
}

func (c FakeResolver) Address() string {
	return ""
}

var _ netx.Resolver = FakeResolver{}

type FakeTransport struct {
	Err  error
	Func func(*http.Request) (*http.Response, error)
	Resp *http.Response
}

func (txp FakeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	time.Sleep(10 * time.Microsecond)
	if txp.Func != nil {
		return txp.Func(req)
	}
	if req.Body != nil {
		iox.ReadAllContext(req.Context(), req.Body)
		req.Body.Close()
	}
	if txp.Err != nil {
		return nil, txp.Err
	}
	txp.Resp.Request = req // non thread safe but it doesn't matter
	return txp.Resp, nil
}

func (txp FakeTransport) CloseIdleConnections() {}

var _ netx.HTTPRoundTripper = FakeTransport{}

type FakeBody struct {
	Data []byte
	Err  error
}

func (fb *FakeBody) Read(p []byte) (int, error) {
	time.Sleep(10 * time.Microsecond)
	if fb.Err != nil {
		return 0, fb.Err
	}
	if len(fb.Data) <= 0 {
		return 0, io.EOF
	}
	n := copy(p, fb.Data)
	fb.Data = fb.Data[n:]
	return n, nil
}

func (fb *FakeBody) Close() error {
	return nil
}

var _ io.ReadCloser = &FakeBody{}

type FakeResponseWriter struct {
	Body       [][]byte
	HeaderMap  http.Header
	StatusCode int
}

func NewFakeResponseWriter() *FakeResponseWriter {
	return &FakeResponseWriter{HeaderMap: make(http.Header)}
}

func (frw *FakeResponseWriter) Header() http.Header {
	return frw.HeaderMap
}

func (frw *FakeResponseWriter) Write(b []byte) (int, error) {
	frw.Body = append(frw.Body, b)
	return len(b), nil
}

func (frw *FakeResponseWriter) WriteHeader(statusCode int) {
	frw.StatusCode = statusCode
}

var _ http.ResponseWriter = &FakeResponseWriter{}
