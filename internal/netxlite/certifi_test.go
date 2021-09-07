package netxlite

import (
	"crypto/x509"
	"testing"
)

func TestPEMCerts(t *testing.T) {
	pool := x509.NewCertPool()
	if !pool.AppendCertsFromPEM([]byte(pemcerts)) {
		t.Fatal("cannot load pemcerts")
	}
}
