//go:build !cgo

package userauth

// This file provides pure-Go stubs for builds without cgo (CGO_ENABLED=0)

// Register is unavailable without cgo.
func Register(url, publicParams, manifestVersion, proxy string, timeout float32) (string, error) {
	return "", ErrUnavailable
}

// Submit is unavailable without cgo.
func Submit(url, content, probeCC, probeASN, proxy string, timeout float32,
	cfg *CredentialConfig) (RotatedCredential, error) {
	return RotatedCredential{}, ErrUnavailable
}

// ProbeID is unavailable without cgo.
func ProbeID(credentialB64, probeASN, probeCC string) (string, error) {
	return "", ErrUnavailable
}
