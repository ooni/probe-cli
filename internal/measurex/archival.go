package measurex

import (
	"log"
	"net/http"
	"strings"
)

//
// Archival
//
// This file defines helpers to serialize to the OONI data format. Some of
// our data structure are already pretty close to the desired format, while
// other are more flat, which makes processing simpler. So, when we need
// help we use routines from this file to serialize correctly.
//

//
// BinaryData
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

//
// HTTPRoundTrip
//

// ArchivalHeadersList is a list of HTTP headers.
type ArchivalHeadersList [][]string

// Get searches for the first header with the named key
// and returns it. If not found, returns an empty string.
func (headers ArchivalHeadersList) Get(key string) string {
	key = strings.ToLower(key)
	for _, entry := range headers {
		if len(entry) != 2 {
			log.Printf("headers: malformed header: %+v", entry)
			continue
		}
		headerKey, headerValue := entry[0], entry[1]
		if strings.ToLower(headerKey) == key {
			return headerValue
		}
	}
	return ""
}

// NewArchivalHeadersList builds a new HeadersList from http.Header.
func NewArchivalHeadersList(in http.Header) (out ArchivalHeadersList) {
	for k, vv := range in {
		for _, v := range vv {
			out = append(out, []string{k, v})
		}
	}
	return
}

//
// TLSCerts
//

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
