package fallback

//
// In-memory state management
//

import (
	"context"
	"errors"
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

// resultOrError contains a result or an error
type resultOrError[Result any] struct {
	Error  error
	Result Result
}

// run runs each service in sequence until one that works is found or all fail
func (ms *memoryState[Config, Result]) run(
	ctx context.Context,
	director Director,
	config Config,
) (Result, error) {
	// create a cancellable context to interrupt running goroutines
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	// create channel for receiving the goroutines results
	outch := make(chan *resultOrError[Result], len(ms.Services))

	// spawn all the services to run in the background
	for _, svc := range ms.Services {
		go svc.runAsync(ctx, director, config, outch)
	}

	// create vector for receiving the results
	outv := []*resultOrError[Result]{}

	// loop receiving results from goroutines
	for len(outv) < len(ms.Services) {
		out := <-outch
		outv = append(outv, out)

		// as soon as we see the first success, interrupt everyone else
		if out.Error == nil {
			cancel()
		}
	}

	// final loop for deciding whether to return result or error
	merr := multierror.New(ErrAllFailed)
	for _, out := range outv {
		if out.Error != nil {
			merr.Add(out.Error)
			continue
		}
		return out.Result, nil
	}
	zeroResult := *new(Result)
	return zeroResult, merr
}

// ewma is the EWMA parameter used to decay the service score.
const ewma = 0.9

// runAsync runs the given service and MUTATES it to update the score.
func (ms *memoryServiceState[Config, Result]) runAsync(
	ctx context.Context,
	director Director,
	config Config,
	outch chan *resultOrError[Result],
) {
	// block until authorized to run or interrupted by context
	select {
	case <-director.Semaphore():
	case <-ctx.Done():
		return
	}

	// XXX: need to tell the parent we're done

	// perform the actual service operation and obtain its results
	result, err := ms.Run(ctx, config)

	// handle errors treating interrupted by context specially
	// because that is not a real error that occurred but rather
	// the parent goroutine that has interrupted us
	if err != nil {
		if !errors.Is(err, context.Canceled) { // XXX
			ms.Score *= (1 - ewma) // decay
		}
		outch <- &resultOrError[Result]{Error: err}
		return
	}

	// handle the successful case
	ms.Score = 1.0*ewma + (1-ewma)*ms.Score // update
	outch <- &resultOrError[Result]{Result: result}
}
