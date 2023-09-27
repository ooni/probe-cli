package enginenetx

//
// Code to keep statistics about the TLS dialing
// tactics that work and the ones that don't
//

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// nullStatsManager is the "null" [httpsDialerEventsHandler].
type nullStatsManager struct{}

var _ httpsDialerEventsHandler = &nullStatsManager{}

// OnStarting implements httpsDialerEventsHandler.
func (*nullStatsManager) OnStarting(tactic *httpsDialerTactic) {
	// nothing
}

// OnSuccess implements httpsDialerEventsHandler.
func (*nullStatsManager) OnSuccess(tactic *httpsDialerTactic) {
	// nothing
}

// OnTCPConnectError implements httpsDialerEventsHandler.
func (*nullStatsManager) OnTCPConnectError(ctx context.Context, tactic *httpsDialerTactic, err error) {
	// nothing
}

// OnTLSHandshakeError implements httpsDialerEventsHandler.
func (*nullStatsManager) OnTLSHandshakeError(ctx context.Context, tactic *httpsDialerTactic, err error) {
	// nothing
}

// OnTLSVerifyError implements httpsDialerEventsHandler.
func (*nullStatsManager) OnTLSVerifyError(tactic *httpsDialerTactic, err error) {
	// nothing
}

// statsTactic keeps stats about an [*httpsDialerTactic].
type statsTactic struct {
	// CountStarted counts the number of operations we started.
	CountStarted int64

	// CountTCPConnectError counts the number of TCP connect errors.
	CountTCPConnectError int64

	// CountTCPConnectInterrupt counts the number of interrupted TCP connect attempts.
	CountTCPConnectInterrupt int64

	// CountTLSHandshakeError counts the number of TLS handshake errors.
	CountTLSHandshakeError int64

	// CountTLSHandshakeInterrupt counts the number of interrupted TLS handshakes.
	CountTLSHandshakeInterrupt int64

	// CountTLSVerificationError counts the number of TLS verification errors.
	CountTLSVerificationError int64

	// CountSuccess counts the number of successes.
	CountSuccess int64

	// HistoTCPConnectError contains an histogram of TCP connect errors.
	HistoTCPConnectError map[string]int64

	// HistoTLSHandshakeError contains an histogram of TLS handshake errors.
	HistoTLSHandshakeError map[string]int64

	// HistoTLSVerificationError contains an histogram of TLS verification errors.
	HistoTLSVerificationError map[string]int64

	// LastUpdated is the last time we updated this record.
	LastUpdated time.Time

	// Tactic is the underlying tactic.
	Tactic *httpsDialerTactic
}

// statsNilSafeSuccessRate is a convenience function for computing the success rate
// which returns zero as the success rate if CountStarted is zero
//
// for robustness, be paranoid about nils here because the stats are
// written on the disk and a user could potentially edit them
func statsNilSafeSuccessRate(t *statsTactic) (rate float64) {
	if t != nil && t.CountStarted > 0 {
		rate = float64(t.CountSuccess) / float64(t.CountStarted)
	}
	return
}

// statsNilSafeLastUpdated is a convenience function for getting the .LastUpdated
// field that takes into account the case where t is nil.
func statsNilSafeLastUpdated(t *statsTactic) (output time.Time) {
	if t != nil {
		output = t.LastUpdated
	}
	return
}

// statsNilSafeCountSuccess is a convenience function for getting the .CountSuccess
// counter that takes into account the case where t is nil.
func statsNilSafeCountSuccess(t *statsTactic) (output int64) {
	if t != nil {
		output = t.CountSuccess
	}
	return
}

// statsDefensivelySortTacticsByDescendingSuccessRateWithAcceptPredicate sorts the input list
// by success rate taking into account that several entries could be malformed, and then
// filters the sorted list using the given boolean predicate to accept elements.
//
// The sorting criteria takes into account:
//
// 1. the success rate; or
//
// 2. the last updated time; or
//
// 3. the number of successes.
//
// The predicate allows to further restrict the returned list.
//
// This function operates on a deep copy of the input list, so it does not create data races.
func statsDefensivelySortTacticsByDescendingSuccessRateWithAcceptPredicate(
	input []*statsTactic, acceptfunc func(*statsTactic) bool) []*statsTactic {
	// first let's create a working list such that we don't modify
	// the input in place thus avoiding any data race
	work := []*statsTactic{}
	for _, t := range input {
		if t != nil && t.Tactic != nil {
			work = append(work, t.Clone()) // DEEP COPY!
		}
	}

	// now let's sort work in place
	sort.SliceStable(work, func(i, j int) bool {
		if statsNilSafeSuccessRate(work[i]) > statsNilSafeSuccessRate(work[j]) {
			return true
		}
		if statsNilSafeLastUpdated(work[i]).Sub(statsNilSafeLastUpdated(work[j])) > 0 {
			return true
		}
		if statsNilSafeCountSuccess(work[i]) > statsNilSafeCountSuccess(work[j]) {
			return true
		}
		return false
	})

	// finally let's apply the predicate to produce output
	output := []*statsTactic{}
	for _, t := range work {
		if acceptfunc(t) {
			output = append(output, t)
		}
	}
	return output
}

func statsMaybeCloneMapStringInt64(input map[string]int64) (output map[string]int64) {
	// distinguish and preserve nil versus empty
	if input == nil {
		return
	}
	output = make(map[string]int64)
	for key, value := range input {
		output[key] = value
	}
	return
}

func statsMaybeCloneTactic(input *httpsDialerTactic) (output *httpsDialerTactic) {
	if input != nil {
		output = input.Clone()
	}
	return
}

// Clone clones a given [*statsTactic]
func (st *statsTactic) Clone() *statsTactic {
	// Implementation note: a time.Time consists of an uint16, an int64 and
	// a pointer to a location which is typically immutable, so it's perfectly
	// fine to copy the LastUpdate field by assignment.
	//
	// here we're using a bunch of robustness aware mechanisms to clone
	// considering that the struct may be edited by the user
	return &statsTactic{
		CountStarted:               st.CountStarted,
		CountTCPConnectError:       st.CountTCPConnectError,
		CountTCPConnectInterrupt:   st.CountTCPConnectInterrupt,
		CountTLSHandshakeError:     st.CountTLSHandshakeError,
		CountTLSHandshakeInterrupt: st.CountTLSHandshakeInterrupt,
		CountTLSVerificationError:  st.CountTLSVerificationError,
		CountSuccess:               st.CountSuccess,
		HistoTCPConnectError:       statsMaybeCloneMapStringInt64(st.HistoTCPConnectError),
		HistoTLSHandshakeError:     statsMaybeCloneMapStringInt64(st.HistoTLSHandshakeError),
		HistoTLSVerificationError:  statsMaybeCloneMapStringInt64(st.HistoTLSVerificationError),
		LastUpdated:                st.LastUpdated,
		Tactic:                     statsMaybeCloneTactic(st.Tactic),
	}
}

// statsDomainEndpoint contains stats associated with a domain endpoint.
type statsDomainEndpoint struct {
	Tactics map[string]*statsTactic
}

// statsDomainEndpointPruneEntries returns a DEEP COPY of a [*statsDomainEndpoint] with old
// and excess entries removed, such that the overall size is not unbounded.
func statsDomainEndpointPruneEntries(input *statsDomainEndpoint) *statsDomainEndpoint {
	tactics := []*statsTactic{}
	now := time.Now()

	// if .Tactics is empty here we're just going to do nothing
	for summary, tactic := range input.Tactics {
		// we serialize stats to disk, so we cannot rule out the case where the user
		// explicitly edits the stats to include a malformed entry
		if summary == "" || tactic == nil || tactic.Tactic == nil {
			continue
		}
		tactics = append(tactics, tactic)
	}

	// oneWeek is a constant representing one week of data.
	const oneWeek = 7 * 24 * time.Hour

	// maxEntriesPerDomainEndpoint is the maximum number of entries per
	// domain endpoint that we would like to keep overall.
	const maxEntriesPerDomainEndpoint = 10

	// Sort by descending success rate and cut all the entries that are older than
	// a given threshold. Note that we need to be defensive here because we are dealing
	// with data stored on disk that might have been modified to crash us.
	//
	// Note that statsDefensivelySortTacticsByDescendingSuccessRateWithAcceptPredicate
	// operates on and returns a DEEP COPY of the original list.
	tactics = statsDefensivelySortTacticsByDescendingSuccessRateWithAcceptPredicate(
		tactics, func(st *statsTactic) bool {
			// When .LastUpdated is the zero time.Time value, the check is going to fail
			// exactly like the time was 1 or 5 or 10 years ago instead.
			//
			// See https://go.dev/play/p/HGQT17ueIkq where we show that the zero time
			// is handled exactly like any time in the past (it was kinda obvious, but
			// sometimes it also make sense to double check assumptions!)
			delta := now.Sub(statsNilSafeLastUpdated(st))
			return delta < oneWeek
		})

	// Cut excess entries, if needed
	if len(tactics) > maxEntriesPerDomainEndpoint {
		tactics = tactics[:maxEntriesPerDomainEndpoint]
	}

	// return a new statsDomainEndpoint to the caller
	output := &statsDomainEndpoint{
		Tactics: map[string]*statsTactic{},
	}
	for _, t := range tactics {
		output.Tactics[t.Tactic.tacticSummaryKey()] = t
	}
	return output
}

// statsContainerVersion is the current version of [statsContainer].
const statsContainerVersion = 5

// statsContainer is the root container for the stats.
//
// The zero value is invalid; construct using [newStatsContainer].
type statsContainer struct {
	// DomainEndpoints maps a domain endpoint to its tactics.
	DomainEndpoints map[string]*statsDomainEndpoint

	// Version is the version of the container data format.
	Version int
}

// statsContainerPruneEntries returns a DEEP COPY of a [*statsContainer] with old entries removed.
func statsContainerPruneEntries(input *statsContainer) (output *statsContainer) {
	output = newStatsContainer()

	// if .DomainEndpoints is nil here we're just going to do nothing
	for domainEpnt, inputStats := range input.DomainEndpoints {

		// We serialize this data to disk, so we need to account for the case
		// where a user has manually edited the JSON to add a nil value
		if domainEpnt == "" || inputStats == nil || len(inputStats.Tactics) <= 0 {
			continue
		}

		prunedStats := statsDomainEndpointPruneEntries(inputStats)

		// We don't want to include an entry when it's empty because all the
		// stats inside it have just been pruned
		if len(prunedStats.Tactics) <= 0 {
			continue
		}

		output.DomainEndpoints[domainEpnt] = prunedStats
	}
	return
}

// GetStatsTacticLocked returns the tactic record for the given [*statsTactic] instance.
//
// As the name implies, this function MUST be called while holding the [*statsManager] mutex.
func (c *statsContainer) GetStatsTacticLocked(tactic *httpsDialerTactic) (*statsTactic, bool) {
	domainEpntRecord, found := c.DomainEndpoints[tactic.domainEndpointKey()]
	if !found || domainEpntRecord == nil {
		return nil, false
	}
	tacticRecord, found := domainEpntRecord.Tactics[tactic.tacticSummaryKey()]
	return tacticRecord, found
}

// SetStatsTacticLocked sets the tactic record for the given the given [*statsTactic] instance.
//
// As the name implies, this function MUST be called while holding the [*statsManager] mutex.
func (c *statsContainer) SetStatsTacticLocked(tactic *httpsDialerTactic, record *statsTactic) {
	domainEpntRecord, found := c.DomainEndpoints[tactic.domainEndpointKey()]
	if !found {
		domainEpntRecord = &statsDomainEndpoint{
			Tactics: map[string]*statsTactic{},
		}

		// make sure the map is initialized -- not a void concern given that we're
		// reading this structure from the disk
		if len(c.DomainEndpoints) <= 0 {
			c.DomainEndpoints = make(map[string]*statsDomainEndpoint)
		}

		c.DomainEndpoints[tactic.domainEndpointKey()] = domainEpntRecord
		// fallthrough
	}
	domainEpntRecord.Tactics[tactic.tacticSummaryKey()] = record
}

// newStatsContainer creates a new empty [*statsContainer].
func newStatsContainer() *statsContainer {
	return &statsContainer{
		DomainEndpoints: map[string]*statsDomainEndpoint{},
		Version:         statsContainerVersion,
	}
}

// statsManager implements [httpsDialerEventsHandler] by storing the
// relevant statistics in a [model.KeyValueStore].
//
// The zero value of this structure is not ready to use; please, use the
// [newStatsManager] factory to create a new instance.
type statsManager struct {
	// cancel allows canceling the background stats pruner.
	cancel context.CancelFunc

	// container is the container container for stats
	container *statsContainer

	// kvStore is the key-value store we're using
	kvStore model.KeyValueStore

	// logger is the logger to use.
	logger model.Logger

	// mu provides mutual exclusion when accessing the stats.
	mu sync.Mutex
}

// statsKey is the key used in the key-value store to access the state.
const statsKey = "httpsdialerstats.state"

// errStatsContainerWrongVersion means that the stats container document has the wrong version number.
var errStatsContainerWrongVersion = errors.New("wrong stats container version")

// loadStatsContainer loads a stats container from the given [model.KeyValueStore].
func loadStatsContainer(kvStore model.KeyValueStore) (*statsContainer, error) {
	// load data from the kvstore
	data, err := kvStore.Get(statsKey)
	if err != nil {
		return nil, err
	}

	// parse as JSON
	var container statsContainer
	if err := json.Unmarshal(data, &container); err != nil {
		return nil, err
	}

	// make sure the version is OK
	if container.Version != statsContainerVersion {
		err := fmt.Errorf(
			"%s: %w: expected=%d got=%d",
			statsKey,
			errStatsContainerWrongVersion,
			statsContainerVersion,
			container.Version,
		)
		return nil, err
	}

	// make sure we prune the data structure
	pruned := statsContainerPruneEntries(&container)
	return pruned, nil
}

// newStatsManager constructs a new instance of [*statsManager].
func newStatsManager(kvStore model.KeyValueStore, logger model.Logger) *statsManager {
	root, err := loadStatsContainer(kvStore)
	if err != nil {
		root = newStatsContainer()
	}

	ctx, cancel := context.WithCancel(context.Background())

	mt := &statsManager{
		cancel:    cancel,
		container: root,
		kvStore:   kvStore,
		logger:    logger,
		mu:        sync.Mutex{},
	}

	// run a background goroutine that trims the stats by removing excessive
	// entries until the programmer calls (*statsManager).Close
	go mt.trim(ctx)

	return mt
}

var _ httpsDialerEventsHandler = &statsManager{}

// OnStarting implements httpsDialerEventsHandler.
func (mt *statsManager) OnStarting(tactic *httpsDialerTactic) {
	// get exclusive access
	defer mt.mu.Unlock()
	mt.mu.Lock()

	// get the record
	record, found := mt.container.GetStatsTacticLocked(tactic)
	if !found {
		record = &statsTactic{
			CountStarted:               0,
			CountTCPConnectError:       0,
			CountTCPConnectInterrupt:   0,
			CountTLSHandshakeError:     0,
			CountTLSHandshakeInterrupt: 0,
			CountTLSVerificationError:  0,
			CountSuccess:               0,
			HistoTCPConnectError:       map[string]int64{},
			HistoTLSHandshakeError:     map[string]int64{},
			HistoTLSVerificationError:  map[string]int64{},
			LastUpdated:                time.Time{},
			Tactic:                     tactic.Clone(), // avoid storing the original
		}
		mt.container.SetStatsTacticLocked(tactic, record)
	}

	// update stats
	record.CountStarted++
	record.LastUpdated = time.Now()
}

func statsSafeIncrementMapStringInt64(input *map[string]int64, value string) {
	runtimex.Assert(input != nil, "passed nil pointer to a map")
	if *input == nil {
		*input = make(map[string]int64)
	}
	(*input)[value]++
}

// OnTCPConnectError implements httpsDialerEventsHandler.
func (mt *statsManager) OnTCPConnectError(ctx context.Context, tactic *httpsDialerTactic, err error) {
	// get exclusive access
	defer mt.mu.Unlock()
	mt.mu.Lock()

	// get the record
	record, found := mt.container.GetStatsTacticLocked(tactic)
	if !found {
		mt.logger.Warnf("statsManager.OnTCPConnectError: not found: %+v", tactic)
		return
	}

	// update stats
	record.LastUpdated = time.Now()
	if ctx.Err() != nil {
		record.CountTCPConnectInterrupt++
		return
	}

	runtimex.Assert(err != nil, "OnTCPConnectError passed a nil error")
	record.CountTCPConnectError++
	statsSafeIncrementMapStringInt64(&record.HistoTCPConnectError, err.Error())
}

// OnTLSHandshakeError implements httpsDialerEventsHandler.
func (mt *statsManager) OnTLSHandshakeError(ctx context.Context, tactic *httpsDialerTactic, err error) {
	// get exclusive access
	defer mt.mu.Unlock()
	mt.mu.Lock()

	// get the record
	record, found := mt.container.GetStatsTacticLocked(tactic)
	if !found {
		mt.logger.Warnf("statsManager.OnTLSHandshakeError: not found: %+v", tactic)
		return
	}

	// update stats
	record.LastUpdated = time.Now()
	if ctx.Err() != nil {
		record.CountTLSHandshakeInterrupt++
		return
	}

	runtimex.Assert(err != nil, "OnTLSHandshakeError passed a nil error")
	record.CountTLSHandshakeError++
	statsSafeIncrementMapStringInt64(&record.HistoTLSHandshakeError, err.Error())
}

// OnTLSVerifyError implements httpsDialerEventsHandler.
func (mt *statsManager) OnTLSVerifyError(tactic *httpsDialerTactic, err error) {
	// get exclusive access
	defer mt.mu.Unlock()
	mt.mu.Lock()

	// get the record
	record, found := mt.container.GetStatsTacticLocked(tactic)
	if !found {
		mt.logger.Warnf("statsManager.OnTLSVerificationError: not found: %+v", tactic)
		return
	}

	// update stats
	runtimex.Assert(err != nil, "OnTLSVerifyError passed a nil error")
	record.CountTLSVerificationError++
	statsSafeIncrementMapStringInt64(&record.HistoTLSVerificationError, err.Error())
	record.LastUpdated = time.Now()
}

// OnSuccess implements httpsDialerEventsHandler.
func (mt *statsManager) OnSuccess(tactic *httpsDialerTactic) {
	// get exclusive access
	defer mt.mu.Unlock()
	mt.mu.Lock()

	// get the record
	record, found := mt.container.GetStatsTacticLocked(tactic)
	if !found {
		mt.logger.Warnf("statsManager.OnSuccess: not found: %+v", tactic)
		return
	}

	// update stats
	record.CountSuccess++
	record.LastUpdated = time.Now()
}

// Close implements io.Closer
func (mt *statsManager) Close() error {
	// TODO(bassosimone): do we need to apply a "once" semantics to this method? Perhaps no
	// given that there is no resource that we can close only once...

	// interrupt the background goroutine
	mt.cancel()

	// get exclusive access
	defer mt.mu.Unlock()
	mt.mu.Lock()

	// make sure we remove the unneeded entries one last time before saving them
	container := statsContainerPruneEntries(mt.container)

	// write updated stats into the underlying key-value store
	return mt.kvStore.Set(statsKey, runtimex.Try1(json.Marshal(container)))
}

// trim runs in the background and trims the mt.container struct
func (mt *statsManager) trim(ctx context.Context) {
	const interval = 30 * time.Second
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return

		case <-t.C:

			// get exclusive access and edit the container
			mt.mu.Lock()
			mt.container = statsContainerPruneEntries(mt.container)
			mt.mu.Unlock()

		}
	}
}

// LookupTacticsStats returns stats about tactics for a given domain and port. The returned
// list is a clone of the one stored by [*statsManager] so, it can easily be modified.
func (mt *statsManager) LookupTactics(domain string, port string) ([]*statsTactic, bool) {
	out := []*statsTactic{}

	// get exclusive access
	defer mt.mu.Unlock()
	mt.mu.Lock()

	// check whether we have information on this endpoint
	//
	// Note: in case mt.container.DomainEndpoints is nil, this access pattern
	// will return to us a nil pointer and false
	//
	// we also protect against the case where a user has configured a nil
	// domainEpnts value inside the serialized JSON to crash us
	domainEpnts, good := mt.container.DomainEndpoints[net.JoinHostPort(domain, port)]
	if !good || domainEpnts == nil {
		return out, false
	}

	// return a copy of each entry
	//
	// Note: if Tactics here is nil, we're just not going to have
	// anything to include into the out list
	for _, entry := range domainEpnts.Tactics {
		out = append(out, entry.Clone())
	}
	return out, len(out) > 0
}
