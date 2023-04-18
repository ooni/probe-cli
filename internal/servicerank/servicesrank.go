// Package servicerank ranks services depending on how well they work.
package servicerank

import (
	"encoding/json"
	"errors"
	"math/rand"
	"sort"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// Service is a service we're monitoring. The status of the service is
// summarized by a score varying between 0.0 and 1.0.
type Service interface {
	// SetScore sets the score.
	//
	// The implementation of this method MUST be concurrency safe.
	SetScore(value float64)

	// Score gets the score.
	//
	// The implementation of this method MUST be concurrency safe.
	Score() float64

	// URL is the unique URL describing this service.
	URL() string
}

// State contains the service rank state.
type State struct {
	// LastShuffle is the last time we shuffled the services.
	LastShuffle time.Time

	// Services contains the list of services.
	Services []Service
}

// serializedState is the serialized state.
type serializedState struct {
	LastShuffle time.Time
	Services    []serializedService
	Version     int
}

// dataFormatVersion is the data format version.
const dataFormatVersion = 1

// errInvalidDataFormatVersion indicates the on-disk data format version is invalid.
var errInvalidDataFormatVersion = errors.New("invalid data format version")

// serializedService contains serialized service information.
type serializedService struct {
	Score float64
	URL   string
}

// Load loads scores from a key-value store using the given key.
func Load(kvs model.KeyValueStore, key string, services ...Service) *State {
	// load serialized state and compensate for errors
	ss, err := loadSerializedState(kvs, key)
	if err != nil {
		ss = &serializedState{
			LastShuffle: time.Now(),
			Services:    []serializedService{},
			Version:     dataFormatVersion,
		}
	}

	// fill state by copying from the serialized state
	state := &State{
		LastShuffle: ss.LastShuffle,
		Services:    services,
	}
	for _, svc := range state.Services {
		if s, good := ss.findService(svc.URL()); good {
			svc.SetScore(s.Score)
		}
	}
	return state
}

// findService finds a service by URL
func (ss *serializedState) findService(URL string) (*serializedService, bool) {
	for _, svc := range ss.Services {
		if URL == svc.URL {
			return &svc, true
		}
	}
	return nil, false
}

// loadSerializedState loads the serialized state from the key-value store.
func loadSerializedState(kvs model.KeyValueStore, key string) (*serializedState, error) {
	data, err := kvs.Get(key)
	if err != nil {
		return nil, err
	}
	var ss serializedState
	if err := json.Unmarshal(data, &ss); err != nil {
		return nil, err
	}
	if ss.Version != dataFormatVersion {
		return nil, errInvalidDataFormatVersion
	}
	return &ss, nil
}

// Ranked returns a copy of the [State] where Services are ranked by score.
func (s *State) Ranked() (out *State) {
	out = &State{
		LastShuffle: s.LastShuffle,
		Services:    append([]Service{}, s.Services...),
	}
	sort.SliceStable(out.Services, func(i, j int) bool {
		return out.Services[i].Score() <= out.Services[j].Score()
	})
	return out
}

// RandomlyShuffled returns a copy of the [State] where Services are randomly shuffled
// and the LastShuffle field has been updated.
func (s *State) RandomlyShuffled() (out *State) {
	out = &State{
		LastShuffle: time.Now(),
		Services:    append([]Service{}, s.Services...),
	}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	r.Shuffle(len(out.Services), func(i, j int) {
		out.Services[i], out.Services[j] = out.Services[j], out.Services[i]
	})
	return out
}

// Store stores the state into a key-value store using the given key.
func Store(kvs model.KeyValueStore, key string, state *State) error {
	ss := &serializedState{
		LastShuffle: state.LastShuffle,
		Services:    []serializedService{},
		Version:     dataFormatVersion,
	}
	for _, svc := range state.Services {
		ss.Services = append(ss.Services, serializedService{
			Score: svc.Score(),
			URL:   svc.URL(),
		})
	}
	data := runtimex.Try1(json.Marshal(ss))
	return kvs.Set(key, data)
}
