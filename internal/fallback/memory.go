package fallback

//
// In-memory state management
//

import (
	"context"
	"math/rand"
	"sort"
	"time"

	"github.com/ooni/probe-cli/v3/internal/multierror"
)

// memoryState is the in-memory state
type memoryState[Config, Result any] struct {
	// LastShuffle is the last shuffle time.
	LastShuffle time.Time

	// Services contains the state of all services.
	Services []*memoryServiceState[Config, Result]
}

// memoryServiceState is the in-memory state of a service
type memoryServiceState[Config, Result any] struct {
	// Run is the service function
	Run func(ctx context.Context, config Config) (Result, error)

	// Score is the score as a number between 0 and 1.
	Score float64

	// URL is the service URL.
	URL string
}

// newMemoryState creates [memoryState] from [serializedState] and [Service] instances.
func newMemoryState[Config, Result any](
	serio *serializedState, services ...Service[Config, Result]) *memoryState[Config, Result] {
	// create empty in-memory state
	mem := &memoryState[Config, Result]{
		LastShuffle: serio.LastShuffle,
		Services:    []*memoryServiceState[Config, Result]{},
	}

	// fill information about the services
	for _, svc := range services {
		ms := &memoryServiceState[Config, Result]{
			Run:   svc.Run,
			Score: 0,
			URL:   svc.URL(),
		}
		if s, good := serio.findService(svc.URL()); good {
			ms.Score = s.Score
		}
		mem.Services = append(mem.Services, ms)
	}

	return mem
}

// toSerializedState converts to [serializedState]
func (ms *memoryState[Config, Result]) toSerializedState() *serializedState {
	// create empty serialized state
	serio := &serializedState{
		LastShuffle: ms.LastShuffle,
		Services:    []serializedServiceState{},
		Version:     serializedDataFormatVersion,
	}

	// fill services
	for _, svc := range ms.Services {
		serio.Services = append(serio.Services, serializedServiceState{
			Score: svc.Score,
			URL:   svc.URL,
		})
	}

	return serio
}

// prioritize MUTATES the memory state to sort services by priority.
func (ms *memoryState[Config, Result]) prioritize() {
	sort.SliceStable(ms.Services, func(i, j int) bool {
		return ms.Services[i].Score <= ms.Services[j].Score
	})
}

// maybeShuffle possibly MUTATES the memory state to shuffle the services randomly
func (ms *memoryState[Config, Result]) maybeShuffle(director Director) {
	shuffleEvery := director.ShuffleEvery()
	now := director.TimeNow()
	if shuffleEvery >= 0 && now.Sub(ms.LastShuffle) > shuffleEvery { // as documented
		ms.LastShuffle = now
		r := rand.New(rand.NewSource(time.Now().UnixNano()))
		r.Shuffle(len(ms.Services), func(i, j int) {
			ms.Services[i], ms.Services[j] = ms.Services[j], ms.Services[i]
		})
	}
}

// run runs each service in sequence until one that works is found or all fail
func (ms *memoryState[Config, Result]) run(
	ctx context.Context, config Config) (Result, error) {
	merr := multierror.New(ErrAllFailed)
	for _, svc := range ms.Services {
		res, err := svc.run(ctx, config)
		if err != nil {
			merr.Add(err)
			continue
		}
		return res, nil
	}
	zero := *new(Result)
	return zero, merr
}

// ewma is the EWMA parameter used to decay the service score.
const ewma = 0.9

// run runs the given service and MUTATES it to update the score.
func (ms *memoryServiceState[Config, Result]) run(
	ctx context.Context, config Config) (Result, error) {
	res, err := ms.Run(ctx, config)
	if err != nil {
		ms.Score *= (1 - ewma) // decay
		zero := *new(Result)
		return zero, err
	}
	ms.Score = 1.0*ewma + (1-ewma)*ms.Score // update
	return res, nil
}
