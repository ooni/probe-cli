package mocks

import "testing"

func TestAddr(t *testing.T) {
	t.Run("String", func(t *testing.T) {
		a := &Addr{
			MockString: func() string {
				return "antani"
			},
		}
		if a.String() != "antani" {
			t.Fatal("invalid result for String")
		}
	})

	t.Run("Network", func(t *testing.T) {
		a := &Addr{
			MockNetwork: func() string {
				return "mascetti"
			},
		}
		if a.Network() != "mascetti" {
			t.Fatal("invalid result for Network")
		}
	})
}
