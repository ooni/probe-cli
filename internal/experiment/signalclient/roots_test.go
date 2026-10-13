package signalclient

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"testing"
	"time"
)

func TestSignalRoots(t *testing.T) {
	certs, err := signalRoots()
	if err != nil {
		t.Fatal(err)
	}
	anchors := newTrustAnchors(certs)
	pool := anchors.pool
	// the fingerprints the spec lists (ts-041 Trust)
	want := []string{"ddb0f92bb95c8d6fd202ea6e8cc5ccd182b544f8cd696f47d580659ddc9df65a",
		"2642f1f90b389fed2558d7324ca26d70e750b06f87c595c5de955e2560821657"}
	if fmt.Sprint(anchors.sha256) != fmt.Sprint(want) {
		t.Errorf("signal_roots %v, want %v", anchors.sha256, want)
	}
	for _, der := range [][]byte{signalRSARootDER, signalEd25519RootDER} {
		cert, err := x509.ParseCertificate(der)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := cert.Verify(x509.VerifyOptions{Roots: pool}); err != nil {
			t.Fatalf("%s does not verify against the pool: %v", cert.Subject, err)
		}
	}
}

func TestParsePEMBundle(t *testing.T) {
	if _, err := parsePEMBundle("not pem"); err == nil {
		t.Error("accepted a bundle without certificates")
	}
	if _, err := parsePEMBundle("-----BEGIN PRIVATE KEY-----\nAAAA\n-----END PRIVATE KEY-----\n"); err == nil {
		t.Error("accepted a key")
	}
	certs, err := parsePEMBundle(otherRoot(t) + otherRoot(t))
	if err != nil || len(certs) != 2 {
		t.Errorf("%d %v", len(certs), err)
	}
	if len(newTrustAnchors(certs).sha256[0]) != 64 {
		t.Error("not a hex SHA-256")
	}
}

func TestGoogleRoots(t *testing.T) {
	pool, err := googleRoots()
	if err != nil {
		t.Fatal(err)
	}
	//lint:ignore SA1019 counting the embedded roots
	if n := len(pool.Subjects()); n != 7 {
		t.Errorf("%d Google roots", n)
	}
}

// mustSelfSigned returns a self-signed ECDSA certificate for name and its key, PEM-encoded. The
// certificate can sign, so it is also a root unrelated to the simulated network's CA.
func mustSelfSigned(t *testing.T, name string) (certPEM, keyPEM []byte) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	tmpl := &x509.Certificate{SerialNumber: big.NewInt(1), Subject: pkix.Name{CommonName: name},
		DNSNames: []string{name}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour),
		IsCA: true, BasicConstraintsValid: true, KeyUsage: x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	kder, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}),
		pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: kder})
}

// otherRoot is mustSelfSigned's certificate, for SignalCA.
func otherRoot(t *testing.T) string {
	t.Helper()
	cert, _ := mustSelfSigned(t, "example.com")
	return string(cert)
}
