package openvpn

import (
	"testing"

	"github.com/google/go-cmp/cmp"
)

func Test_resolveTarget(t *testing.T) {
	// TODO: mustHaveExternalNetwork() equivalent.
	if testing.Short() {
		t.Skip("skip test in short mode")
	}

	_, err := resolveTarget("google.com")

	if err != nil {
		if err.Error() == "connection_refused" {
			// connection_refused is raised when running this test
			// on the restricted network for coverage tests.
			// so we bail out
			return
		}
		t.Fatal("should be able to resolve the target")
	}

	_, err = resolveTarget("nothing.corp")
	if err == nil {
		t.Fatal("should not be able to resolve the target")
	}

	_, err = resolveTarget("asfasfasfasfasfafs.ooni.io")
	if err == nil {
		t.Fatal("should not be able to resolve the target")
	}
}

func Test_defaultOONIOpenVPNTargetUDP(t *testing.T) {
	url, err := defaultOONIOpenVPNTargetUDP()
	if err != nil {
		if err.Error() == "connection_refused" {
			// connection_refused is raised when running this test
			// on the restricted network for coverage tests.
			// so we bail out
			return
		}
		t.Fatal("unexpected error")
	}
	expected := "openvpn://oonivpn.corp/?address=37.218.243.98:1194&transport=udp"
	if diff := cmp.Diff(url, expected); diff != "" {
		t.Fatal(diff)
	}
}

func Test_defaultOONIOpenVPNTargetTCP(t *testing.T) {
	url, err := defaultOONIOpenVPNTargetTCP()
	if err != nil {
		if err.Error() == "connection_refused" {
			// connection_refused is raised when running this test
			// on the restricted network for coverage tests.
			// so we bail out
			return
		}
		t.Fatal("unexpected error")
	}
	expected := "openvpn://oonivpn.corp/?address=37.218.243.98:1194&transport=tcp"
	if diff := cmp.Diff(url, expected); diff != "" {
		t.Fatal(diff)
	}
}

func Test_pickFromDefaultOONIOpenVPNConfig(t *testing.T) {
	pick := pickFromDefaultOONIOpenVPNConfig()

	if pick.Cipher != "AES-256-GCM" {
		t.Fatal("cipher unexpected")
	}
	if pick.SafeCA != defaultCA {
		t.Fatal("ca unexpected")
	}
}
