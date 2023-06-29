// Package throttling wraps connections to measure throttling.
package throttling

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/memoryless"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// Sampler periodically samples the bytes sent and received by a [*measurexlite.Trace]. The zero
// value of this structure is invalid; please, construct using [NewSampler].
type Sampler struct {
	// cancel tells the background goroutine to stop
	cancel context.CancelFunc

	// mu provides mutual exclusion
	mu *sync.Mutex

	// once ensures that close has "once" semantics
	once *sync.Once

	// q is the queue of events we are collecting
	q []*model.ArchivalNetworkEvent

	// tx is the trace we are sampling from
	tx *measurexlite.Trace

	// wg is the waitgroup to wait for the sampler to join
	wg *sync.WaitGroup
}

// NewSampler attaches a [*Sampler] to a [*measurexlite.Trace], starts sampling in the
// background and returns the [*Sampler]. Remember to call [*Sampler.Close] to stop
// the background goroutine that performs the sampling.
func NewSampler(tx *measurexlite.Trace) *Sampler {
	ctx, cancel := context.WithCancel(context.Background())
	smpl := &Sampler{
		cancel: cancel,
		mu:     &sync.Mutex{},
		once:   &sync.Once{},
		q:      []*model.ArchivalNetworkEvent{},
		tx:     tx,
		wg:     &sync.WaitGroup{},
	}
	smpl.wg.Add(1)
	go smpl.mainLoop(ctx)
	return smpl
}

func (smpl *Sampler) mainLoop(ctx context.Context) {
	// make sure the parent knows when we're done running
	defer smpl.wg.Done()

	// From the memoryless documentation:
	//
	//	The exact mathematical meaning of "too extreme" depends on your situation,
	//	but a nice rule of thumb is config.Min should be at most 10% of expected and
	//	config.Max should be at least 250% of expected. These values mean that less
	//	than 10% of time you will be waiting config.Min and less than 10% of the time
	//	you will be waiting config.Max.
	//
	// So, we are going to use 250 milliseconds of expected, 25 milliseconds for the
	// minimum value, and 650 milliseconds for the maximum value.
	config := memoryless.Config{
		Expected: 250 * time.Millisecond,
		Min:      25 * time.Millisecond,
		Max:      650 * time.Millisecond,
		Once:     false,
	}

	// create the memoryless ticker
	ticker := runtimex.Try1(memoryless.NewTicker(ctx, config))
	defer ticker.Stop()

	// loop until we're asked to stop through the context
	for {
		select {
		case <-ctx.Done():
			return

		case <-ticker.C:
			smpl.collectSnapshot(smpl.tx.CloneBytesReceivedMap())
		}
	}
}

// BytesReceivedCumulativeOperation is the operation we set for network events.
const BytesReceivedCumulativeOperation = "bytes_received_cumulative"

func (smpl *Sampler) collectSnapshot(stats map[string]int64) {
	// compute just once the events sampling time
	now := smpl.tx.TimeSince(smpl.tx.ZeroTime).Seconds()

	// process each entry
	for key, count := range stats {
		// extract the network and the address from the map key
		// note: the format is "EPNT_ADDRESS NETWORK"
		vector := strings.Split(key, " ")
		if len(vector) != 2 {
			continue
		}
		address, network := vector[0], vector[1]

		// fill the event
		ev := &model.ArchivalNetworkEvent{
			Address:       address,
			Failure:       nil,
			NumBytes:      count,
			Operation:     BytesReceivedCumulativeOperation,
			Proto:         network,
			T0:            now,
			T:             now,
			TransactionID: smpl.tx.Index,
			Tags:          smpl.tx.Tags(),
		}

		// lock and insert
		smpl.mu.Lock()
		smpl.q = append(smpl.q, ev)
		smpl.mu.Unlock()
	}
}

// Close closes the [*Sampler]. This method is goroutine safe and idempotent.
func (smpl *Sampler) Close() error {
	smpl.once.Do(func() {
		smpl.cancel()
		smpl.wg.Wait()
	})
	return nil
}

// ExtractSamples extracts the samples from the [*Sampler]
func (smpl *Sampler) ExtractSamples() []*model.ArchivalNetworkEvent {
	// collect one last sample -- no need to lock since collectSnapshot locks the mutex
	smpl.collectSnapshot(smpl.tx.CloneBytesReceivedMap())

	// lock and extract all samples
	smpl.mu.Lock()
	o := smpl.q
	smpl.q = []*model.ArchivalNetworkEvent{}
	smpl.mu.Unlock()
	return o
}
