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

func TestConfig_sni(t *testing.T) {
	t.Run("without config", func(t *testing.T) {
		c := Config{}
		if c.sni("example.com") != "example.com" {
			t.Fatal("invalid sni")
		}
	})
	t.Run("with config", func(t *testing.T) {
		c := Config{
			SNI: "google.com",
		}
		if c.sni("example.com") != "google.com" {
			t.Fatal("invalid sni")
		}
	})
}

func TestConfig_clientid(t *testing.T) {
	c := Config{}
	if c.clientid() != 0 {
		t.Fatal("invalid default ClientHello ID")
	}
}
