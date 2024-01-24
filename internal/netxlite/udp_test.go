package netxlite

import "testing"

func TestNewUDPListener(t *testing.T) {
	netx := &Netx{}
	ql := netx.NewUDPListener()
	qew := ql.(*udpListenerErrWrapper)
	_ = qew.UDPListener.(*udpListenerStdlib)
}
