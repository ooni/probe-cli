package netem

//
// TLS: MITM configuration
//

import (
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"time"

	"github.com/google/martian/v3/mitm"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// TLSMITMConfig contains configuration for TLS MITM operations. You MUST use the
// [NewMITMConfig] factory to create a new instance. You will need to pass this
// instance to [NewGVisorStack] so that all the [GvisorStack] can communicate with
// each other using the same underlying (fake) root CA pool.
type TLSMITMConfig struct {
	// cert is the fake CA certificate for MITM.
	cert *x509.Certificate

	// config is the MITM config to generate certificates on the fly.
	config *mitm.Config

	// key is the private key that signed the mitmCert.
	key *rsa.PrivateKey
}

// NewTLSMITMConfig creates a new [MITMConfig]. This function calls
// [runtimex.PanicOnError] on failure.
func NewTLSMITMConfig() *TLSMITMConfig {
	cert, key := runtimex.Try2(mitm.NewAuthority("jafar", "OONI", 24*time.Hour))
	config := runtimex.Try1(mitm.NewConfig(cert, key))
	return &TLSMITMConfig{
		cert:   cert,
		config: config,
		key:    key,
	}
}

// CertPool returns an [x509.CertPool] using the given [MITMConfig].
func (c *TLSMITMConfig) CertPool() *x509.CertPool {
	pool := x509.NewCertPool()
	pool.AddCert(c.cert)
	return pool
}

// TLSConfig returns a *tls.Config that will generate certificates on-the-fly using
// the SNI extension in the TLS ClientHello.
func (c *TLSMITMConfig) TLSConfig() *tls.Config {
	return c.config.TLS()
}
