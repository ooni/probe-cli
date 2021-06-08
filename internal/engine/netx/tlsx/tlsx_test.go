package tlsx

import (
	"crypto/tls"
	"testing"
)

func TestVersionString(t *testing.T) {
	if VersionString(tls.VersionTLS13) != "TLSv1.3" {
		t.Fatal("not working for existing version")
	}
	if VersionString(1) != "TLS_VERSION_UNKNOWN_1" {
		t.Fatal("not working for nonexisting version")
	}
	if VersionString(0) != "" {
		t.Fatal("not working for zero version")
	}
}

func TestCipherSuite(t *testing.T) {
	if CipherSuiteString(tls.TLS_AES_128_GCM_SHA256) != "TLS_AES_128_GCM_SHA256" {
		t.Fatal("not working for existing cipher suite")
	}
	if CipherSuiteString(1) != "TLS_CIPHER_SUITE_UNKNOWN_1" {
		t.Fatal("not working for nonexisting cipher suite")
	}
	if CipherSuiteString(0) != "" {
		t.Fatal("not working for zero cipher suite")
	}
}
