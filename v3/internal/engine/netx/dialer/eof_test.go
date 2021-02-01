package dialer

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"time"
)

type EOFDialer struct{}

func (EOFDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	time.Sleep(10 * time.Microsecond)
	return nil, io.EOF
}

type EOFConnDialer struct{}

func (EOFConnDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	return EOFConn{}, nil
}

type EOFConn struct {
	net.Conn
}

func (EOFConn) Read(p []byte) (int, error) {
	time.Sleep(10 * time.Microsecond)
	return 0, io.EOF
}

func (EOFConn) Write(p []byte) (int, error) {
	time.Sleep(10 * time.Microsecond)
	return 0, io.EOF
}

func (EOFConn) Close() error {
	time.Sleep(10 * time.Microsecond)
	return io.EOF
}

func (EOFConn) LocalAddr() net.Addr {
	return EOFAddr{}
}

func (EOFConn) RemoteAddr() net.Addr {
	return EOFAddr{}
}

func (EOFConn) SetDeadline(t time.Time) error {
	return nil
}

func (EOFConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (EOFConn) SetWriteDeadline(t time.Time) error {
	return nil
}

type EOFAddr struct{}

func (EOFAddr) Network() string {
	return "tcp"
}

func (EOFAddr) String() string {
	return "127.0.0.1:1234"
}

type EOFTLSHandshaker struct{}

func (EOFTLSHandshaker) Handshake(
	ctx context.Context, conn net.Conn, config *tls.Config,
) (net.Conn, tls.ConnectionState, error) {
	time.Sleep(10 * time.Microsecond)
	return nil, tls.ConnectionState{}, io.EOF
}
