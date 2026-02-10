//go:build ooni_userauth

package userauth

// #cgo CFLAGS: -I${SRCDIR}/ffi
// #cgo linux,amd64 LDFLAGS: -L${SRCDIR}/lib/linux/amd64 -looniprobe_userauth -ldl -lm -lpthread
// #cgo linux,arm64 LDFLAGS: -L${SRCDIR}/lib/linux/arm64 -looniprobe_userauth -ldl -lm -lpthread
// #cgo darwin,amd64 LDFLAGS: -L${SRCDIR}/lib/darwin/amd64 -looniprobe_userauth -framework CoreFoundation -framework Security
// #cgo darwin,arm64 LDFLAGS: -L${SRCDIR}/lib/darwin/arm64 -looniprobe_userauth -framework CoreFoundation -framework Security
// #cgo windows,amd64 LDFLAGS: -L${SRCDIR}/lib/windows/amd64 -looniprobe_userauth -lws2_32 -luserenv -lbcrypt
// #include <stdlib.h>
// #include "ooniprobe_userauth.h"
import "C"

import (
	"encoding/json"
	"errors"
	"unsafe"
)

// RegistrationResponse represents the result of user registration
type RegistrationResponse struct {
	Credential  string `json:"credential"`
	EmissionDay int16  `json:"emission_day"`
}

// SubmitResponse represents the result of submitting measurement
type SubmitResponse struct {
	MeasurementUID string `json:"measurement_uid,omitempty"`
	IsVerified     bool   `json:"is_verified"`
	SubmitResponse string `json:"submit_response"`
}

// HTTPGet performs an HTTP GET request
func HTTPGet(url string) (string, error) {
	cURL := C.CString(url)
	defer C.free(unsafe.Pointer(cURL))

	resp := C.client_get(cURL)
	defer C.client_response_free(resp)

	return parseResponse(resp)
}

// HTTPPost performs an HTTP POST request
func HTTPPost(url, payload string) (string, error) {
	cURL := C.CString(url)
	defer C.free(unsafe.Pointer(cURL))

	cPayload := C.CString(payload)
	defer C.free(unsafe.Pointer(cPayload))

	resp := C.client_post(cURL, cPayload)
	defer C.client_response_free(resp)

	return parseResponse(resp)
}

// Register registers a new user and obtains a credential
func Register(url, publicParams, manifestVersion string) (*RegistrationResponse, error) {
	cURL := C.CString(url)
	defer C.free(unsafe.Pointer(cURL))

	cPublicParams := C.CString(publicParams)
	defer C.free(unsafe.Pointer(cPublicParams))

	cManifestVersion := C.CString(manifestVersion)
	defer C.free(unsafe.Pointer(cManifestVersion))

	resp := C.userauth_register(cURL, cPublicParams, cManifestVersion)
	defer C.client_response_free(resp)

	jsonStr, err := parseResponse(resp)
	if err != nil {
		return nil, err
	}

	var result RegistrationResponse
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, errors.New("failed to parse registration response: " + err.Error())
	}

	return &result, nil
}

// Submit submits user credentials with measurement data
func Submit(url, credentialB64, publicParams, probeCC, probeASN, manifestVersion string) (*SubmitResponse, error) {
	cURL := C.CString(url)
	defer C.free(unsafe.Pointer(cURL))

	cCredentialB64 := C.CString(credentialB64)
	defer C.free(unsafe.Pointer(cCredentialB64))

	cPublicParams := C.CString(publicParams)
	defer C.free(unsafe.Pointer(cPublicParams))

	cProbeCC := C.CString(probeCC)
	defer C.free(unsafe.Pointer(cProbeCC))

	cProbeASN := C.CString(probeASN)
	defer C.free(unsafe.Pointer(cProbeASN))

	cManifestVersion := C.CString(manifestVersion)
	defer C.free(unsafe.Pointer(cManifestVersion))

	resp := C.userauth_submit(cURL, cCredentialB64, cPublicParams, cProbeCC, cProbeASN, cManifestVersion)
	defer C.client_response_free(resp)

	jsonStr, err := parseResponse(resp)
	if err != nil {
		return nil, err
	}

	var result SubmitResponse
	if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
		return nil, errors.New("failed to parse submit response: " + err.Error())
	}

	return &result, nil
}

// parseResponse converts a C ClientResponse to Go types
func parseResponse(resp C.ClientResponse) (string, error) {
	if resp.error != nil {
		errStr := C.GoString(resp.error)
		return "", errors.New(errStr)
	}

	if resp.json != nil {
		return C.GoString(resp.json), nil
	}

	return "", errors.New("empty response from FFI")
}
