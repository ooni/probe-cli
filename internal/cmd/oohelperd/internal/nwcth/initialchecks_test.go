package nwcth

import (
	"testing"
)

var checker = &defaultInitChecker{}

func TestMeasureWithInvalidURL(t *testing.T) {
	_, err := checker.InitialChecks("http://[::1]aaaa")

	if err == nil || err != ErrInvalidURL {
		t.Fatal("expected an error here")
	}
}

func TestMeasureWithUnsupportedScheme(t *testing.T) {
	_, err := checker.InitialChecks("abc://example.com")

	if err == nil || err != ErrUnsupportedScheme {
		t.Fatal("expected an error here")
	}
}

func TestMeasureWithInvalidHost(t *testing.T) {
	_, err := checker.InitialChecks("http://www.ooni.ooni")

	if err == nil || err != ErrNoSuchHost {
		t.Fatal("expected an error here")
	}
}
