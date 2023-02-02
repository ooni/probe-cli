package mocks

import (
	"net/url"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
)

func TestTunnel(t *testing.T) {
	t.Run("BootstrapTime", func(t *testing.T) {
		var expected time.Duration = 114
		tun := &Tunnel{
			MockBootstrapTime: func() time.Duration {
				return expected
			},
		}
		if tun.BootstrapTime() != expected {
			t.Fatal("invalid BootstrapTime")
		}
	})

	t.Run("SOCKS5ProxyURL", func(t *testing.T) {
		expected := &url.URL{
			Scheme:      "https",
			Opaque:      "",
			User:        &url.Userinfo{},
			Host:        "www.google.com",
			Path:        "/robots.txt",
			RawPath:     "",
			ForceQuery:  false,
			RawQuery:    "",
			Fragment:    "",
			RawFragment: "",
		}
		tun := &Tunnel{
			MockSOCKS5ProxyURL: func() *url.URL {
				return expected
			},
		}
		if diff := cmp.Diff(expected.String(), tun.SOCKS5ProxyURL().String()); diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("Stop", func(t *testing.T) {
		called := &atomic.Int64{}
		tun := &Tunnel{
			MockStop: func() {
				called.Add(1)
			},
		}
		tun.Stop()
		if called.Load() != 1 {
			t.Fatal("not called")
		}
	})
}
