package measurex

import (
	"net/http"
	"strings"
)

//
// Archival
//
// This file defines helpers to serialize to the OONI data format.
//

// ArchivalBinaryData is the archival format for binary data.
type ArchivalBinaryData struct {
	Data   []byte `json:"data"`
	Format string `json:"format"`
}

// NewArchivalBinaryData builds a new ArchivalBinaryData
// from an array of bytes. If the array is nil, we return nil.
func NewArchivalBinaryData(data []byte) (out *ArchivalBinaryData) {
	if len(data) > 0 {
		out = &ArchivalBinaryData{
			Data:   data,
			Format: "base64",
		}
	}
	return
}

// ArchivalHeaders is a list of HTTP headers.
type ArchivalHeaders map[string]string

// Get searches for the first header with the named key
// and returns it. If not found, returns an empty string.
func (headers ArchivalHeaders) Get(key string) string {
	return headers[strings.ToLower(key)]
}

// NewArchivalHeaders builds a new HeadersList from http.Header.
func NewArchivalHeaders(in http.Header) (out ArchivalHeaders) {
	out = make(ArchivalHeaders)
	for k, vv := range in {
		for _, v := range vv {
			// It breaks my hearth a little bit to ignore
			// subsequent headers, but this does not happen
			// very frequently, and I know the pipeline
			// parses the map headers format only.
			out[strings.ToLower(k)] = v
			break
		}
	}
	return
}

// NewArchivalTLSCertList builds a new []ArchivalBinaryData
// from a list of raw x509 certificates data.
func NewArchivalTLSCerts(in [][]byte) (out []*ArchivalBinaryData) {
	for _, cert := range in {
		out = append(out, &ArchivalBinaryData{
			Data:   cert,
			Format: "base64",
		})
	}
	return
}

// NewArchivalFailure creates an archival failure from an error. We
// cannot round trip an error using JSON, so we serialize to this
// intermediate format that is a sort of Optional<string>.
func NewArchivalFailure(err error) *string {
	if err == nil {
		return nil
	}
	s := err.Error()
	return &s
}
