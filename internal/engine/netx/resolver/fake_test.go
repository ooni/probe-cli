package resolver

import (
	"context"
	"errors"
	"io"
	"net"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
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

func NewFakeResolverThatFails() model.Resolver {
	return NewFakeResolverWithExplicitError(netxlite.ErrOODNSNoSuchHost)
}

func NewFakeResolverWithExplicitError(err error) model.Resolver {
	runtimex.PanicIfNil(err, "passed nil error")
	return &mocks.Resolver{
		MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
			return nil, err
		},
		MockNetwork: func() string {
			return "fake"
		},
		MockAddress: func() string {
			return ""
		},
		MockCloseIdleConnections: func() {
			// nothing
		},
		MockLookupHTTPS: func(ctx context.Context, domain string) (*model.HTTPSSvc, error) {
			return nil, errors.New("not implemented")
		},
		MockLookupNS: func(ctx context.Context, domain string) ([]*net.NS, error) {
			return nil, errors.New("not implemented")
		},
	}
}

func NewFakeResolverWithResult(r []string) model.Resolver {
	return &mocks.Resolver{
		MockLookupHost: func(ctx context.Context, domain string) ([]string, error) {
			return r, nil
		},
		MockNetwork: func() string {
			return "fake"
		},
		MockAddress: func() string {
			return ""
		},
		MockCloseIdleConnections: func() {
			// nothing
		},
		MockLookupHTTPS: func(ctx context.Context, domain string) (*model.HTTPSSvc, error) {
			return nil, errors.New("not implemented")
		},
		MockLookupNS: func(ctx context.Context, domain string) ([]*net.NS, error) {
			return nil, errors.New("not implemented")
		},
	}
}
