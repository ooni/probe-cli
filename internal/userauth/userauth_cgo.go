//go:build (cgo && linux && !android && (amd64 || arm64 || arm || 386)) || (cgo && darwin && !ios && (amd64 || arm64)) || (cgo && windows && (amd64 || 386))

package userauth

// #cgo CFLAGS: -I${SRCDIR}/lib/include
// #cgo linux,amd64 LDFLAGS: -L${SRCDIR}/lib/linux/x86_64 -luniffi_ooniprobe -ldl -lm -lpthread
// #cgo linux,arm64 LDFLAGS: -L${SRCDIR}/lib/linux/aarch64 -luniffi_ooniprobe -ldl -lm -lpthread
// #cgo linux,arm LDFLAGS: -L${SRCDIR}/lib/linux/arm -luniffi_ooniprobe -ldl -lm -lpthread
// #cgo linux,386 LDFLAGS: -L${SRCDIR}/lib/linux/x86 -luniffi_ooniprobe -ldl -lm -lpthread
// #cgo darwin,amd64 LDFLAGS: -L${SRCDIR}/lib/macos/x86_64 -luniffi_ooniprobe -framework CoreFoundation -framework Security
// #cgo darwin,arm64 LDFLAGS: -L${SRCDIR}/lib/macos/aarch64 -luniffi_ooniprobe -framework CoreFoundation -framework Security
// #cgo windows,amd64 LDFLAGS: -L${SRCDIR}/lib/windows/x86_64 -luniffi_ooniprobe -lws2_32 -luserenv -lbcrypt
// #cgo windows,386 LDFLAGS: -L${SRCDIR}/lib/windows/x86 -luniffi_ooniprobe -lws2_32 -luserenv -lbcrypt
// #include <stdlib.h>
// #include "ooniprobe_userauth.h"
import "C"

import (
	"encoding/json"
	"errors"
	"fmt"
	"unsafe"
)

// registerResult mirrors the JSON returned by the C `userauth_register`.
type registerResult struct {
	Credential string `json:"credential"`
	StatusCode int    `json:"status_code"`
	Body       string `json:"body"`
}

// submitResult mirrors the JSON returned by the C `userauth_submit`. The
// measurement UID is carried inside the raw collector response body.
type submitResult struct {
	Credential string `json:"credential"`
	StatusCode int    `json:"status_code"`
	Body       string `json:"body"`
}

// submitBody is the subset of the collector response body we care about.
type submitBody struct {
	MeasurementUID string `json:"measurement_uid"`
}

func httpStatusOK(code int) bool {
	return code >= 200 && code < 300
}

// optionalCString returns a C string for s, or nil when s is empty.
// NOTE: The caller must free a non-nil return with C.free.
func optionalCString(s string) *C.char {
	if s == "" {
		return nil
	}
	return C.CString(s)
}

// parseResponse converts a C ClientResponse into a Go (string, error).
func parseResponse(resp C.ClientResponse) (string, error) {
	if resp.error != nil {
		return "", errors.New(C.GoString(resp.error))
	}
	if resp.json != nil {
		return C.GoString(resp.json), nil
	}
	return "", errors.New("userauth: empty response from FFI")
}

// Register registers a new user and returns the base64-encoded credential.
func Register(url, publicParams, manifestVersion, proxy, userAgent string, timeout float32) (string, error) {
	cURL := C.CString(url)
	defer C.free(unsafe.Pointer(cURL))

	cPublicParams := C.CString(publicParams)
	defer C.free(unsafe.Pointer(cPublicParams))

	cManifestVersion := C.CString(manifestVersion)
	defer C.free(unsafe.Pointer(cManifestVersion))

	cProxy := optionalCString(proxy)
	if cProxy != nil {
		defer C.free(unsafe.Pointer(cProxy))
	}

	cUserAgent := optionalCString(userAgent)
	if cUserAgent != nil {
		defer C.free(unsafe.Pointer(cUserAgent))
	}

	resp := C.userauth_register(cURL, cPublicParams, cManifestVersion, cProxy, C.float(timeout), cUserAgent)
	defer C.client_response_free(resp)

	jsonStr, err := parseResponse(resp)
	if err != nil {
		return "", err
	}

	var result registerResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return "", fmt.Errorf("userauth: cannot parse registration response: %w", err)
	}
	if !httpStatusOK(result.StatusCode) || result.Credential == "" {
		return "", fmt.Errorf("userauth: registration failed (status %d): %s",
			result.StatusCode, result.Body)
	}
	return result.Credential, nil
}

// Submit submits a measurement.
func Submit(url, content, probeCC, probeASN, proxy, userAgent string, timeout float32,
	cfg *CredentialConfig) (RotatedCredential, error) {
	cURL := C.CString(url)
	defer C.free(unsafe.Pointer(cURL))

	cContent := C.CString(content)
	defer C.free(unsafe.Pointer(cContent))

	cProbeCC := C.CString(probeCC)
	defer C.free(unsafe.Pointer(cProbeCC))

	cProbeASN := C.CString(probeASN)
	defer C.free(unsafe.Pointer(cProbeASN))

	cProxy := optionalCString(proxy)
	if cProxy != nil {
		defer C.free(unsafe.Pointer(cProxy))
	}

	cUserAgent := optionalCString(userAgent)
	if cUserAgent != nil {
		defer C.free(unsafe.Pointer(cUserAgent))
	}

	// A nil cConfig tells the FFI to use the anonymous submission path.
	var cConfig *C.char
	if cfg != nil {
		raw, err := json.Marshal(cfg)
		if err != nil {
			return RotatedCredential{}, fmt.Errorf("userauth: cannot marshal credential config: %w", err)
		}
		cConfig = C.CString(string(raw))
		defer C.free(unsafe.Pointer(cConfig))
	}

	resp := C.userauth_submit(cURL, cContent, cProbeCC, cProbeASN, cProxy, C.float(timeout), cUserAgent, cConfig)
	defer C.client_response_free(resp)

	jsonStr, err := parseResponse(resp)
	if err != nil {
		return RotatedCredential{}, err
	}

	var result submitResult
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return RotatedCredential{}, fmt.Errorf("userauth: cannot parse submit response: %w", err)
	}
	if !httpStatusOK(result.StatusCode) {
		return RotatedCredential{}, fmt.Errorf("userauth: submission failed (status %d): %s",
			result.StatusCode, result.Body)
	}
	// For an authenticated submission we expect a rotated credential back.
	if cfg != nil && result.Credential == "" {
		return RotatedCredential{}, fmt.Errorf("userauth: authenticated submission returned no credential: %s",
			result.Body)
	}

	// The measurement UID lives in the collector response body; a parse failure
	// here is non-fatal (the submission itself already succeeded).
	var body submitBody
	_ = json.Unmarshal([]byte(result.Body), &body)

	return RotatedCredential{
		Credential:     result.Credential,
		MeasurementUID: body.MeasurementUID,
	}, nil
}

// ProbeID derives the hex-encoded probe id for the given credential.
func ProbeID(credentialB64, probeASN, probeCC string) (string, error) {
	cCredential := C.CString(credentialB64)
	defer C.free(unsafe.Pointer(cCredential))

	cProbeASN := C.CString(probeASN)
	defer C.free(unsafe.Pointer(cProbeASN))

	cProbeCC := C.CString(probeCC)
	defer C.free(unsafe.Pointer(cProbeCC))

	resp := C.get_probe_id(cCredential, cProbeASN, cProbeCC)
	defer C.client_response_free(resp)

	jsonStr, err := parseResponse(resp)
	if err != nil {
		return "", err
	}

	var result struct {
		ProbeID string `json:"probe_id"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return "", fmt.Errorf("userauth: cannot parse probe id response: %w", err)
	}
	return result.ProbeID, nil
}
