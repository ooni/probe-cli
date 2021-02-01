package netx

import "crypto/x509"

// DefaultCertPool allows tests to access the default cert pool.
func DefaultCertPool() *x509.CertPool {
	return defaultCertPool
}
