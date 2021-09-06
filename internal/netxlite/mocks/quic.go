package mocks

import (
	"context"
	"crypto/tls"
	"net"
	"syscall"
	"time"

	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/netxlite/quicx"
)

// QUICListener is a mockable netxlite.QUICListener.
type QUICListener struct {
	MockListen func(addr *net.UDPAddr) (quicx.UDPLikeConn, error)
}

// Listen calls MockListen.
func (ql *QUICListener) Listen(addr *net.UDPAddr) (quicx.UDPLikeConn, error) {
	return ql.MockListen(addr)
}

// QUICDialer is a mockable netxlite.QUICDialer.
type QUICDialer struct {
	// MockDialContext allows mocking DialContext.
	MockDialContext func(ctx context.Context, network, address string,
		tlsConfig *tls.Config, quicConfig *quic.Config) (quic.EarlySession, error)

	// MockCloseIdleConnections allows mocking CloseIdleConnections.
	MockCloseIdleConnections func()
}

// DialContext calls MockDialContext.
func (qcd *QUICDialer) DialContext(ctx context.Context, network, address string,
	tlsConfig *tls.Config, quicConfig *quic.Config) (quic.EarlySession, error) {
	return qcd.MockDialContext(ctx, network, address, tlsConfig, quicConfig)
}

// CloseIdleConnections calls MockCloseIdleConnections.
func (qcd *QUICDialer) CloseIdleConnections() {
	qcd.MockCloseIdleConnections()
}

// QUICEarlySession is a mockable quic.EarlySession.
type QUICEarlySession struct {
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
	MockNextSession       func() quic.Session
	MockSendMessage       func(b []byte) error
	MockReceiveMessage    func() ([]byte, error)
}

var _ quic.EarlySession = &QUICEarlySession{}

// AcceptStream calls MockAcceptStream.
func (s *QUICEarlySession) AcceptStream(ctx context.Context) (quic.Stream, error) {
	return s.MockAcceptStream(ctx)
}

// AcceptUniStream calls MockAcceptUniStream.
func (s *QUICEarlySession) AcceptUniStream(ctx context.Context) (quic.ReceiveStream, error) {
	return s.MockAcceptUniStream(ctx)
}

// OpenStream calls MockOpenStream.
func (s *QUICEarlySession) OpenStream() (quic.Stream, error) {
	return s.MockOpenStream()
}

// OpenStreamSync calls MockOpenStreamSync.
func (s *QUICEarlySession) OpenStreamSync(ctx context.Context) (quic.Stream, error) {
	return s.MockOpenStreamSync(ctx)
}

// OpenUniStream calls MockOpenUniStream.
func (s *QUICEarlySession) OpenUniStream() (quic.SendStream, error) {
	return s.MockOpenUniStream()
}

// OpenUniStreamSync calls MockOpenUniStreamSync.
func (s *QUICEarlySession) OpenUniStreamSync(ctx context.Context) (quic.SendStream, error) {
	return s.MockOpenUniStreamSync(ctx)
}

// LocalAddr class MockLocalAddr.
func (c *QUICEarlySession) LocalAddr() net.Addr {
	return c.MockLocalAddr()
}

// RemoteAddr calls MockRemoteAddr.
func (c *QUICEarlySession) RemoteAddr() net.Addr {
	return c.MockRemoteAddr()
}

// CloseWithError calls MockCloseWithError.
func (c *QUICEarlySession) CloseWithError(
	code quic.ApplicationErrorCode, reason string) error {
	return c.MockCloseWithError(code, reason)
}

// Context calls MockContext.
func (s *QUICEarlySession) Context() context.Context {
	return s.MockContext()
}

// ConnectionState calls MockConnectionState.
func (s *QUICEarlySession) ConnectionState() quic.ConnectionState {
	return s.MockConnectionState()
}

// HandshakeComplete calls MockHandshakeComplete.
func (s *QUICEarlySession) HandshakeComplete() context.Context {
	return s.MockHandshakeComplete()
}

// NextSession calls MockNextSession.
func (s *QUICEarlySession) NextSession() quic.Session {
	return s.MockNextSession()
}

// SendMessage calls MockSendMessage.
func (s *QUICEarlySession) SendMessage(b []byte) error {
	return s.MockSendMessage(b)
}

// ReceiveMessage calls MockReceiveMessage.
func (s *QUICEarlySession) ReceiveMessage() ([]byte, error) {
	return s.MockReceiveMessage()
}

// QUICUDPConn is an UDP conn used by QUIC.
type QUICUDPConn struct {
	MockWriteTo          func(p []byte, addr net.Addr) (int, error)
	MockClose            func() error
	MockLocalAddr        func() net.Addr
	MockRemoteAddr       func() net.Addr
	MockSetDeadline      func(t time.Time) error
	MockSetReadDeadline  func(t time.Time) error
	MockSetWriteDeadline func(t time.Time) error
	MockReadFrom         func(p []byte) (n int, addr net.Addr, err error)
	MockSyscallConn      func() (syscall.RawConn, error)
	MockSetReadBuffer    func(n int) error
}

var _ quicx.UDPLikeConn = &QUICUDPConn{}

// WriteTo calls MockWriteTo.
func (c *QUICUDPConn) WriteTo(p []byte, addr net.Addr) (int, error) {
	return c.MockWriteTo(p, addr)
}

// Close calls MockClose.
func (c *QUICUDPConn) Close() error {
	return c.MockClose()
}

// LocalAddr calls MockLocalAddr.
func (c *QUICUDPConn) LocalAddr() net.Addr {
	return c.MockLocalAddr()
}

// RemoteAddr calls MockRemoteAddr.
func (c *QUICUDPConn) RemoteAddr() net.Addr {
	return c.MockRemoteAddr()
}

// SetDeadline calls MockSetDeadline.
func (c *QUICUDPConn) SetDeadline(t time.Time) error {
	return c.MockSetDeadline(t)
}

// SetReadDeadline calls MockSetReadDeadline.
func (c *QUICUDPConn) SetReadDeadline(t time.Time) error {
	return c.MockSetReadDeadline(t)
}

// SetWriteDeadline calls MockSetWriteDeadline.
func (c *QUICUDPConn) SetWriteDeadline(t time.Time) error {
	return c.MockSetWriteDeadline(t)
}

// ReadFrom calls MockReadFrom.
func (c *QUICUDPConn) ReadFrom(b []byte) (int, net.Addr, error) {
	return c.MockReadFrom(b)
}

// SyscallConn calls MockSyscallConn.
func (c *QUICUDPConn) SyscallConn() (syscall.RawConn, error) {
	return c.MockSyscallConn()
}

// SetReadBuffer calls MockSetReadBuffer.
func (c *QUICUDPConn) SetReadBuffer(n int) error {
	return c.MockSetReadBuffer(n)
}
