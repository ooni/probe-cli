package probeservices

import (
	"encoding/json"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// State is the state stored inside the state file
type State struct {
	ClientID string
	Expire   time.Time
	Password string
	Token    string
}

// Auth returns an authentication structure, if possible, otherwise
// it returns nil, meaning that you should login again.
func (s State) Auth() *model.OOAPILoginAuth {
	if s.Token == "" {
		return nil
	}
	if time.Now().Add(30 * time.Second).After(s.Expire) {
		return nil
	}
	return &model.OOAPILoginAuth{Expire: s.Expire, Token: s.Token}
}

// Credentials returns login credentials, if possible, otherwise it
// returns nil, meaning that you should create an account.
func (s State) Credentials() *model.OOAPILoginCredentials {
	if s.ClientID == "" {
		return nil
	}
	if s.Password == "" {
		return nil
	}
	return &model.OOAPILoginCredentials{ClientID: s.ClientID, Password: s.Password}
}

// StateFile is the orchestra state file. It is backed by
// a generic key-value store configured by the user.
type StateFile struct {
	Store model.KeyValueStore
	key   string
}

// NewStateFile creates a new state file backed by a key-value store
func NewStateFile(kvstore model.KeyValueStore) StateFile {
	return StateFile{key: "orchestra.state", Store: kvstore}
}

// SetMockable is a mockable version of Set
func (sf StateFile) SetMockable(s State, mf func(interface{}) ([]byte, error)) error {
	data, err := mf(s)
	if err != nil {
		return err
	}
	return sf.Store.Set(sf.key, data)
}

// Set saves the current state on the key-value store.
func (sf StateFile) Set(s State) error {
	return sf.SetMockable(s, json.Marshal)
}

// GetMockable is a mockable version of Get
func (sf StateFile) GetMockable(sfget func(string) ([]byte, error),
	unmarshal func([]byte, interface{}) error) (State, error) {
	value, err := sfget(sf.key)
	if err != nil {
		return State{}, err
	}
	var state State
	if err := unmarshal(value, &state); err != nil {
		return State{}, err
	}
	return state, nil
}

// Get returns the current state. In case of any error with the
// underlying key-value store, we return an empty state.
func (sf StateFile) Get() (state State) {
	state, _ = sf.GetMockable(sf.Store.Get, json.Unmarshal)
	return
}
