package errorsx

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net"
	"testing"

	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/netxmocks"
)

func TestErrorWrapperQUICListenerSuccess(t *testing.T) {
	ql := &ErrorWrapperQUICListener{
		QUICListener: &netxmocks.QUICListener{
			MockListen: func(addr *net.UDPAddr) (quic.OOBCapablePacketConn, error) {
				return &net.UDPConn{}, nil
			},
		},
	}
	pconn, err := ql.Listen(&net.UDPAddr{})
	if err != nil {
		t.Fatal(err)
	}
	pconn.Close()
}

func TestErrorWrapperQUICListenerFailure(t *testing.T) {
	ql := &ErrorWrapperQUICListener{
		QUICListener: &netxmocks.QUICListener{
			MockListen: func(addr *net.UDPAddr) (quic.OOBCapablePacketConn, error) {
				return nil, io.EOF
			},
		},
	}
	pconn, err := ql.Listen(&net.UDPAddr{})
	if err.Error() != "eof_error" {
		t.Fatal("not the error we expected", err)
	}
	if pconn != nil {
		t.Fatal("expected nil pconn here")
	}
}

func TestErrorWrapperUDPConnWriteToSuccess(t *testing.T) {
	quc := &errorWrapperUDPConn{
		OOBCapablePacketConn: &netxmocks.QUICUDPConn{
			MockWriteTo: func(p []byte, addr net.Addr) (int, error) {
				return 10, nil
			},
		},
	}
	pkt := make([]byte, 128)
	addr := &net.UDPAddr{}
	cnt, err := quc.WriteTo(pkt, addr)
	if err != nil {
		t.Fatal("not the error we expected", err)
	}
	if cnt != 10 {
		t.Fatal("expected 10 here")
	}
}

func TestErrorWrapperUDPConnWriteToFailure(t *testing.T) {
	expected := errors.New("mocked error")
	quc := &errorWrapperUDPConn{
		OOBCapablePacketConn: &netxmocks.QUICUDPConn{
			MockWriteTo: func(p []byte, addr net.Addr) (int, error) {
				return 0, expected
			},
		},
	}
	pkt := make([]byte, 128)
	addr := &net.UDPAddr{}
	cnt, err := quc.WriteTo(pkt, addr)
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected", err)
	}
	if cnt != 0 {
		t.Fatal("expected 0 here")
	}
}

func TestErrorWrapperUDPConnReadMsgUDPSuccess(t *testing.T) {
	expected := errors.New("mocked error")
	quc := &errorWrapperUDPConn{
		OOBCapablePacketConn: &netxmocks.QUICUDPConn{
			MockReadMsgUDP: func(b, oob []byte) (int, int, int, *net.UDPAddr, error) {
				return 0, 0, 0, nil, expected
			},
		},
	}
	b := make([]byte, 128)
	oob := make([]byte, 128)
	n, oobn, flags, addr, err := quc.ReadMsgUDP(b, oob)
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected", err)
	}
	if n != 0 {
		t.Fatal("expected 0 here")
	}
	if oobn != 0 {
		t.Fatal("expected 0 here")
	}
	if flags != 0 {
		t.Fatal("expected 0 here")
	}
	if addr != nil {
		t.Fatal("expected nil here")
	}
}

func TestErrorWrapperUDPConnReadMsgUDPFailure(t *testing.T) {
	quc := &errorWrapperUDPConn{
		OOBCapablePacketConn: &netxmocks.QUICUDPConn{
			MockReadMsgUDP: func(b, oob []byte) (int, int, int, *net.UDPAddr, error) {
				return 10, 1, 0, nil, nil
			},
		},
	}
	b := make([]byte, 128)
	oob := make([]byte, 128)
	n, oobn, flags, addr, err := quc.ReadMsgUDP(b, oob)
	if err != nil {
		t.Fatal("not the error we expected", err)
	}
	if n != 10 {
		t.Fatal("expected 10 here")
	}
	if oobn != 1 {
		t.Fatal("expected 1 here")
	}
	if flags != 0 {
		t.Fatal("expected 0 here")
	}
	if addr != nil {
		t.Fatal("expected nil here")
	}
}

func TestErrorWrapperQUICDialerFailure(t *testing.T) {
	ctx := context.Background()
	d := &ErrorWrapperQUICDialer{Dialer: &netxmocks.QUICContextDialer{
		MockDialContext: func(ctx context.Context, network, address string, tlsConfig *tls.Config, quicConfig *quic.Config) (quic.EarlySession, error) {
			return nil, io.EOF
		},
	}}
	sess, err := d.DialContext(
		ctx, "udp", "www.google.com:443", &tls.Config{}, &quic.Config{})
	if sess != nil {
		t.Fatal("expected a nil sess here")
	}
	if !errors.Is(err, io.EOF) {
		t.Fatal("expected another error here")
	}
	var errWrapper *ErrWrapper
	if !errors.As(err, &errWrapper) {
		t.Fatal("cannot cast to ErrWrapper")
	}
	if errWrapper.Operation != QUICHandshakeOperation {
		t.Fatal("unexpected Operation")
	}
	if errWrapper.Failure != FailureEOFError {
		t.Fatal("unexpected failure")
	}
}
