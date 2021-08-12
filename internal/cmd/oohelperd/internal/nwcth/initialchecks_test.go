package nwcth

import (
	"testing"
)

func TestMeasureWithInvalidURL(t *testing.T) {
	_, err := InitialChecks("http://[::1]aaaa")

	if err == nil || err != ErrInvalidURL {
		t.Fatal("expected an error here")
	}
}

func TestMeasureWithUnsupportedScheme(t *testing.T) {
	_, err := InitialChecks("abc://example.com")

	if err == nil || err != ErrUnsupportedScheme {
		t.Fatal("expected an error here")
	}
}

func TestMeasureWithInvalidHost(t *testing.T) {
	_, err := InitialChecks("http://www.ooni.ooni")

	if err == nil || err != ErrNoSuchHost {
		t.Fatal("expected an error here")
	}
}
