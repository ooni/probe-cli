package main

//
// certs: local test certificates shaped like Signal's production chain (measured 2026-10 with
// tlsprofile.sh): two root CAs like Signal's (RSA 4096 and Ed25519), and leaves like production's,
// with an empty subject, extendedKeyUsage serverAuth, critical basicConstraints CA:FALSE, subject
// and authority key identifiers, and a critical SAN (Go writes them in production's order).
//
//	chat.signal.org:      an Ed25519 leaf and an RSA 2048 leaf, both signed (sha256WithRSA) by the
//	                      RSA CA; the server picks one by the client's signature algorithms, as
//	                      production does.
//	grpc.chat.signal.org: an Ed25519 leaf signed by the Ed25519 CA.
//
// The CAs are named as test CAs, not as Signal. Keys are written as PKCS#8.
//

import (
	"crypto"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1" // #nosec G505 -- the key identifier method of RFC 5280, section 4.2.1.2
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/asn1"
	"encoding/pem"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"
)

const certDays = 30

type certKey struct {
	cert *x509.Certificate
	key  crypto.Signer
}

func pemBlock(typ string, der []byte) []byte {
	return pem.EncodeToMemory(&pem.Block{Type: typ, Bytes: der})
}

// newCertificate signs template with the issuer's key (self-signed when issuer is nil).
func newCertificate(template *x509.Certificate, key crypto.Signer, issuer *certKey) (*certKey, error) {
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 159))
	if err != nil {
		return nil, err
	}
	pub, err := x509.MarshalPKIXPublicKey(key.Public())
	if err != nil {
		return nil, err
	}
	var spki struct {
		Algorithm pkix.AlgorithmIdentifier
		PublicKey asn1.BitString
	}
	if _, err := asn1.Unmarshal(pub, &spki); err != nil {
		return nil, err
	}
	keyID := sha1.Sum(spki.PublicKey.Bytes) // #nosec G401 -- an identifier, not a signature
	now := time.Now()
	template.SerialNumber = serial
	template.NotBefore, template.NotAfter = now.Add(-5*time.Minute), now.Add(certDays*24*time.Hour)
	template.SubjectKeyId = keyID[:]
	parent, signer := template, key
	if issuer != nil {
		parent, signer = issuer.cert, issuer.key
	}
	der, err := x509.CreateCertificate(rand.Reader, template, parent, key.Public(), signer)
	if err != nil {
		return nil, err
	}
	cert, err := x509.ParseCertificate(der)
	if err != nil {
		return nil, err
	}
	return &certKey{cert, key}, nil
}

func newCA(cn string, key crypto.Signer) (*certKey, error) {
	return newCertificate(&x509.Certificate{
		Subject:               pkix.Name{Organization: []string{"OONI signal_client test"}, CommonName: cn},
		BasicConstraintsValid: true,
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageDigitalSignature | x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
	}, key, nil)
}

func newLeaf(dns string, key crypto.Signer, ca *certKey) (*certKey, error) {
	return newCertificate(&x509.Certificate{
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		DNSNames:              []string{dns}, // critical, since the subject is empty
	}, key, ca)
}

func writeCerts(out string) error {
	write := func(name string, data ...[]byte) error {
		return os.WriteFile(filepath.Join(out, name), slices.Concat(data...), 0o600)
	}
	writeKey := func(name string, key crypto.Signer) error {
		der, err := x509.MarshalPKCS8PrivateKey(key)
		if err != nil {
			return err
		}
		return write(name, pemBlock("PRIVATE KEY", der))
	}
	certPEM := func(c *certKey) []byte { return pemBlock("CERTIFICATE", c.cert.Raw) }
	if err := os.MkdirAll(out, 0o750); err != nil {
		return err
	}

	rsaCAKey, err := rsa.GenerateKey(rand.Reader, 4096)
	if err != nil {
		return err
	}
	_, edCAKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}
	_, chatEdKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}
	chatRSAKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return err
	}
	_, grpcEdKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return err
	}

	rsaCA, err := newCA("signal_client test CA (RSA)", rsaCAKey)
	if err != nil {
		return err
	}
	edCA, err := newCA("signal_client test CA (Ed25519)", edCAKey)
	if err != nil {
		return err
	}
	chatEd, err := newLeaf("chat.signal.org", chatEdKey, rsaCA)
	if err != nil {
		return err
	}
	chatRSA, err := newLeaf("chat.signal.org", chatRSAKey, rsaCA)
	if err != nil {
		return err
	}
	grpcEd, err := newLeaf("grpc.chat.signal.org", grpcEdKey, edCA)
	if err != nil {
		return err
	}

	for _, err := range []error{
		write("ca.pem", certPEM(rsaCA), certPEM(edCA)), // trust anchors for the tests
		writeKey("chat-ed25519.key", chatEdKey),
		write("chat-ed25519.pem", certPEM(chatEd)),
		writeKey("chat-rsa.key", chatRSAKey),
		write("chat-rsa.pem", certPEM(chatRSA)),
		writeKey("grpc-ed25519.key", grpcEdKey),
		write("grpc-ed25519.pem", certPEM(grpcEd)),
	} {
		if err != nil {
			return err
		}
	}
	entries, err := os.ReadDir(out)
	if err != nil {
		return err
	}
	var names []string
	for _, e := range entries {
		names = append(names, e.Name())
	}
	fmt.Println("wrote", strings.Join(names, ", "))
	return nil
}
