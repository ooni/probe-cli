package wireguard

import (
	"errors"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func Test_wireguardOptions(t *testing.T) {
	t.Run("amnezia values are the expected set", func(t *testing.T) {
		wc := wireguardOptions{
			jc:   "1",
			jmin: "2",
			jmax: "3",
			s1:   "4",
			s2:   "5",
			h1:   "6",
			h2:   "7",
			h3:   "8",
			h4:   "9",
		}
		expected := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9"}
		if diff := cmp.Diff(wc.amneziaValues(), expected); diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("validate() is true when the mandatory fields are filled", func(t *testing.T) {
		wc := wireguardOptions{
			endpoint:     "1.1.1.1:8020",
			ip:           "10.1.2.8",
			pubKey:       "foobar",
			privKey:      "foobar",
			presharedKey: "foobar",
		}
		if wc.validate() != true {
			t.Fatal("expected options to be valid")
		}
	})

	t.Run("validate() is false when one mandatory field is missing", func(t *testing.T) {
		wc := wireguardOptions{
			endpoint:     "1.1.1.1:8020",
			pubKey:       "foobar",
			privKey:      "foobar",
			presharedKey: "foobar",
		}
		if wc.validate() != false {
			t.Fatal("expected options not to be valid")
		}
	})

	t.Run("validate() is true when the all amnezia fields are filled", func(t *testing.T) {
		wc := wireguardOptions{
			endpoint:     "1.1.1.1:8020",
			ip:           "10.1.2.8",
			pubKey:       "foobar",
			privKey:      "foobar",
			presharedKey: "foobar",
			jc:           "1",
			jmin:         "2",
			jmax:         "3",
			s1:           "4",
			s2:           "5",
			h1:           "6",
			h2:           "7",
			h3:           "8",
			h4:           "9",
		}
		if wc.validate() != true {
			t.Fatal("expected options to be valid")
		}
	})

	t.Run("validate() is false when any of the amnezia fields is missing", func(t *testing.T) {
		wc := wireguardOptions{
			endpoint:     "1.1.1.1:8020",
			ip:           "10.1.2.8",
			pubKey:       "foobar",
			privKey:      "foobar",
			presharedKey: "foobar",
			jc:           "1",
			jmin:         "2",
			jmax:         "3",
			s1:           "4",
			s2:           "5",
			h1:           "6",
			h2:           "7",
			h3:           "8",
			h4:           "",
		}
		if wc.validate() != false {
			t.Fatal("expected options not to be valid")
		}
	})

	t.Run("isAmneziaFlavored() is true when none of the amnezia fields is missing", func(t *testing.T) {
		wc := wireguardOptions{
			endpoint:     "1.1.1.1:8020",
			ip:           "10.1.2.8",
			pubKey:       "foobar",
			privKey:      "foobar",
			presharedKey: "foobar",
			jc:           "1",
			jmin:         "2",
			jmax:         "3",
			s1:           "4",
			s2:           "5",
			h1:           "6",
			h2:           "7",
			h3:           "8",
			h4:           "9",
		}
		if wc.isAmneziaFlavored() != true {
			t.Fatal("expected to be amnezia flavored")
		}
	})

	t.Run("configurationHash() is empty for non-amnezia values", func(t *testing.T) {
		wc := wireguardOptions{
			endpoint: "1.1.1.1:8020",
		}
		expected := ""
		if diff := cmp.Diff(wc.configurationHash(), expected); diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("get the expected configurationHash()", func(t *testing.T) {
		wc := wireguardOptions{
			endpoint: "1.1.1.1:8020",
			jc:       "1",
			jmin:     "2",
			jmax:     "3",
			s1:       "4",
			s2:       "5",
			h1:       "6",
			h2:       "7",
			h3:       "8",
			h4:       "9",
		}
		expected := "adb00b0ab179bfbdf9835bc124cbc7ab7e59bd8b"
		if diff := cmp.Diff(wc.configurationHash(), expected); diff != "" {
			t.Fatal(diff)
		}
	})
}

func Test_newWireguardOptionsFromConfig(t *testing.T) {
	t.Run("good config does not fail", func(t *testing.T) {
		c := &Config{
			SafePublicKey:    "ZGVhZGJlZWY=",
			SafePrivateKey:   "ZGVhZGJlZWY=",
			SafePresharedKey: "ZGVhZGJlZWY=",
			SafeRemote:       "1.2.3.4:8080",
		}

		opts, err := newWireguardOptionsFromConfig(c)
		if !errors.Is(err, nil) {
			t.Fatal("did not expect error")
		}

		hexExpected := "6465616462656566" // deadbeef

		if diff := cmp.Diff(opts.pubKey, hexExpected); diff != "" {
			t.Fatal(diff)
		}
		if diff := cmp.Diff(opts.privKey, hexExpected); diff != "" {
			t.Fatal(diff)
		}
		if diff := cmp.Diff(opts.presharedKey, hexExpected); diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("bad pubkey fails", func(t *testing.T) {
		c := &Config{
			SafePublicKey:    "ZGVhZGJlZWY",
			SafePrivateKey:   "ZGVhZGJlZWY=",
			SafePresharedKey: "ZGVhZGJlZWY=",
			SafeRemote:       "1.2.3.4:8080",
		}

		opts, err := newWireguardOptionsFromConfig(c)
		if opts != nil {
			t.Fatal("did not expect anything other than nil")
		}
		if !errors.Is(err, ErrInvalidInput) {
			t.Fatal("not the error we expected")
		}
	})

	t.Run("bad privkey fails", func(t *testing.T) {
		c := &Config{
			SafePublicKey:    "ZGVhZGJlZWY=",
			SafePrivateKey:   "ZGVhZGJlZWY",
			SafePresharedKey: "ZGVhZGJlZWY=",
			SafeRemote:       "1.2.3.4:8080",
		}

		opts, err := newWireguardOptionsFromConfig(c)
		if opts != nil {
			t.Fatal("did not expect anything other than nil")
		}
		if !errors.Is(err, ErrInvalidInput) {
			t.Fatal("not the error we expected")
		}
	})

	t.Run("bad preshared key fails", func(t *testing.T) {
		c := &Config{
			SafePublicKey:    "ZGVhZGJlZWY=",
			SafePrivateKey:   "ZGVhZGJlZWY=",
			SafePresharedKey: "ZGVhZGJlZWY",
			SafeRemote:       "1.2.3.4:8080",
		}

		opts, err := newWireguardOptionsFromConfig(c)
		if opts != nil {
			t.Fatal("did not expect anything other than nil")
		}
		if !errors.Is(err, ErrInvalidInput) {
			t.Fatal("not the error we expected")
		}
	})
}
