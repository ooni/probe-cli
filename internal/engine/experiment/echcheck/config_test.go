package echcheck

import (
	"testing"
)

func TestConfig(t *testing.T) {

	c := Config{
		ResolverURL: "",
	}
	s1 := c.resolverURL()
	if s1 != defaultResolver {
		t.Fatalf("expected: %s, got %s", defaultResolver, s1)
	}

	testResolver := "testResolver"

	c = Config{
		ResolverURL: testResolver,
	}
	s1 = c.resolverURL()
	if s1 != testResolver {
		t.Fatalf("expected: %s, got %s", testResolver, s1)
	}
}
