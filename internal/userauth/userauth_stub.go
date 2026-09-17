//go:build nouserauth || !((cgo && linux && !android && (amd64 || arm64 || arm || 386)) || (cgo && darwin && !ios && (amd64 || arm64)) || (cgo && windows && (amd64 || 386)))

package userauth

// This file provides pure-Go stubs for builds without cgo (CGO_ENABLED=0) and
// for builds using the "nouserauth" tag.

// Register is unavailable without cgo.
func Register(url, publicParams, manifestVersion, proxy, userAgent string, timeout float32) (string, error) {
	return "", ErrUnavailable
}

// Submit is unavailable without cgo.
func Submit(url, content, probeCC, probeASN, proxy, userAgent string, timeout float32,
	cfg *CredentialConfig) (RotatedCredential, error) {
	return RotatedCredential{}, ErrUnavailable
}

// ProbeID is unavailable without cgo.
func ProbeID(credentialB64, probeASN, probeCC string) (string, error) {
	return "", ErrUnavailable
}
