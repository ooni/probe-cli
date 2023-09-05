package testingx

import (
	"crypto/tls"
	"crypto/x509"

	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// TLSMITMProvider provides TLS MITM capabilities. Two structs are known
// to implement this interface:
//
// 1. a [*netem.UNetStack] instance.
//
// 2. the one returned by [MustNewTLSMITMProviderNetem].
//
// Both use [github.com/google/martian/v3/mitm] under the hood.
//
// Use the former when you're using netem; the latter when using the stdlib.
type TLSMITMProvider interface {
	// DefaultCertPool returns the default cert pool to use.
	DefaultCertPool() (*x509.CertPool, error)

	// ServerTLSConfig returns ready to use server TLS configuration.
	ServerTLSConfig() *tls.Config
}

var _ TLSMITMProvider = &netem.UNetStack{}

// MustNewTLSMITMProviderNetem uses [github.com/ooni/netem] to implement [TLSMITMProvider].
func MustNewTLSMITMProviderNetem() TLSMITMProvider {
	return &netemTLSMITMProvider{runtimex.Try1(netem.NewTLSMITMConfig())}
}

type netemTLSMITMProvider struct {
	cfg *netem.TLSMITMConfig
}

// DefaultCertPool implements TLSMITMProvider.
func (p *netemTLSMITMProvider) DefaultCertPool() (*x509.CertPool, error) {
	return p.cfg.CertPool()
}

// ServerTLSConfig implements TLSMITMProvider.
func (p *netemTLSMITMProvider) ServerTLSConfig() *tls.Config {
	return p.cfg.TLSConfig()
}
