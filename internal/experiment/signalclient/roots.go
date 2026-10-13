package signalclient

//
// Trust anchors: Signal's two roots (or the SignalCA option's), the Google roots libsignal pins
// for its Google fronts, and Mozilla's roots for the Fastly fronts.
//

import (
	"crypto/sha256"
	"crypto/x509"
	"embed"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// googleRootsFS holds the Google roots libsignal v0.105.0 pins for its Google fronts
// (rust/net/res/): GTS Root R1 to R4, GlobalSign R2 and R4, and GIAG2. GIAG2 and GlobalSign R2
// have expired; Go's verifier does not chain to them, but they are kept as libsignal ships them.
//
//go:embed roots/*.cer
var googleRootsFS embed.FS

// trustAnchors is a pool and the SHA-256 of each certificate in it (lowercase hex of the DER).
type trustAnchors struct {
	pool   *x509.CertPool
	sha256 []string
}

func newTrustAnchors(certs []*x509.Certificate) *trustAnchors {
	ta := &trustAnchors{pool: x509.NewCertPool()}
	for _, c := range certs {
		ta.pool.AddCert(c)
		sum := sha256.Sum256(c.Raw)
		ta.sha256 = append(ta.sha256, hex.EncodeToString(sum[:]))
	}
	return ta
}

// signalRoots returns Signal's two roots (RSA and Ed25519), the only trust anchors Signal's
// clients accept for Signal-hosted services.
func signalRoots() ([]*x509.Certificate, error) {
	var out []*x509.Certificate
	for _, der := range [][]byte{signalRSARootDER, signalEd25519RootDER} {
		cert, err := x509.ParseCertificate(der)
		if err != nil {
			return nil, fmt.Errorf("signal_client: parsing Signal root: %w", err)
		}
		out = append(out, cert)
	}
	return out, nil
}

// errNoCertificates means a SignalCA bundle contained no certificate.
var errNoCertificates = errors.New("signal_client: SignalCA contains no certificate")

// parsePEMBundle parses the certificates of a PEM bundle (the SignalCA option). Blocks other
// than CERTIFICATE are an error, so that a key pasted by mistake is not silently ignored.
func parsePEMBundle(bundle string) ([]*x509.Certificate, error) {
	var out []*x509.Certificate
	rest := []byte(bundle)
	for {
		var block *pem.Block
		block, rest = pem.Decode(rest)
		if block == nil {
			break
		}
		if block.Type != "CERTIFICATE" {
			return nil, fmt.Errorf("signal_client: SignalCA: unexpected PEM block %q", block.Type)
		}
		cert, err := x509.ParseCertificate(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("signal_client: SignalCA: %w", err)
		}
		out = append(out, cert)
	}
	if len(out) == 0 {
		return nil, errNoCertificates
	}
	return out, nil
}

// googleRoots returns the pool for the Google fronts.
func googleRoots() (*x509.CertPool, error) {
	entries, err := googleRootsFS.ReadDir("roots")
	if err != nil {
		return nil, err
	}
	pool := x509.NewCertPool()
	for _, e := range entries {
		der, err := googleRootsFS.ReadFile("roots/" + e.Name())
		if err != nil {
			return nil, err
		}
		cert, err := x509.ParseCertificate(der)
		if err != nil {
			return nil, fmt.Errorf("signal_client: parsing %s: %w", e.Name(), err)
		}
		pool.AddCert(cert)
	}
	return pool, nil
}

// mozillaRoots returns the pool for the Fastly fronts: Mozilla's roots, or, under netem, the
// simulated network's CA.
func mozillaRoots() (*x509.CertPool, error) {
	return (&netxlite.Netx{}).MaybeCustomUnderlyingNetwork().Get().DefaultCertPool(), nil
}

// signalAnchors returns Signal's roots, or SignalCA's certificates when set.
func (m *Measurer) signalAnchors() (*trustAnchors, error) {
	if m.config.SignalCA != "" {
		certs, err := parsePEMBundle(m.config.SignalCA)
		if err != nil {
			return nil, err
		}
		return newTrustAnchors(certs), nil
	}
	certs, err := m.roots()
	if err != nil {
		return nil, err
	}
	return newTrustAnchors(certs), nil
}
