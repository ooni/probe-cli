package fallback

//
// Implementation of Run
//

import (
	"context"
	"errors"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// Director directs the execution of multiple ...
type Director interface {
	// Acquire returns a channel that is posted each time a goroutine
	// is allowed to start ...
	Acquire() <-chan any

	// KVStore returns the key-value store to use.
	KVStore() model.KeyValueStore

	// Key returns the key within the key-value store.
	Key() string

	// Release indicates that a new service terminated running
	Release()

	// ShuffleEvery returns the amount of time after which we should shuffle
	// the services to discover more working services. Returning zero will
	// shuffle at every execution, while returning a negative value will completely
	// disable the shuffling algorithm and always use the most reliable service.
	ShuffleEvery() time.Duration

	// TimeNow should generally be equivalent to [time.Now] but may be
	// a different function to simplify testing.
	TimeNow() time.Time
}

// Service is a service that takes a given Config and returns a Result.
type Service[Config, Result any] interface {
	// Run invokes the service.
	Run(ctx context.Context, config Config) (Result, error)

	// URL is the service unique URL.
	URL() string
}

// ErrAllFailed indicates all services failed.
var ErrAllFailed = errors.New("fallback: all services failed")

// Run runs all services in sequence until one of them work or all of them failed.
func Run[Config, Result any](
	ctx context.Context,
	director Director,
	config Config,
	services ...Service[Config, Result],
) (Result, error) {
	// load serialized state
	serio := newSerializedState(director)

	// convert serialized state to in-memory state
	memo := newMemoryState(serio, services...)

	// sort services by priority - MUTATES
	memo.prioritize()

	// possibly shuffle the services - MUTATES
	memo.maybeShuffle(director)

	// run all services until one works or all fail - MUTATES
	res, err := memo.run(ctx, director, config)

	// attempt to write the state back to disk
	_ = memo.toSerializedState().store(director)

	// return to the caller
	return res, err
}
