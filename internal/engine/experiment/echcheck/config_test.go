package echcheck

import (
	"testing"
)

const (
	defualtResolver = "https://mozilla.cloudflare-dns.com/dns-query"
	defaultSNI      = "google.com"
)

func TestConfig(t *testing.T) {

	c := Config{
		ResolverURL: "",
	}
	s1 := c.resolverURL()
	if s1 != defualtResolver {
		t.Fatalf("expected: %s, got %s", defualtResolver, s1)
	}

	testResover := "testResolver"

	c = Config{
		ResolverURL: testResover,
	}
	s1 = c.resolverURL()
	if s1 != testResover {
		t.Fatalf("expected: %s, got %s", testResover, s1)
	}
}
