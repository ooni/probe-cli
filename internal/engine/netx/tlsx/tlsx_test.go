package tlsx

import (
	"crypto/tls"
	"errors"
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

func TestNewDefaultCertPoolWorks(t *testing.T) {
	pool := NewDefaultCertPool()
	if pool == nil {
		t.Fatal("expected non-nil value here")
	}
}

func TestConfigureTLSVersion(t *testing.T) {
	tests := []struct {
		name       string
		version    string
		wantErr    error
		versionMin int
		versionMax int
	}{{
		name:       "with TLSv1.3",
		version:    "TLSv1.3",
		wantErr:    nil,
		versionMin: tls.VersionTLS13,
		versionMax: tls.VersionTLS13,
	}, {
		name:       "with TLSv1.2",
		version:    "TLSv1.2",
		wantErr:    nil,
		versionMin: tls.VersionTLS12,
		versionMax: tls.VersionTLS12,
	}, {
		name:       "with TLSv1.1",
		version:    "TLSv1.1",
		wantErr:    nil,
		versionMin: tls.VersionTLS11,
		versionMax: tls.VersionTLS11,
	}, {
		name:       "with TLSv1.0",
		version:    "TLSv1.0",
		wantErr:    nil,
		versionMin: tls.VersionTLS10,
		versionMax: tls.VersionTLS10,
	}, {
		name:       "with TLSv1",
		version:    "TLSv1",
		wantErr:    nil,
		versionMin: tls.VersionTLS10,
		versionMax: tls.VersionTLS10,
	}, {
		name:       "with default",
		version:    "",
		wantErr:    nil,
		versionMin: 0,
		versionMax: 0,
	}, {
		name:       "with invalid version",
		version:    "TLSv999",
		wantErr:    ErrInvalidTLSVersion,
		versionMin: 0,
		versionMax: 0,
	}}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			conf := new(tls.Config)
			err := ConfigureTLSVersion(conf, tt.version)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("not the error we expected: %+v", err)
			}
			if conf.MinVersion != uint16(tt.versionMin) {
				t.Fatalf("not the min version we expected: %+v", conf.MinVersion)
			}
			if conf.MaxVersion != uint16(tt.versionMax) {
				t.Fatalf("not the max version we expected: %+v", conf.MaxVersion)
			}
		})
	}
}
