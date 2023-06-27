package measurexlite

//
// Collecting speed samples
//

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/ooni/probe-cli/v3/internal/memoryless"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// ReadSummaryOperation is the [model.ArchivalNetworkEvent] operation used when
// summarizing the download speed during a TCP or UDP download.
const ReadSummaryOperation = "read_summary"

// updateReadSummaryPacketConn updates the read summary for a packet conn. The addr argument
// MAY be nil in case the [net.PacketConn.ReadFrom] operation failed. In such a case, this
// function will do nothing and otherwise it will update the statistics. This function locks
// the underling read summary mutex and, as such, it does not cause data races.
func (tx *Trace) updateReadSummaryPacketConn(network string, addr net.Addr, count int) {
	if addr != nil {
		tx.updateReadSummaryNetworkAddress(network, addr.String(), count)
	}
}

// updateReadSummaryNetworkAddress updates the read summary for a given network and address. This
// function locks the underling read summary mutex and, as such, it does not cause data races.
func (tx *Trace) updateReadSummaryNetworkAddress(network, address string, count int) {
	switch network {
	case "tcp":
		tx.readSummaryMu.Lock()
		tx.readSummaryTCP[address] += int64(count)
		tx.readSummaryMu.Unlock()

	case "udp":
		tx.readSummaryMu.Lock()
		tx.readSummaryUDP[address] += int64(count)
		tx.readSummaryMu.Unlock()
	}
}

// sampleReadSummary collects the currently available read summary information and organizes
// it as a list of [model.ArchivalNetworkEvent]. Calling this function clears the previous
// read summary information such that a subsequent call would only observe what changed since
// the current call. This function locks the underling read summary mutex and, as such,
// it does not cause data races. The returned list is empty if nothing has changed.
func (tx *Trace) sampleReadSummary() (out []*model.ArchivalNetworkEvent) {
	// obtain the elapsed time
	elapsed := tx.TimeSince(tx.ZeroTime).Seconds()

	// obtain a copy of the tags
	tags := tx.Tags()

	// make sure the output list is not nil
	out = []*model.ArchivalNetworkEvent{}

	// make sure we access the summary in mutual exclusion
	defer tx.readSummaryMu.Unlock()
	tx.readSummaryMu.Lock()

	// collect TCP entries
	for address, bytes := range tx.readSummaryTCP {
		out = append(out, &model.ArchivalNetworkEvent{
			Address:       address,
			Failure:       nil,
			NumBytes:      bytes,
			Operation:     ReadSummaryOperation,
			Proto:         "tcp",
			T0:            elapsed,
			T:             elapsed,
			TransactionID: tx.Index,
			Tags:          tags,
		})
	}

	// clear the available TCP entries
	tx.readSummaryTCP = make(map[string]int64)

	// collect UDP entries
	for address, bytes := range tx.readSummaryUDP {
		out = append(out, &model.ArchivalNetworkEvent{
			Address:       address,
			Failure:       nil,
			NumBytes:      bytes,
			Operation:     ReadSummaryOperation,
			Proto:         "udp",
			T0:            elapsed,
			T:             elapsed,
			TransactionID: tx.Index,
			Tags:          tags,
		})
	}

	// clear the available UDP entries
	tx.readSummaryUDP = make(map[string]int64)

	// return to the caller
	return out
}

// SpeedCollector collects speed samples. The zero value of this struct
// is invalid; please, use the [NewSpeedCollector] constructor.
type SpeedCollector struct {
	cancel context.CancelFunc
	evs    []*model.ArchivalNetworkEvent
	mu     sync.Mutex
	once   sync.Once
	trace  *Trace
	wg     *sync.WaitGroup
}

// NewSpeedCollector creates a new [SpeedCollector] instance and spawns
// a background goroutine collecting speed samples. You MUST call the
// [SpeedCollector.Close] method when done to join this goroutine.
func NewSpeedCollector(tx *Trace) *SpeedCollector {
	ctx, cancel := context.WithCancel(context.Background())
	sc := &SpeedCollector{
		cancel: cancel,
		evs:    []*model.ArchivalNetworkEvent{},
		mu:     sync.Mutex{},
		once:   sync.Once{},
		trace:  tx,
		wg:     &sync.WaitGroup{},
	}
	sc.wg.Add(1)
	go sc.mainLoop(ctx)
	return sc
}

// Close stops the background goroutine and waits for it to join. This
// method is idempotent. Subsequent calls do nothing.
func (sc *SpeedCollector) Close() error {
	sc.once.Do(func() {
		sc.cancel()
		sc.wg.Wait()
	})
	return nil
}

// mainLoop is the main loop collecting samples.
func (sc *SpeedCollector) mainLoop(ctx context.Context) {
	// let the parent goroutine know when we're done
	defer sc.wg.Done()

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
			sc.collectSnapshot()
			return

		case <-ticker.C:
			sc.collectSnapshot()
		}
	}
}

// collectSnapshot reads the summary and updates our internal events list. This method locks
// the [SpeedCollector] mutex and therefore it does not cause any data race.
func (sc *SpeedCollector) collectSnapshot() {
	// obtain the current samples
	samples := sc.trace.sampleReadSummary()

	// update the list of samples in mutual exclusion
	sc.mu.Lock()
	sc.evs = append(sc.evs, samples...)
	sc.mu.Unlock()
}

// ExtractSamples extracts the samples collected so far and empties the internal list
// of samples. This method locks the [SpeedCollector] mutex and is data-race safe.
func (sc *SpeedCollector) ExtractSamples() (out []*model.ArchivalNetworkEvent) {
	sc.mu.Lock()
	out = append(out, sc.evs...)
	sc.evs = nil
	sc.mu.Unlock()
	return
}
