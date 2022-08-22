package mocks

import (
	"context"
	"crypto/tls"
	"net"
	"syscall"
	"time"

	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// QUICListener is a mockable netxlite.QUICListener.
type QUICListener struct {
	MockListen func(addr *net.UDPAddr) (model.UDPLikeConn, error)
}

// Listen calls MockListen.
func (ql *QUICListener) Listen(addr *net.UDPAddr) (model.UDPLikeConn, error) {
	return ql.MockListen(addr)
}

// QUICDialer is a mockable netxlite.QUICDialer.
type QUICDialer struct {
	// MockDialContext allows mocking DialContext.
	MockDialContext func(ctx context.Context, address string,
		tlsConfig *tls.Config, quicConfig *quic.Config) (quic.EarlyConnection, error)

	// MockCloseIdleConnections allows mocking CloseIdleConnections.
	MockCloseIdleConnections func()
}

var _ model.QUICDialer = &QUICDialer{}

// DialContext calls MockDialContext.
func (qcd *QUICDialer) DialContext(ctx context.Context, address string,
	tlsConfig *tls.Config, quicConfig *quic.Config) (quic.EarlyConnection, error) {
	return qcd.MockDialContext(ctx, address, tlsConfig, quicConfig)
}

// CloseIdleConnections calls MockCloseIdleConnections.
func (qcd *QUICDialer) CloseIdleConnections() {
	qcd.MockCloseIdleConnections()
}

// QUICEarlyConnection is a mockable quic.EarlyConnection.
type QUICEarlyConnection struct {
	MockAcceptStream      func(context.Context) (quic.Stream, error)
	MockAcceptUniStream   func(context.Context) (quic.ReceiveStream, error)
	MockOpenStream        func() (quic.Stream, error)
	MockOpenStreamSync    func(ctx context.Context) (quic.Stream, error)
	MockOpenUniStream     func() (quic.SendStream, error)
	MockOpenUniStreamSync func(ctx context.Context) (quic.SendStream, error)
	MockLocalAddr         func() net.Addr
	MockRemoteAddr        func() net.Addr
	MockCloseWithError    func(code quic.ApplicationErrorCode, reason string) error
	MockContext           func() context.Context
	MockConnectionState   func() quic.ConnectionState
	MockHandshakeComplete func() context.Context
	MockNextConnection    func() quic.Connection
	MockSendMessage       func(b []byte) error
	MockReceiveMessage    func() ([]byte, error)
}

var _ quic.EarlyConnection = &QUICEarlyConnection{}

// AcceptStream calls MockAcceptStream.
func (s *QUICEarlyConnection) AcceptStream(ctx context.Context) (quic.Stream, error) {
	return s.MockAcceptStream(ctx)
}

// AcceptUniStream calls MockAcceptUniStream.
func (s *QUICEarlyConnection) AcceptUniStream(ctx context.Context) (quic.ReceiveStream, error) {
	return s.MockAcceptUniStream(ctx)
}

// OpenStream calls MockOpenStream.
func (s *QUICEarlyConnection) OpenStream() (quic.Stream, error) {
	return s.MockOpenStream()
}

// OpenStreamSync calls MockOpenStreamSync.
func (s *QUICEarlyConnection) OpenStreamSync(ctx context.Context) (quic.Stream, error) {
	return s.MockOpenStreamSync(ctx)
}

// OpenUniStream calls MockOpenUniStream.
func (s *QUICEarlyConnection) OpenUniStream() (quic.SendStream, error) {
	return s.MockOpenUniStream()
}

// OpenUniStreamSync calls MockOpenUniStreamSync.
func (s *QUICEarlyConnection) OpenUniStreamSync(ctx context.Context) (quic.SendStream, error) {
	return s.MockOpenUniStreamSync(ctx)
}

// LocalAddr class MockLocalAddr.
func (c *QUICEarlyConnection) LocalAddr() net.Addr {
	return c.MockLocalAddr()
}

// RemoteAddr calls MockRemoteAddr.
func (c *QUICEarlyConnection) RemoteAddr() net.Addr {
	return c.MockRemoteAddr()
}

// CloseWithError calls MockCloseWithError.
func (c *QUICEarlyConnection) CloseWithError(
	code quic.ApplicationErrorCode, reason string) error {
	return c.MockCloseWithError(code, reason)
}

// Context calls MockContext.
func (s *QUICEarlyConnection) Context() context.Context {
	return s.MockContext()
}

// ConnectionState calls MockConnectionState.
func (s *QUICEarlyConnection) ConnectionState() quic.ConnectionState {
	return s.MockConnectionState()
}

// HandshakeComplete calls MockHandshakeComplete.
func (s *QUICEarlyConnection) HandshakeComplete() context.Context {
	return s.MockHandshakeComplete()
}

// NextConnection calls MockNextConnection.
func (s *QUICEarlyConnection) NextConnection() quic.Connection {
	return s.MockNextConnection()
}

// SendMessage calls MockSendMessage.
func (s *QUICEarlyConnection) SendMessage(b []byte) error {
	return s.MockSendMessage(b)
}

// ReceiveMessage calls MockReceiveMessage.
func (s *QUICEarlyConnection) ReceiveMessage() ([]byte, error) {
	return s.MockReceiveMessage()
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
