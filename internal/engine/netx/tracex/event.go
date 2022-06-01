package tracex

import (
	"crypto/x509"
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
