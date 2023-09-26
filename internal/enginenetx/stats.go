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

// statsDomain contains stats associated with a domain.
type statsDomain struct {
	Tactics map[string]*statsTactic
}

// statsContainerVersion is the current version of [statsContainer].
const statsContainerVersion = 2

// statsContainer is the root container for stats.
//
// The zero value is invalid; construct using [newStatsContainer].
type statsContainer struct {
	// Domains maps a domain name to its tactics
	Domains map[string]*statsDomain

	// Version is the version of the container data format.
	Version int
}

// Get returns the tactic record for the given [*statsTactic] instance.
//
// At the name implies, this function MUST be called while holding the [*statsManager] mutex.
func (c *statsContainer) GetLocked(tactic *HTTPSDialerTactic) (*statsTactic, bool) {
	domainRecord, found := c.Domains[tactic.VerifyHostname]
	if !found {
		return nil, false
	}
	tacticRecord, found := domainRecord.Tactics[tactic.Summary()]
	return tacticRecord, found
}

// Set sets the tactic record for the given the given [*statsTactic] instance.
//
// At the name implies, this function MUST be called while holding the [*statsManager] mutex.
func (c *statsContainer) SetLocked(tactic *HTTPSDialerTactic, record *statsTactic) {
	domainRecord, found := c.Domains[tactic.VerifyHostname]
	if !found {
		domainRecord = &statsDomain{
			Tactics: map[string]*statsTactic{},
		}

		// make sure the map is initialized
		if len(c.Domains) <= 0 {
			c.Domains = make(map[string]*statsDomain)
		}

		c.Domains[tactic.VerifyHostname] = domainRecord
		// fallthrough
	}
	domainRecord.Tactics[tactic.Summary()] = record
}

// newStatsContainer creates a new empty [*statsContainer].
func newStatsContainer() *statsContainer {
	return &statsContainer{
		Domains: map[string]*statsDomain{},
		Version: statsContainerVersion,
	}
}

// statsManager implements [HTTPSDialerStatsTracker] by storing
// the relevant statistics in a [model.KeyValueStore].
//
// The zero value of this structure is not ready to use; please, use the
// [newStatsManager] factory to create a new instance.
type statsManager struct {
	// kvStore is the key-value store we're using
	kvStore model.KeyValueStore

	// logger is the logger to use.
	logger model.Logger

	// mu provides mutual exclusion when accessing the stats.
	mu sync.Mutex

	// root is the root container for stats
	root *statsContainer
}

// statsKey is the key used in the key-value store to access the state.
const statsKey = "httpsdialerstats.state"

// errStatsContainerWrongVersion means that the stats container document has the wrong version number.
var errStatsContainerWrongVersion = errors.New("wrong stats container version")

// loadStatsContainer loads a state container from the given key-value store.
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

	return &container, nil
}

// newStatsManager constructs a new instance of [*statsManager].
func newStatsManager(kvStore model.KeyValueStore, logger model.Logger) *statsManager {
	root, err := loadStatsContainer(kvStore)
	if err != nil {
		root = newStatsContainer()
	}

	return &statsManager{
		root:    root,
		kvStore: kvStore,
		logger:  logger,
		mu:      sync.Mutex{},
	}
}

var _ HTTPSDialerStatsTracker = &statsManager{}

// OnStarting implements HTTPSDialerStatsManager.
func (mt *statsManager) OnStarting(tactic *HTTPSDialerTactic) {
	// get exclusive access
	defer mt.mu.Unlock()
	mt.mu.Lock()

	// get the record
	record, found := mt.root.GetLocked(tactic)
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
		mt.root.SetLocked(tactic, record)
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
	record, found := mt.root.GetLocked(tactic)
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
	record, found := mt.root.GetLocked(tactic)
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
	record, found := mt.root.GetLocked(tactic)
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
	record, found := mt.root.GetLocked(tactic)
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
	return mt.kvStore.Set(statsKey, runtimex.Try1(json.Marshal(mt.root)))
}
