package resolver

import (
	"context"
	"errors"
	"io"
	"net"
	"time"

	"github.com/ooni/probe-cli/v3/internal/atomicx"
	"github.com/ooni/probe-cli/v3/internal/model"
)

type FakeDialer struct {
	Conn net.Conn
	Err  error
}

func (d FakeDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	time.Sleep(10 * time.Microsecond)
	return d.Conn, d.Err
}

type FakeConn struct {
	ReadError             error
	ReadData              []byte
	SetDeadlineError      error
	SetReadDeadlineError  error
	SetWriteDeadlineError error
	WriteError            error
}

func (c *FakeConn) Read(b []byte) (int, error) {
	if len(c.ReadData) > 0 {
		n := copy(b, c.ReadData)
		c.ReadData = c.ReadData[n:]
		return n, nil
	}
	if c.ReadError != nil {
		return 0, c.ReadError
	}
	return 0, io.EOF
}

func (c *FakeConn) Write(b []byte) (n int, err error) {
	if c.WriteError != nil {
		return 0, c.WriteError
	}
	n = len(b)
	return
}

func (*FakeConn) Close() (err error) {
	return
}

func (*FakeConn) LocalAddr() net.Addr {
	return &net.TCPAddr{}
}

func (*FakeConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{}
}

func (c *FakeConn) SetDeadline(t time.Time) (err error) {
	return c.SetDeadlineError
}

func (c *FakeConn) SetReadDeadline(t time.Time) (err error) {
	return c.SetReadDeadlineError
}

func (c *FakeConn) SetWriteDeadline(t time.Time) (err error) {
	return c.SetWriteDeadlineError
}

type FakeTransport struct {
	Data []byte
	Err  error
}

func (ft FakeTransport) RoundTrip(ctx context.Context, query []byte) ([]byte, error) {
	return ft.Data, ft.Err
}

func (ft FakeTransport) RequiresPadding() bool {
	return false
}

func (ft FakeTransport) Address() string {
	return ""
}

func (ft FakeTransport) Network() string {
	return "fake"
}

func (fk FakeTransport) CloseIdleConnections() {
	// nothing to do
}

type FakeEncoder struct {
	Data []byte
	Err  error
}

func (fe FakeEncoder) Encode(domain string, qtype uint16, padding bool) ([]byte, error) {
	return fe.Data, fe.Err
}

type FakeResolver struct {
	NumFailures *atomicx.Int64
	Err         error
	Result      []string
}

func NewFakeResolverThatFails() FakeResolver {
	return FakeResolver{NumFailures: &atomicx.Int64{}, Err: errNotFound}
}

func NewFakeResolverWithResult(r []string) FakeResolver {
	return FakeResolver{NumFailures: &atomicx.Int64{}, Result: r}
}

var errNotFound = &net.DNSError{
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

func (c FakeResolver) CloseIdleConnections() {}

func (c FakeResolver) LookupHTTPS(
	ctx context.Context, domain string) (*model.HTTPSSvc, error) {
	return nil, errors.New("not implemented")
}

var _ model.Resolver = FakeResolver{}
