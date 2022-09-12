package main

//
// HTTP measurements
//

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/tracex"
)

// TODO(bassosimone): we should refactor the TH to use step-by-step such that we
// can use an existing connection for the HTTP-measuring task

// ctrlHTTPResponse is the result of the HTTP check performed by
// the Web Connectivity test helper.
type ctrlHTTPResponse = model.THHTTPRequestResult

// httpConfig configures the HTTP check.
type httpConfig struct {
	// Cache is the MANDATORY HTTP cache to use.
	Cache model.KeyValueStore

	// Headers is OPTIONAL and contains the request headers we should set.
	Headers map[string][]string

	// Logger is the MANDATORY logger to use.
	Logger model.Logger

	// MaxAcceptableBody is MANDATORY and specifies the maximum acceptable body size.
	MaxAcceptableBody int64

	// NewClient is the MANDATORY factory to create a new client.
	NewClient func(model.Logger) model.HTTPClient

	// Out is the MANDATORY channel where we'll post results.
	Out chan ctrlHTTPResponse

	// URL is the MANDATORY URL to measure.
	URL string

	// Wg is MANDATORY and allows synchronizing with parent.
	Wg *sync.WaitGroup
}

// httpCacheKey is the key used by the HTTP cache
type httpCacheKey struct {
	// Accept is the value of the accept header.
	Accept string

	// AcceptLanguage is the value of the accept-language header.
	AcceptLanguage string

	// MaxAcceptableBody is the maximum acceptable body size.
	MaxAcceptableBody int64

	// URL is the MANDATORY URL to measure.
	URL string

	// UserAgent is the value of the user-agent header.
	UserAgent string
}

// newHTTPCacheKey creates a new httpCacheKey from the given [config].
func newHTTPCacheKey(config *httpConfig) *httpCacheKey {
	headers := http.Header(config.Headers)
	if headers == nil {
		headers = http.Header{}
	}
	return &httpCacheKey{
		Accept:            headers.Get("accept"),
		AcceptLanguage:    headers.Get("accept-language"),
		MaxAcceptableBody: config.MaxAcceptableBody,
		URL:               config.URL,
		UserAgent:         headers.Get("user-agent"),
	}
}

// asCacheKeyString returns the string used by the underlying cache as key.
func (tck *httpCacheKey) asCacheKeyString() string {
	return fmt.Sprintf("%+v", tck)
}

// Equals returns whether two instances are equal
func (tck *httpCacheKey) Equals(other *httpCacheKey) bool {
	return tck.asCacheKeyString() == other.asCacheKeyString()
}

// httpCacheEntry is an entry inside the HTTP cache.
type httpCacheEntry struct {
	// Created is when we created this entry.
	Created time.Time

	// Key identifies this cache entry.
	Key httpCacheKey

	// Result is the cached result.
	Result ctrlHTTPResponse
}

// httpCacheGet gets a list of results from the HTTP cache key.
func httpCacheGet(cache model.KeyValueStore, key *httpCacheKey) ([]*httpCacheEntry, error) {
	rawdata, err := cache.Get(key.asCacheKeyString())
	if err != nil {
		return nil, err
	}
	var values []*httpCacheEntry
	if err := json.Unmarshal(rawdata, &values); err != nil {
		return nil, err
	}
	const tcpCacheExpirationTime = 15 * time.Minute
	var out []*httpCacheEntry
	for _, value := range values {
		if value == nil || time.Since(value.Created) >= tcpCacheExpirationTime {
			continue // this entry is malformed or has expired
		}
		out = append(out, value)
	}
	return out, nil
}

// httpCacheEntriesFind searches for a given domain inside a set of entries.
func httpCacheEntriesFind(epv []*httpCacheEntry, key *httpCacheKey) (*httpCacheEntry, bool) {
	for _, ep := range epv {
		if ep != nil && key.Equals(&ep.Key) {
			return ep, true
		}
	}
	return nil, false
}

// httpCacheWriteBack writes back into the cache.
func httpCacheWriteBack(cache model.KeyValueStore, key *httpCacheKey, epv []*httpCacheEntry) error {
	rawdata, err := json.Marshal(epv)
	if err != nil {
		return err
	}
	return cache.Set(key.asCacheKeyString(), rawdata)
}

// httpDo performs the HTTP check.
func httpDo(ctx context.Context, config *httpConfig) {
	defer config.Wg.Done()
	key := newHTTPCacheKey(config)
	entries, _ := httpCacheGet(config.Cache, key) // the error is not so relevant
	entry, _ := httpCacheEntriesFind(entries, key)
	if entry == nil {
		entry = &httpCacheEntry{
			Created: time.Now(),
			Key:     *key,
			Result:  httpDoWithoutCache(ctx, config),
		}
		entries = append(entries, entry)
	}
	config.Out <- entry.Result
	_ = httpCacheWriteBack(config.Cache, key, entries)
}

// httpDoWithoutCache implements httpDo
func httpDoWithoutCache(ctx context.Context, config *httpConfig) ctrlHTTPResponse {
	ol := measurexlite.NewOperationLogger(config.Logger, "GET %s", config.URL)
	const timeout = 15 * time.Second
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "GET", config.URL, nil)
	if err != nil {
		ol.Stop(err)
		// fix: emit -1 like the old test helper does
		return ctrlHTTPResponse{
			BodyLength: -1,
			Failure:    httpMapFailure(err),
			Title:      "",
			Headers:    map[string]string{},
			StatusCode: -1,
		}
	}
	// The original test helper failed with extra headers while here
	// we're implementing (for now?) a more liberal approach.
	for k, vs := range config.Headers {
		switch strings.ToLower(k) {
		// WARNING: if you enable more headers here then you must modify
		// the caching code to use thise headers for the cache key
		case "user-agent", "accept", "accept-language":
			for _, v := range vs {
				req.Header.Add(k, v)
			}
		}
	}
	clnt := config.NewClient(config.Logger)
	defer clnt.CloseIdleConnections()
	resp, err := clnt.Do(req)
	if err != nil {
		ol.Stop(err)
		// fix: emit -1 like the old test helper does
		return ctrlHTTPResponse{
			BodyLength: -1,
			Failure:    httpMapFailure(err),
			Title:      "",
			Headers:    map[string]string{},
			StatusCode: -1,
		}
	}
	defer resp.Body.Close()
	headers := make(map[string]string)
	for k := range resp.Header {
		headers[k] = resp.Header.Get(k)
	}
	reader := &io.LimitedReader{R: resp.Body, N: config.MaxAcceptableBody}
	data, err := netxlite.ReadAllContext(ctx, reader)
	ol.Stop(err)
	return ctrlHTTPResponse{
		BodyLength: int64(len(data)),
		Failure:    httpMapFailure(err),
		StatusCode: int64(resp.StatusCode),
		Headers:    headers,
		Title:      measurexlite.WebGetTitle(string(data)),
	}
}

// httpMapFailure attempts to map netxlite failures to the strings
// used by the original OONI test helper.
//
// See https://github.com/ooni/backend/blob/6ec4fda5b18/oonib/testhelpers/http_helpers.py#L361
func httpMapFailure(err error) *string {
	failure := newfailure(err)
	failedOperation := tracex.NewFailedOperation(err)
	switch failure {
	case nil:
		return nil
	default:
		switch *failure {
		case netxlite.FailureDNSNXDOMAINError,
			netxlite.FailureDNSNoAnswer,
			netxlite.FailureDNSNonRecoverableFailure,
			netxlite.FailureDNSRefusedError,
			netxlite.FailureDNSServerMisbehaving,
			netxlite.FailureDNSTemporaryFailure:
			// Strangely the HTTP code uses the more broad
			// dns_lookup_error and does not check for
			// the NXDOMAIN-equivalent-error dns_name_error
			s := "dns_lookup_error"
			return &s
		case netxlite.FailureGenericTimeoutError:
			// The old TH would return "dns_lookup_error" when
			// there is a timeout error during the DNS phase of HTTP.
			switch failedOperation {
			case nil:
				// nothing
			default:
				switch *failedOperation {
				case netxlite.ResolveOperation:
					s := "dns_lookup_error"
					return &s
				}
			}
			return failure // already using the same name
		case netxlite.FailureConnectionRefused:
			s := "connection_refused_error"
			return &s
		default:
			s := "unknown_error"
			return &s
		}
	}
}
