package tlsmiddlebox

import (
	"testing"
	"time"
)

func TestConfig_maxttl(t *testing.T) {
	t.Run("default maxttl", func(t *testing.T) {
		c := Config{}
		if c.maxttl() != 20 {
			t.Fatal("invalid default number of repetitions")
		}
	})

	t.Run("configured maxttl", func(t *testing.T) {
		c := Config{
			MaxTTL: 32,
		}
		if c.maxttl() != 32 {
			t.Fatal("invalid number of repetitions")
		}
	})

}

func TestConfig_delay(t *testing.T) {
	t.Run("default delay", func(t *testing.T) {
		c := Config{}
		if c.delay() != 100*time.Millisecond {
			t.Fatal("invalid default delay")
		}
	})

	t.Run("configured delay", func(t *testing.T) {
		c := Config{
			Delay: 200,
		}
		if c.delay() != 200*time.Millisecond {
			t.Fatal("invalid delay")
		}
	})

}

func TestConfig_resolver(t *testing.T) {
	t.Run("default resolver", func(t *testing.T) {
		c := Config{}
		if c.resolverURL() != "https://mozilla.cloudflare-dns.com/dns-query" {
			t.Fatal("invalid default resolver URL")
		}
	})

	t.Run("configured resolver", func(t *testing.T) {
		c := Config{
			ResolverURL: "https://cloudflare-dns.com/dns-query",
		}
		if c.resolverURL() != "https://cloudflare-dns.com/dns-query" {
			t.Fatal("invalid configured resolver URL")
		}
	})

}

func TestConfig_snipass(t *testing.T) {
	t.Run("default sni control", func(t *testing.T) {
		c := Config{}
		if c.snicontrol() != "example.com" {
			t.Fatal("invalid default pass SNI")
		}
	})

	t.Run("configured sni control", func(t *testing.T) {
		c := Config{
			SNIControl: "example.org",
		}
		if c.snicontrol() != "example.org" {
			t.Fatal("invalid configured pass SNI")
		}
	})

}

func TestConfig_testhelper(t *testing.T) {
	t.Run("without config", func(t *testing.T) {
		c := Config{}
		th, err := c.testhelper("example.com")
		if err != nil {
			t.Fatal("unexpected error")
		}
		if th.Scheme != "tlshandshake" {
			t.Fatal("unexpected scheme")
		}
		if th.Host != "example.com" {
			t.Fatal("unexpected host")
		}
	})

	t.Run("with config", func(t *testing.T) {
		c := Config{
			TestHelper: "tlshandshake://example.com:80",
		}
		th, err := c.testhelper("google.com")
		if err != nil {
			t.Fatal("unexpected error")
		}
		if th.Scheme != "tlshandshake" {
			t.Fatal("unexpected scheme")
		}
		if th.Host != "example.com:80" {
			t.Fatal("unexpected host")
		}
	})

	t.Run("failure case", func(t *testing.T) {
		c := Config{
			TestHelper: "\t",
		}
		th, _ := c.testhelper("google.com")
		if th != nil {
			t.Fatal("expected nil url")
		}
	})
}

func TestConfig_clientid(t *testing.T) {
	t.Run("default Client ID", func(t *testing.T) {
		c := Config{}
		if c.clientid() != 0 {
			t.Fatal("invalid default ClientHello ID")
		}
	})

	t.Run("configured Client ID", func(t *testing.T) {
		c := Config{
			ClientId: 6,
		}
		if c.clientid() != 6 {
			t.Fatal("invalid ClientHello ID")
		}
	})

}

func TestConfig_privacymode(t *testing.T) {

	t.Run("default Privacy Mode", func(t *testing.T) {
		c := Config{}
		if c.privacymode() != "safe" {
			t.Fatalf("invalif default Privacy Mode")
		}
	})

	t.Run("unsafe Privacy Mode", func(t *testing.T) {
		c := Config{
			PrivacyMode: "unsafe",
		}
		if c.privacymode() != "unsafe" {
			t.Fatalf("expected unsafe Privacy Mode, got %s", c.privacymode())
		}
	})

}
