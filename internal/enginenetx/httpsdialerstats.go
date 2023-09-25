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

// HTTPSDialerStatsTacticRecord keeps stats about an [HTTPSDialerTactic].
type HTTPSDialerStatsTacticRecord struct {
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

// HTTPSDialerStatsTacticsContainer contains tactics.
type HTTPSDialerStatsTacticsContainer struct {
	// Tactic maps the summary of a tactic to the tactic record.
	Tactics map[string]*HTTPSDialerStatsTacticRecord
}

// HTTPSDialerStatsContainerVersion is the current version of [HTTPSDialerStatsContainer].
const HTTPSDialerStatsContainerVersion = 2

// HTTPSDialerStatsRootContainer is the root container for stats.
//
// The zero value is invalid; construct using [NewHTTPSDialerStatsRootContainer].
type HTTPSDialerStatsRootContainer struct {
	// Domains maps a domain name to its tactics
	Domains map[string]*HTTPSDialerStatsTacticsContainer

	// Version is the version of the container data format.
	Version int
}

// Get returns the tactic record for the given [*HTTPSDialerTactic] instance.
//
// At the name implies, this function MUST be called while holding the [HTTPSDialerStatsManager] mutex.
func (c *HTTPSDialerStatsRootContainer) GetLocked(tactic *HTTPSDialerTactic) (*HTTPSDialerStatsTacticRecord, bool) {
	domainRecord, found := c.Domains[tactic.VerifyHostname]
	if !found {
		return nil, false
	}
	tacticRecord, found := domainRecord.Tactics[tactic.Summary()]
	return tacticRecord, found
}

// Set sets the tactic record for the given the given [*HTTPSDialerTactic] instance.
//
// At the name implies, this function MUST be called while holding the [HTTPSDialerStatsManager] mutex.
func (c *HTTPSDialerStatsRootContainer) SetLocked(tactic *HTTPSDialerTactic, record *HTTPSDialerStatsTacticRecord) {
	domainRecord, found := c.Domains[tactic.VerifyHostname]
	if !found {
		domainRecord = &HTTPSDialerStatsTacticsContainer{
			Tactics: map[string]*HTTPSDialerStatsTacticRecord{},
		}

		// make sure the map is initialized
		if len(c.Domains) <= 0 {
			c.Domains = make(map[string]*HTTPSDialerStatsTacticsContainer)
		}

		c.Domains[tactic.VerifyHostname] = domainRecord
		// fallthrough
	}
	domainRecord.Tactics[tactic.Summary()] = record
}

// NewHTTPSDialerStatsRootContainer creates a new empty [*HTTPSDialerStatsRootContainer].
func NewHTTPSDialerStatsRootContainer() *HTTPSDialerStatsRootContainer {
	return &HTTPSDialerStatsRootContainer{
		Domains: map[string]*HTTPSDialerStatsTacticsContainer{},
		Version: HTTPSDialerStatsContainerVersion,
	}
}

// HTTPSDialerStatsManager implements [HTTPSDialerStatsTracker] by storing
// the relevant statistics in a [model.KeyValueStore].
//
// The zero value of this structure is not ready to use; please, use the
// [NewHTTPSDialerStatsManager] factory to create a new instance.
type HTTPSDialerStatsManager struct {
	// kvStore is the key-value store we're using
	kvStore model.KeyValueStore

	// logger is the logger to use.
	logger model.Logger

	// mu provides mutual exclusion when accessing the stats.
	mu sync.Mutex

	// root is the root container for stats
	root *HTTPSDialerStatsRootContainer
}

// HTTPSDialerStatsKey is the key used in the key-value store to access the state.
const HTTPSDialerStatsKey = "httpsdialerstats.state"

// errDialerStatsContainerWrongVersion means that the stats container document has the wrong version number.
var errDialerStatsContainerWrongVersion = errors.New("wrong stats container version")

// loadHTTPSDialerStatsRootContainer loads a state container from the given key-value store.
func loadHTTPSDialerStatsRootContainer(kvStore model.KeyValueStore) (*HTTPSDialerStatsRootContainer, error) {
	// load data from the kvstore
	data, err := kvStore.Get(HTTPSDialerStatsKey)
	if err != nil {
		return nil, err
	}

	// parse as JSON
	var container HTTPSDialerStatsRootContainer
	if err := json.Unmarshal(data, &container); err != nil {
		return nil, err
	}

	// make sure the version is OK
	if container.Version != HTTPSDialerStatsContainerVersion {
		err := fmt.Errorf(
			"%s: %w: expected=%d got=%d",
			HTTPSDialerStatsKey,
			errDialerStatsContainerWrongVersion,
			HTTPSDialerStatsContainerVersion,
			container.Version,
		)
		return nil, err
	}

	return &container, nil
}

// NewHTTPSDialerStatsManager constructs a new instance of [*HTTPSDialerStatsManager].
func NewHTTPSDialerStatsManager(kvStore model.KeyValueStore, logger model.Logger) *HTTPSDialerStatsManager {
	root, err := loadHTTPSDialerStatsRootContainer(kvStore)
	if err != nil {
		root = NewHTTPSDialerStatsRootContainer()
	}

	return &HTTPSDialerStatsManager{
		root:    root,
		kvStore: kvStore,
		logger:  logger,
		mu:      sync.Mutex{},
	}
}

var _ HTTPSDialerStatsTracker = &HTTPSDialerStatsManager{}

// OnStarting implements HTTPSDialerStatsManager.
func (mt *HTTPSDialerStatsManager) OnStarting(tactic *HTTPSDialerTactic) {
	// get exclusive access
	defer mt.mu.Unlock()
	mt.mu.Lock()

	// get the record
	record, found := mt.root.GetLocked(tactic)
	if !found {
		record = &HTTPSDialerStatsTacticRecord{
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
func (mt *HTTPSDialerStatsManager) OnTCPConnectError(ctx context.Context, tactic *HTTPSDialerTactic, err error) {
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
func (mt *HTTPSDialerStatsManager) OnTLSHandshakeError(ctx context.Context, tactic *HTTPSDialerTactic, err error) {
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
func (mt *HTTPSDialerStatsManager) OnTLSVerifyError(tactic *HTTPSDialerTactic, err error) {
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
func (mt *HTTPSDialerStatsManager) OnSuccess(tactic *HTTPSDialerTactic) {
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
func (mt *HTTPSDialerStatsManager) Close() error {
	// TODO(bassosimone): do we need to apply a "once" semantics to this method?

	// get exclusive access
	defer mt.mu.Unlock()
	mt.mu.Lock()

	// write updated stats into the underlying key-value store
	return mt.kvStore.Set(HTTPSDialerStatsKey, runtimex.Try1(json.Marshal(mt.root)))
}
