package netxlite

import "testing"

func TestNewUDPListener(t *testing.T) {
	ql := NewUDPListener()
	qew := ql.(*udpListenerErrWrapper)
	_ = qew.UDPListener.(*udpListenerStdlib)
}
