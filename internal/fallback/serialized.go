package fallback

//
// Serialized state management
//

import (
	"encoding/json"
	"errors"
	"time"

	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// serializedState contains the state serialized on disk.
type serializedState struct {
	// LastShuffle is the last shuffle time.
	LastShuffle time.Time

	// Services contains the state of all services.
	Services []serializedServiceState

	// Version is the data format version number.
	Version int64
}

// serializedServiceState is the serialized state of a service.
type serializedServiceState struct {
	// Score is the score as a number between 0 and 1.
	Score float64

	// URL is the service URL.
	URL string
}

// serializedDataFormatVersion is the data format version of serialized data.
const serializedDataFormatVersion = 1

// errInvalidSerializedDataFormatVersion indicates the serialized data format version does
// not match the expected data format version we know how to parse.
var errInvalidSerializedDataFormatVersion = errors.New("invalid serialized data format version")

// newSerializedState returns the [serializedState] if possible
// and otherwise returns a default-initialized instance.
func newSerializedState(director Director) *serializedState {
	serio, _ := loadSerializedState(director)
	if serio == nil {
		serio = &serializedState{
			LastShuffle: time.Now(),
			Services:    []serializedServiceState{},
			Version:     serializedDataFormatVersion,
		}
	}
	return serio
}

// loadSerializedState returns the state serialized on disk or empty state.
func loadSerializedState(director Director) (*serializedState, error) {
	// read raw data from the kvstore
	data, err := director.KVStore().Get(director.Key())
	if err != nil {
		return nil, err
	}

	// parse raw data into a proper serialized state
	var serio serializedState
	if err := json.Unmarshal(data, &serio); err != nil {
		return nil, err
	}

	// make sure the version is okay
	if serio.Version != serializedDataFormatVersion {
		return nil, errInvalidSerializedDataFormatVersion
	}

	return &serio, nil
}

// store stores [serializedState] on disk
func (serio *serializedState) store(director Director) error {
	// serialize to bytes
	data := runtimex.Try1(json.Marshal(serio))

	// write to the key-value store
	return director.KVStore().Set(director.Key(), data)
}

// findService finds a service by URL.
func (serio *serializedState) findService(URL string) (*serializedServiceState, bool) {
	for _, s := range serio.Services {
		if URL == s.URL {
			return &s, true
		}
	}
	return nil, false
}
