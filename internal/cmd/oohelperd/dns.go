package main

//
// DNS measurements
//

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/tracex"
)

// newfailure is a convenience shortcut to save typing
var newfailure = tracex.NewFailure

// ctrlDNSResult is the result of the DNS check performed by
// the Web Connectivity test helper.
type ctrlDNSResult = model.THDNSResult

// dnsConfig configures the DNS check.
type dnsConfig struct {
	// Cache is the MANDATORY cache to use.
	Cache model.KeyValueStore

	// Domain is the MANDATORY domain to resolve.
	Domain string

	// Logger is the MANDATORY logger to use.
	Logger model.Logger

	// NewResolver is the MANDATORY factory to create a new resolver.
	NewResolver func(model.Logger) model.Resolver

	// Out is the channel where we publish the results.
	Out chan ctrlDNSResult

	// Wg allows to synchronize with the parent.
	Wg *sync.WaitGroup
}

// dnsCacheKey is the key used inside the DNS cache.
type dnsCacheKey string

// newDNSCacheKey creates a new dnsCacheKey
func newDNSCacheKey(config *dnsConfig) dnsCacheKey {
	return dnsCacheKey(config.Domain)
}

// asCacheKeyString returns the string used by the underlying cache as key.
func (tck dnsCacheKey) asCacheKeyString() string {
	return string(tck)
}

// dnsCacheEntry is an entry inside the DNS cache.
type dnsCacheEntry struct {
	// Created is when we created this entry.
	Created time.Time

	// Key is the domain we've resolved.
	Key dnsCacheKey

	// Result is the cached result.
	Result ctrlDNSResult
}

// dnsCacheGet gets a list of results from the DNS cache key.
func dnsCacheGet(cache model.KeyValueStore, key dnsCacheKey) ([]*dnsCacheEntry, error) {
	rawdata, err := cache.Get(key.asCacheKeyString())
	if err != nil {
		return nil, err
	}
	var values []*dnsCacheEntry
	if err := json.Unmarshal(rawdata, &values); err != nil {
		return nil, err
	}
	const dnsCacheExpirationTime = 15 * time.Minute
	var out []*dnsCacheEntry
	for _, value := range values {
		if value == nil || time.Since(value.Created) >= dnsCacheExpirationTime {
			continue // this entry is malformed or has expired
		}
		out = append(out, value)
	}
	return out, nil
}

// dnsCacheEntriesFind searches for a given domain inside a set of entries.
func dnsCacheEntriesFind(epv []*dnsCacheEntry, key dnsCacheKey) (*dnsCacheEntry, bool) {
	for _, ep := range epv {
		if ep != nil && key == ep.Key {
			return ep, true
		}
	}
	return nil, false
}

// dnsCacheWriteBack writes back into the cache.
func dnsCacheWriteBack(cache model.KeyValueStore, key dnsCacheKey, epv []*dnsCacheEntry) error {
	rawdata, err := json.Marshal(epv)
	if err != nil {
		return err
	}
	return cache.Set(key.asCacheKeyString(), rawdata)
}

// dnsDo performs the DNS check.
func dnsDo(ctx context.Context, config *dnsConfig) {
	defer config.Wg.Done()
	key := newDNSCacheKey(config)
	entries, _ := dnsCacheGet(config.Cache, key) // the error is not so relevant
	entry, _ := dnsCacheEntriesFind(entries, key)
	if entry == nil {
		entry = &dnsCacheEntry{
			Created: time.Now(),
			Key:     key,
			Result:  dnsDoWithoutCache(ctx, config),
		}
		entries = append(entries, entry)
	}
	config.Out <- entry.Result
	_ = dnsCacheWriteBack(config.Cache, key, entries)
}

// dnsDoWithoutCache implements dnsDo.
func dnsDoWithoutCache(ctx context.Context, config *dnsConfig) ctrlDNSResult {
	const timeout = 4 * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	reso := config.NewResolver(config.Logger)
	defer reso.CloseIdleConnections()
	ol := measurexlite.NewOperationLogger(config.Logger, "DNSLookup %s", config.Domain)
	addrs, err := reso.LookupHost(ctx, config.Domain)
	ol.Stop(err)
	if addrs == nil {
		addrs = []string{} // fix: the old test helper did that
	}
	failure := dnsMapFailure(newfailure(err))
	return ctrlDNSResult{
		Failure: failure,
		Addrs:   addrs,
		ASNs:    []int64{}, // unused by the TH and not serialized
	}
}

// dnsMapFailure attempts to map netxlite failures to the strings
// used by the original OONI test helper.
//
// See https://github.com/ooni/backend/blob/6ec4fda5b18/oonib/testhelpers/http_helpers.py#L430
func dnsMapFailure(failure *string) *string {
	switch failure {
	case nil:
		return nil
	default:
		switch *failure {
		case netxlite.FailureDNSNXDOMAINError:
			// We have a name for this string because dnsanalysis.go is
			// already checking for this specific error string.
			s := model.THDNSNameError
			return &s
		case netxlite.FailureDNSNoAnswer:
			// In this case the legacy TH would produce an empty
			// reply that is not attached to any error.
			//
			// See https://github.com/ooni/probe/issues/1707#issuecomment-944322725
			return nil
		case netxlite.FailureDNSNonRecoverableFailure,
			netxlite.FailureDNSRefusedError,
			netxlite.FailureDNSServerMisbehaving,
			netxlite.FailureDNSTemporaryFailure:
			s := "dns_server_failure"
			return &s
		default:
			s := "unknown_error"
			return &s
		}
	}
}
