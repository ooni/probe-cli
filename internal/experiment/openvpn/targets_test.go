package openvpn

import (
	"testing"
)

func Test_pickFromDefaultOONIOpenVPNConfig(t *testing.T) {
	pick := pickFromDefaultOONIOpenVPNConfig()

	if pick.Cipher != "AES-256-GCM" {
		t.Fatal("cipher unexpected")
	}
	if pick.SafeCA != defaultCA {
		t.Fatal("ca unexpected")
	}
}
