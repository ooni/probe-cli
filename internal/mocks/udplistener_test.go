package mocks

import (
	"errors"
	"net"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestUDPListener(t *testing.T) {
	t.Run("Listen", func(t *testing.T) {
		expected := errors.New("mocked error")
		ql := &UDPListener{
			MockListen: func(addr *net.UDPAddr) (model.UDPLikeConn, error) {
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
	})
}
