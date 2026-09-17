package userauth

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/ooni/probe-cli/v3/internal/model"
)

const (
	signCredentialPath    = "/api/v1/sign_credential"
	submitMeasurementPath = "/api/v1/submit_measurement"
)

// CredentialSubmitter implements [model.Submitter].
type CredentialSubmitter struct {
	registerURL           string
	submitURL             string
	probeCC               string
	probeASN              string
	publicParams          string
	manifestVersion       string
	ageRange              ParamRange
	measurementCountRange ParamRange
	proxy                 string
	timeout               float32
	store                 *CredStore
	logger                model.Logger
	fallback              model.Submitter
	userAgent             string
}

var _ model.Submitter = &CredentialSubmitter{}

// CredentialSubmitterConfig carries the dependencies for [NewCredentialSubmitter].
type CredentialSubmitterConfig struct {
	// BaseURL is the backend base URL (e.g. https://api.ooni.io). The register
	// and submit endpoints are resolved relative to it.
	BaseURL string

	// ProbeCC and ProbeASN identify the probe (the latter in "ASxxxx" form).
	ProbeCC  string
	ProbeASN string

	// PublicParams, ManifestVersion, AgeRange, MeasurementCountRange come from
	// the manifest and its matched submission policy.
	PublicParams          string
	ManifestVersion       string
	AgeRange              ParamRange
	MeasurementCountRange ParamRange

	// Proxy is the proxy URL to route FFI requests through ("" means none).
	Proxy string

	// Timeout is the per-request timeout in seconds (<= 0 means the default).
	Timeout float32

	// Store persists the credential across runs.
	Store *CredStore

	// Logger is the logger to use.
	Logger model.Logger

	// Fallback is the legacy submitter used when credential submission fails.
	Fallback model.Submitter

	// UserAgent is the user-agent string used with by the http client
	UserAgent string
}

// NewCredentialSubmitter builds a [CredentialSubmitter], registering for a fresh
// credential when there is no usable stored one.
func NewCredentialSubmitter(
	ctx context.Context, config CredentialSubmitterConfig) (*CredentialSubmitter, error) {
	baseURL := strings.TrimRight(config.BaseURL, "/")
	cs := &CredentialSubmitter{
		registerURL:           baseURL + signCredentialPath,
		submitURL:             baseURL + submitMeasurementPath,
		probeCC:               config.ProbeCC,
		probeASN:              config.ProbeASN,
		publicParams:          config.PublicParams,
		manifestVersion:       config.ManifestVersion,
		ageRange:              config.AgeRange,
		measurementCountRange: config.MeasurementCountRange,
		proxy:                 config.Proxy,
		timeout:               config.Timeout,
		store:                 config.Store,
		logger:                config.Logger,
		fallback:              config.Fallback,
		userAgent:             config.UserAgent,
	}

	// Reuse the stored credential when it matches the current manifest version;
	// otherwise register for a new one.
	stored := cs.store.Get()
	if stored.Credential != "" && stored.ManifestVersion == cs.manifestVersion {
		return cs, nil
	}

	cs.logger.Info("userauth: registering for a new anonymous credential")
	credential, err := Register(cs.registerURL, cs.publicParams, cs.manifestVersion, cs.proxy, cs.userAgent, cs.timeout)
	if err != nil {
		return nil, err
	}
	if err := cs.store.Set(StoredCredential{
		Credential:      credential,
		ManifestVersion: cs.manifestVersion,
		PublicParams:    cs.publicParams,
	}); err != nil {
		return nil, err
	}
	return cs, nil
}

// submitWithCredential performs the authenticated submission and credential
// rotation. It returns the measurement UID on success.
func (cs *CredentialSubmitter) submitWithCredential(m *model.Measurement) (string, error) {
	stored := cs.store.Get()
	if stored.Credential == "" {
		return "", errors.New("userauth: no stored credential")
	}

	// Stamp the measurement with the probe id derived from the credential, so the
	// uploaded measurement carries the anonymous identity.
	probeID, err := ProbeID(stored.Credential, cs.probeASN, cs.probeCC)
	if err != nil {
		return "", err
	}
	m.ProbeID = probeID

	content, err := json.Marshal(m)
	if err != nil {
		return "", err
	}

	config := &CredentialConfig{
		Credential:            stored.Credential,
		PublicParams:          cs.publicParams,
		ManifestVersion:       cs.manifestVersion,
		AgeRange:              cs.ageRange,
		MeasurementCountRange: cs.measurementCountRange,
	}

	result, err := Submit(cs.submitURL, string(content), cs.probeCC, cs.probeASN,
		cs.proxy, cs.userAgent, cs.timeout, config)
	if err != nil {
		return "", err
	}

	// Persist the rotated credential for the next submission.
	if err := cs.store.Set(StoredCredential{
		Credential:      result.Credential,
		ManifestVersion: cs.manifestVersion,
		PublicParams:    cs.publicParams,
	}); err != nil {
		return "", err
	}

	cs.logger.Infof("Measurement URL: https://explorer.ooni.org/m/%s", result.MeasurementUID)
	return result.MeasurementUID, nil
}

// Submit implements [model.Submitter]. It submits the measurement using the
// stored credential and persists the rotated credential.
func (cs *CredentialSubmitter) Submit(ctx context.Context, m *model.Measurement) (string, error) {
	uid, err := cs.submitWithCredential(m)
	if err != nil {
		cs.logger.Warnf("userauth: credential submission failed, falling back to collector: %s", err.Error())
		return cs.fallback.Submit(ctx, m)
	}

	m.MeasurementUID = uid
	return uid, nil
}
