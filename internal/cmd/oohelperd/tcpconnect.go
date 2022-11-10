package main

//
// TCP connect (and optionally TLS handshake) measurements
//

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// ctrlTCPResult is the result of the TCP check performed by the test helper.
type ctrlTCPResult = model.THTCPConnectResult

// ctrlTLSResult is the result of the TLS check performed by the test helper.
type ctrlTLSResult = model.THTLSHandshakeResult

// tcpResultPair contains the endpoint and the corresponding result.
type tcpResultPair struct {
	// Address is the IP address we measured.
	Address string

	// Endpoint is the endpoint we measured.
	Endpoint string

	// TCP contains the TCP results.
	TCP ctrlTCPResult

	// TLS contains the TLS results
	TLS *ctrlTLSResult
}

// tcpConfig configures the TCP connect check.
type tcpConfig struct {
	// Address is the MANDATORY address to measure.
	Address string

	// Cache is the MANDATORY TCP cache to use.
	Cache model.KeyValueStore

	// EnableTLS OPTIONALLY enables TLS.
	EnableTLS bool

	// Endpoint is the MANDATORY endpoint to connect to.
	Endpoint string

	// Logger is the MANDATORY logger to use.
	Logger model.Logger

	// NewDialer is the MANDATORY factory for creating a new dialer.
	NewDialer func(model.Logger) model.Dialer

	// NewTSLHandshaker is the MANDATORY factory for creating a new handshaker.
	NewTSLHandshaker func(model.Logger) model.TLSHandshaker

	// Out is the MANDATORY where we'll post the TCP measurement results.
	Out chan *tcpResultPair

	// URLHostname is the MANDATORY URL.Hostname() to use.
	URLHostname string

	// Wg is MANDATORY and is used to sync with the parent.
	Wg *sync.WaitGroup
}

// tcpCacheKey is the key used by the TCP cache
type tcpCacheKey struct {
	// EnableTLS OPTIONALLY enables TLS.
	EnableTLS bool

	// Endpoint is the MANDATORY endpoint to connect to.
	Endpoint string

	// URLHostname is the MANDATORY URL.Hostname() to use.
	URLHostname string
}

// newTCPCacheKey creates a new tcpCacheKey from the given [config].
func newTCPCacheKey(config *tcpConfig) *tcpCacheKey {
	return &tcpCacheKey{
		EnableTLS:   config.EnableTLS,
		Endpoint:    config.Endpoint,
		URLHostname: config.URLHostname,
	}
}

// asCacheKeyString returns the string used by the underlying cache as key.
func (tck *tcpCacheKey) asCacheKeyString() string {
	return fmt.Sprintf("%+v", tck)
}

// Equals returns whether two instances are equal
func (tck *tcpCacheKey) Equals(other *tcpCacheKey) bool {
	return tck.EnableTLS == other.EnableTLS && tck.Endpoint == other.Endpoint &&
		tck.URLHostname == other.URLHostname
}

// tcpCacheEntry is an entry inside the TCP cache.
type tcpCacheEntry struct {
	// Created is when we created this entry.
	Created time.Time

	// Key identifies this cache entry.
	Key tcpCacheKey

	// Result is the cached result.
	Result *tcpResultPair
}

// tcpCacheGet gets a list of results from the TCP cache key.
func tcpCacheGet(cache model.KeyValueStore, key *tcpCacheKey) ([]*tcpCacheEntry, error) {
	rawdata, err := cache.Get(key.asCacheKeyString())
	if err != nil {
		return nil, err
	}
	var values []*tcpCacheEntry
	if err := json.Unmarshal(rawdata, &values); err != nil {
		return nil, err
	}
	const tcpCacheExpirationTime = 15 * time.Minute
	var out []*tcpCacheEntry
	for _, value := range values {
		if value == nil || value.Result == nil || time.Since(value.Created) >= tcpCacheExpirationTime {
			continue // this entry is malformed or has expired
		}
		out = append(out, value)
	}
	return out, nil
}

// tcpCacheEntriesFind searches for a given domain inside a set of entries.
func tcpCacheEntriesFind(epv []*tcpCacheEntry, key *tcpCacheKey) (*tcpCacheEntry, bool) {
	for _, ep := range epv {
		if ep != nil && key.Equals(&ep.Key) {
			return ep, true
		}
	}
	return nil, false
}

// tcpCacheWriteBack writes back into the cache.
func tcpCacheWriteBack(cache model.KeyValueStore, key *tcpCacheKey, epv []*tcpCacheEntry) error {
	rawdata, err := json.Marshal(epv)
	if err != nil {
		return err
	}
	return cache.Set(key.asCacheKeyString(), rawdata)
}

// tcpDo performs the TCP check.
func tcpDo(ctx context.Context, config *tcpConfig) {
	defer config.Wg.Done()
	key := newTCPCacheKey(config)
	entries, _ := tcpCacheGet(config.Cache, key) // the error is not so relevant
	entry, _ := tcpCacheEntriesFind(entries, key)
	if entry == nil {
		entry = &tcpCacheEntry{
			Created: time.Now(),
			Key:     *key,
			Result:  tcpDoWithoutCache(ctx, config),
		}
		entries = append(entries, entry)
	}
	config.Out <- entry.Result
	_ = tcpCacheWriteBack(config.Cache, key, entries)
}

// tcpDoWithoutCache implements tcpDo
func tcpDoWithoutCache(ctx context.Context, config *tcpConfig) *tcpResultPair {
	const timeout = 15 * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	out := &tcpResultPair{
		Address:  config.Address,
		Endpoint: config.Endpoint,
		TCP:      model.THTCPConnectResult{},
		TLS:      nil, // means: not measured
	}
	ol := measurexlite.NewOperationLogger(
		config.Logger,
		"TCPConnect %s EnableTLS=%v SNI=%s",
		config.Endpoint,
		config.EnableTLS,
		config.URLHostname,
	)
	dialer := config.NewDialer(config.Logger)
	defer dialer.CloseIdleConnections()
	conn, err := dialer.DialContext(ctx, "tcp", config.Endpoint)
	out.TCP.Failure = tcpMapFailure(newfailure(err))
	out.TCP.Status = err == nil
	defer measurexlite.MaybeClose(conn)
	if err != nil || !config.EnableTLS {
		ol.Stop(err)
		return out
	}
	tlsConfig := &tls.Config{
		NextProtos: []string{"h2", "http/1.1"},
		RootCAs:    netxlite.NewDefaultCertPool(),
		ServerName: config.URLHostname,
	}
	thx := config.NewTSLHandshaker(config.Logger)
	tlsConn, _, err := thx.Handshake(ctx, conn, tlsConfig)
	ol.Stop(err)
	out.TLS = &ctrlTLSResult{
		ServerName: config.URLHostname,
		Status:     err == nil,
		Failure:    newfailure(err),
	}
	measurexlite.MaybeClose(tlsConn)
	return out
}

// tcpMapFailure attempts to map netxlite failures to the strings
// used by the original OONI test helper.
//
// See https://github.com/ooni/backend/blob/6ec4fda5b18/oonib/testhelpers/http_helpers.py#L392
func tcpMapFailure(failure *string) *string {
	switch failure {
	case nil:
		return nil
	default:
		switch *failure {
		case netxlite.FailureGenericTimeoutError:
			return failure // already using the same name
		case netxlite.FailureConnectionRefused:
			s := "connection_refused_error"
			return &s
		default:
			// The definition of this error according to Twisted is
			// "something went wrong when connecting". Because we are
			// indeed basically just connecting here, it seems safe
			// to map any other error to "connect_error" here.
			s := "connect_error"
			return &s
		}
	}
}
