package mocks

import (
	"context"
	"crypto/tls"
	"net"
	"syscall"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/quic-go/quic-go"
)

// QUICDialer is a mockable netxlite.QUICDialer.
type QUICDialer struct {
	// MockDialContext allows mocking DialContext.
	MockDialContext func(ctx context.Context, address string,
		tlsConfig *tls.Config, quicConfig *quic.Config) (model.QUICConn, error)

	// MockCloseIdleConnections allows mocking CloseIdleConnections.
	MockCloseIdleConnections func()
}

var _ model.QUICDialer = &QUICDialer{}

// DialContext calls MockDialContext.
func (qcd *QUICDialer) DialContext(ctx context.Context, address string,
	tlsConfig *tls.Config, quicConfig *quic.Config) (model.QUICConn, error) {
	return qcd.MockDialContext(ctx, address, tlsConfig, quicConfig)
}

// CloseIdleConnections calls MockCloseIdleConnections.
func (qcd *QUICDialer) CloseIdleConnections() {
	qcd.MockCloseIdleConnections()
}

// QUICConn is a mockable model.QUICConn.
type QUICConn struct {
	MockCloseWithError    func(code quic.ApplicationErrorCode, reason string) error
	MockHandshakeComplete func() <-chan struct{}
	MockConnectionState   func() quic.ConnectionState
	MockLocalAddr         func() net.Addr
	MockRemoteAddr        func() net.Addr
	MockContext           func() context.Context
}

var _ model.QUICConn = &QUICConn{}

// CloseWithError calls MockCloseWithError.
func (c *QUICConn) CloseWithError(
	code quic.ApplicationErrorCode, reason string) error {
	return c.MockCloseWithError(code, reason)
}

// HandshakeComplete calls MockHandshakeComplete.
func (c *QUICConn) HandshakeComplete() <-chan struct{} {
	return c.MockHandshakeComplete()
}

// ConnectionState calls MockConnectionState.
func (c *QUICConn) ConnectionState() quic.ConnectionState {
	return c.MockConnectionState()
}

// LocalAddr calls MockLocalAddr.
func (c *QUICConn) LocalAddr() net.Addr {
	return c.MockLocalAddr()
}

// RemoteAddr calls MockRemoteAddr.
func (c *QUICConn) RemoteAddr() net.Addr {
	return c.MockRemoteAddr()
}

// Context calls MockContext.
func (c *QUICConn) Context() context.Context {
	return c.MockContext()
}

// UDPLikeConn is an UDP conn used by QUIC.
type UDPLikeConn struct {
	MockWriteTo          func(p []byte, addr net.Addr) (int, error)
	MockClose            func() error
	MockLocalAddr        func() net.Addr
	MockRemoteAddr       func() net.Addr
	MockSetDeadline      func(t time.Time) error
	MockSetReadDeadline  func(t time.Time) error
	MockSetWriteDeadline func(t time.Time) error
	MockReadFrom         func(p []byte) (int, net.Addr, error)
	MockSyscallConn      func() (syscall.RawConn, error)
	MockSetReadBuffer    func(n int) error
}

var _ model.UDPLikeConn = &UDPLikeConn{}

// WriteTo calls MockWriteTo.
func (c *UDPLikeConn) WriteTo(p []byte, addr net.Addr) (int, error) {
	return c.MockWriteTo(p, addr)
}

// Close calls MockClose.
func (c *UDPLikeConn) Close() error {
	return c.MockClose()
}

// LocalAddr calls MockLocalAddr.
func (c *UDPLikeConn) LocalAddr() net.Addr {
	return c.MockLocalAddr()
}

// RemoteAddr calls MockRemoteAddr.
func (c *UDPLikeConn) RemoteAddr() net.Addr {
	return c.MockRemoteAddr()
}

// SetDeadline calls MockSetDeadline.
func (c *UDPLikeConn) SetDeadline(t time.Time) error {
	return c.MockSetDeadline(t)
}

// SetReadDeadline calls MockSetReadDeadline.
func (c *UDPLikeConn) SetReadDeadline(t time.Time) error {
	return c.MockSetReadDeadline(t)
}

// SetWriteDeadline calls MockSetWriteDeadline.
func (c *UDPLikeConn) SetWriteDeadline(t time.Time) error {
	return c.MockSetWriteDeadline(t)
}

// ReadFrom calls MockReadFrom.
func (c *UDPLikeConn) ReadFrom(b []byte) (int, net.Addr, error) {
	return c.MockReadFrom(b)
}

// SyscallConn calls MockSyscallConn.
func (c *UDPLikeConn) SyscallConn() (syscall.RawConn, error) {
	return c.MockSyscallConn()
}

// SetReadBuffer calls MockSetReadBuffer.
func (c *UDPLikeConn) SetReadBuffer(n int) error {
	return c.MockSetReadBuffer(n)
}
