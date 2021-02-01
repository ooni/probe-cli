package trace

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"net/http"
	"time"
)

// Event is one of the events within a trace
type Event struct {
	Addresses          []string            `json:",omitempty"`
	Address            string              `json:",omitempty"`
	DNSQuery           []byte              `json:",omitempty"`
	DNSReply           []byte              `json:",omitempty"`
	DataIsTruncated    bool                `json:",omitempty"`
	Data               []byte              `json:",omitempty"`
	Duration           time.Duration       `json:",omitempty"`
	Err                error               `json:",omitempty"`
	HTTPHeaders        http.Header         `json:",omitempty"`
	HTTPMethod         string              `json:",omitempty"`
	HTTPStatusCode     int                 `json:",omitempty"`
	HTTPURL            string              `json:",omitempty"`
	Hostname           string              `json:",omitempty"`
	Name               string              `json:",omitempty"`
	NoTLSVerify        bool                `json:",omitempty"`
	NumBytes           int                 `json:",omitempty"`
	Proto              string              `json:",omitempty"`
	TLSServerName      string              `json:",omitempty"`
	TLSCipherSuite     string              `json:",omitempty"`
	TLSNegotiatedProto string              `json:",omitempty"`
	TLSNextProtos      []string            `json:",omitempty"`
	TLSPeerCerts       []*x509.Certificate `json:",omitempty"`
	TLSVersion         string              `json:",omitempty"`
	Time               time.Time           `json:",omitempty"`
	Transport          string              `json:",omitempty"`
}

// PeerCerts returns the certificates presented by the peer regardless
// of whether the TLS handshake was successful
func PeerCerts(state tls.ConnectionState, err error) []*x509.Certificate {
	var x509HostnameError x509.HostnameError
	if errors.As(err, &x509HostnameError) {
		// Test case: https://wrong.host.badssl.com/
		return []*x509.Certificate{x509HostnameError.Certificate}
	}
	var x509UnknownAuthorityError x509.UnknownAuthorityError
	if errors.As(err, &x509UnknownAuthorityError) {
		// Test case: https://self-signed.badssl.com/. This error has
		// never been among the ones returned by MK.
		return []*x509.Certificate{x509UnknownAuthorityError.Cert}
	}
	var x509CertificateInvalidError x509.CertificateInvalidError
	if errors.As(err, &x509CertificateInvalidError) {
		// Test case: https://expired.badssl.com/
		return []*x509.Certificate{x509CertificateInvalidError.Cert}
	}
	return state.PeerCertificates
}
