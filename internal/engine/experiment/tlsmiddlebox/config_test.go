package tlsmiddlebox

import (
	"testing"
	"time"
)

func TestConfig_maxttl(t *testing.T) {
	c := Config{}
	if c.maxttl() != 20 {
		t.Fatal("invalid default number of repetitions")
	}
}

func TestConfig_delay(t *testing.T) {
	c := Config{}
	if c.delay() != 100*time.Millisecond {
		t.Fatal("invalid default delay")
	}
}

func TestConfig_resolver(t *testing.T) {
	c := Config{}
	if c.resolverURL() != "https://mozilla.cloudflare-dns.com/dns-query" {
		t.Fatal("invalid resolver URL")
	}
}

func TestConfig_snipass(t *testing.T) {
	c := Config{}
	if c.snicontrol() != "example.com" {
		t.Fatal("invalid pass SNI")
	}
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
	c := Config{}
	if c.clientid() != 0 {
		t.Fatal("invalid default ClientHello ID")
	}
}
