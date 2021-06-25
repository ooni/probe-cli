package netxmocks

import (
	"errors"
	"net"
	"testing"
)

func TestQUICListenerListen(t *testing.T) {
	expected := errors.New("mocked error")
	ql := &QUICListener{
		MockListen: func(addr *net.UDPAddr) (net.PacketConn, error) {
			return nil, expected
		},
	}
	pconn, err := ql.Listen(&net.UDPAddr{})
	if !errors.Is(err, expected) {
		t.Fatal("not the error we expected", expected)
	}
	if pconn != nil {
		t.Fatal("expected nil conn here")
	}
}
