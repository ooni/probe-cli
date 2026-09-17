package userauth

import "errors"

// ErrUnavailable
var ErrUnavailable = errors.New("userauth: anonymous-credential FFI unavailable in this build")

// ParamRange
type ParamRange struct {
	Min uint32 `json:"min"`
	Max uint32 `json:"max"`
}

// CredentialConfig carries the stored credential together with the
// manifest-derived parameters
type CredentialConfig struct {
	// Credential is the base64-encoded credential to authenticate with.
	Credential string `json:"credential"`

	// PublicParams is the base64-encoded public parameters from the manifest.
	PublicParams string `json:"public_params"`

	// ManifestVersion is the manifest version the credential belongs to.
	ManifestVersion string `json:"manifest_version"`

	// AgeRange is the allowed credential age window.
	AgeRange ParamRange `json:"age_range"`

	// MeasurementCountRange is the allowed measurement-count window.
	MeasurementCountRange ParamRange `json:"measurement_count_range"`
}

// RotatedCredential is the outcome of a successful authenticated submission.
type RotatedCredential struct {
	// Credential is the updated credential to persist for the next submission.
	Credential string

	// MeasurementUID is the collector-assigned measurement identifier.
	MeasurementUID string
}
