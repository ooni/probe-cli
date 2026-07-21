package userauth

import (
	"encoding/json"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// credStoreKey is the key-value store key under which we persist the credential.
const credStoreKey = "userauth.state"

// StoredCredential is the persisted anonymous credential together with the
// manifest metadata it was issued against.
type StoredCredential struct {
	// Credential is the base64-encoded credential.
	Credential string `json:"credential"`

	// ManifestVersion is the manifest version the credential was issued for.
	ManifestVersion string `json:"manifest_version"`

	// PublicParams is the base64-encoded public parameters of that manifest.
	PublicParams string `json:"public_params"`
}

// CredStore persists a [StoredCredential] in a generic key-value store,
// mirroring probeservices.StateFile.
type CredStore struct {
	Store model.KeyValueStore
}

// NewCredStore creates a new CredStore backed by the given key-value store.
func NewCredStore(kvstore model.KeyValueStore) *CredStore {
	return &CredStore{Store: kvstore}
}

// Get returns the stored credential. When there is no credential yet, or the
// store cannot be read, it returns a zero StoredCredential
func (cs *CredStore) Get() (cred StoredCredential) {
	value, err := cs.Store.Get(credStoreKey)
	if err != nil {
		return StoredCredential{}
	}
	if err := json.Unmarshal(value, &cred); err != nil {
		return StoredCredential{}
	}
	return cred
}

// Set persists the given credential.
func (cs *CredStore) Set(cred StoredCredential) error {
	data, err := json.Marshal(cred)
	if err != nil {
		return err
	}
	return cs.Store.Set(credStoreKey, data)
}
