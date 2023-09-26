package enginenetx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// statsTactic keeps stats about an [*HTTPSDialerTactic].
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
	Tactic *HTTPSDialerTactic
}

func statsCloneMapStringInt64(input map[string]int64) (output map[string]int64) {
	for key, value := range input {
		if output == nil {
			output = make(map[string]int64) // the idea here is to clone a nil map to a nil map
		}
		output[key] = value
	}
	return
}

// Clone clones a given [*statsTactic]
func (st *statsTactic) Clone() *statsTactic {
	return &statsTactic{
		CountStarted:               st.CountStarted,
		CountTCPConnectError:       st.CountTCPConnectError,
		CountTCPConnectInterrupt:   st.CountTCPConnectInterrupt,
		CountTLSHandshakeError:     st.CountTLSHandshakeError,
		CountTLSHandshakeInterrupt: st.CountTLSHandshakeInterrupt,
		CountTLSVerificationError:  st.CountTLSVerificationError,
		CountSuccess:               st.CountSuccess,
		HistoTCPConnectError:       statsCloneMapStringInt64(st.HistoTCPConnectError),
		HistoTLSHandshakeError:     statsCloneMapStringInt64(st.HistoTLSHandshakeError),
		HistoTLSVerificationError:  statsCloneMapStringInt64(st.HistoTLSVerificationError),
		LastUpdated:                st.LastUpdated,
		Tactic:                     st.Tactic.Clone(),
	}
}

// statsDomainEndpoint contains stats associated with a domain endpoint.
type statsDomainEndpoint struct {
	Tactics map[string]*statsTactic
}

// statsDomainRemoveOldEntries returns a copy of a [*statsDomain] with old entries removed.
func statsDomainRemoveOldEntries(input *statsDomainEndpoint) (output *statsDomainEndpoint) {
	output = &statsDomainEndpoint{
		Tactics: map[string]*statsTactic{},
	}
	oneWeek := 7 * 24 * time.Hour
	now := time.Now()
	for summary, tactic := range input.Tactics {
		if delta := now.Sub(tactic.LastUpdated); delta > oneWeek {
			continue
		}
		output.Tactics[summary] = tactic.Clone()
	}
	return
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

// statsDomainRemoveOldEntries returns a copy of a [*statsContainer] with old entries removed.
func statsContainerRemoveOldEntries(input *statsContainer) (output *statsContainer) {
	output = newStatsContainer()
	for domainEpnt, inputStats := range input.DomainEndpoints {
		prunedStats := statsDomainRemoveOldEntries(inputStats)
		if len(prunedStats.Tactics) <= 0 {
			continue
		}
		output.DomainEndpoints[domainEpnt] = prunedStats
	}
	return
}

// GetStatsTacticLocked returns the tactic record for the given [*statsTactic] instance.
//
// At the name implies, this function MUST be called while holding the [*statsManager] mutex.
func (c *statsContainer) GetStatsTacticLocked(tactic *HTTPSDialerTactic) (*statsTactic, bool) {
	domainEpntRecord, found := c.DomainEndpoints[tactic.domainEndpointKey()]
	if !found {
		return nil, false
	}
	tacticRecord, found := domainEpntRecord.Tactics[tactic.Summary()]
	return tacticRecord, found
}

// SetStatsTacticLocked sets the tactic record for the given the given [*statsTactic] instance.
//
// At the name implies, this function MUST be called while holding the [*statsManager] mutex.
func (c *statsContainer) SetStatsTacticLocked(tactic *HTTPSDialerTactic, record *statsTactic) {
	domainEpntRecord, found := c.DomainEndpoints[tactic.domainEndpointKey()]
	if !found {
		domainEpntRecord = &statsDomainEndpoint{
			Tactics: map[string]*statsTactic{},
		}

		// make sure the map is initialized
		if len(c.DomainEndpoints) <= 0 {
			c.DomainEndpoints = make(map[string]*statsDomainEndpoint)
		}

		c.DomainEndpoints[tactic.domainEndpointKey()] = domainEpntRecord
		// fallthrough
	}
	domainEpntRecord.Tactics[tactic.Summary()] = record
}

// newStatsContainer creates a new empty [*statsContainer].
func newStatsContainer() *statsContainer {
	return &statsContainer{
		DomainEndpoints: map[string]*statsDomainEndpoint{},
		Version:         statsContainerVersion,
	}
}

// statsManager implements [HTTPSDialerStatsTracker] by storing
// the relevant statistics in a [model.KeyValueStore].
//
// The zero value of this structure is not ready to use; please, use the
// [newStatsManager] factory to create a new instance.
type statsManager struct {
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

	// make sure we remove old entries
	pruned := statsContainerRemoveOldEntries(&container)
	return pruned, nil
}

// newStatsManager constructs a new instance of [*statsManager].
func newStatsManager(kvStore model.KeyValueStore, logger model.Logger) *statsManager {
	root, err := loadStatsContainer(kvStore)
	if err != nil {
		root = newStatsContainer()
	}

	return &statsManager{
		container: root,
		kvStore:   kvStore,
		logger:    logger,
		mu:        sync.Mutex{},
	}
}

var _ HTTPSDialerStatsTracker = &statsManager{}

// OnStarting implements HTTPSDialerStatsManager.
func (mt *statsManager) OnStarting(tactic *HTTPSDialerTactic) {
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

// OnTCPConnectError implements HTTPSDialerStatsManager.
func (mt *statsManager) OnTCPConnectError(ctx context.Context, tactic *HTTPSDialerTactic, err error) {
	// get exclusive access
	defer mt.mu.Unlock()
	mt.mu.Lock()

	// get the record
	record, found := mt.container.GetStatsTacticLocked(tactic)
	if !found {
		mt.logger.Warnf("HTTPSDialerStatsManager.OnTCPConnectError: not found: %+v", tactic)
		return
	}

	// update stats
	record.LastUpdated = time.Now()
	if ctx.Err() != nil {
		record.CountTCPConnectInterrupt++
		return
	}
	record.CountTCPConnectError++
	record.HistoTCPConnectError[err.Error()]++
}

// OnTLSHandshakeError implements HTTPSDialerStatsManager.
func (mt *statsManager) OnTLSHandshakeError(ctx context.Context, tactic *HTTPSDialerTactic, err error) {
	// get exclusive access
	defer mt.mu.Unlock()
	mt.mu.Lock()

	// get the record
	record, found := mt.container.GetStatsTacticLocked(tactic)
	if !found {
		mt.logger.Warnf("HTTPSDialerStatsManager.OnTLSHandshakeError: not found: %+v", tactic)
		return
	}

	// update stats
	record.LastUpdated = time.Now()
	if ctx.Err() != nil {
		record.CountTLSHandshakeInterrupt++
		return
	}
	record.CountTLSHandshakeError++
	record.HistoTLSHandshakeError[err.Error()]++
}

// OnTLSVerifyError implements HTTPSDialerStatsManager.
func (mt *statsManager) OnTLSVerifyError(tactic *HTTPSDialerTactic, err error) {
	// get exclusive access
	defer mt.mu.Unlock()
	mt.mu.Lock()

	// get the record
	record, found := mt.container.GetStatsTacticLocked(tactic)
	if !found {
		mt.logger.Warnf("HTTPSDialerStatsManager.OnTLSVerificationError: not found: %+v", tactic)
		return
	}

	// update stats
	record.CountTLSVerificationError++
	record.HistoTLSVerificationError[err.Error()]++
	record.LastUpdated = time.Now()
}

// OnSuccess implements HTTPSDialerStatsManager.
func (mt *statsManager) OnSuccess(tactic *HTTPSDialerTactic) {
	// get exclusive access
	defer mt.mu.Unlock()
	mt.mu.Lock()

	// get the record
	record, found := mt.container.GetStatsTacticLocked(tactic)
	if !found {
		mt.logger.Warnf("HTTPSDialerStatsManager.OnSuccess: not found: %+v", tactic)
		return
	}

	// update stats
	record.CountSuccess++
	record.LastUpdated = time.Now()
}

// Close implements io.Closer
func (mt *statsManager) Close() error {
	// TODO(bassosimone): do we need to apply a "once" semantics to this method?

	// get exclusive access
	defer mt.mu.Unlock()
	mt.mu.Lock()

	// write updated stats into the underlying key-value store
	return mt.kvStore.Set(statsKey, runtimex.Try1(json.Marshal(mt.container)))
}
