package loader

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/obfuscate"
)

// LoadTor loads and returns the tor [*ExperimentSpec].
func (c *Client) LoadTor(ctx context.Context, pi *ProbeInfo) (*ExperimentSpec, error) {
	// refresh the tor targets cache using an orchestra token
	err := c.accountCallWithToken(ctx, func(token string) error {
		return c.refreshTorTargetsCache(ctx, pi, token)
	})

	// handle the error case
	if err != nil {
		return nil, err
	}

	// because we have a fresh targets cache
	spec := &ExperimentSpec{
		Name: "tor",
		Targets: []ExperimentTarget{{
			Options:      map[string]any{},
			Input:        "",
			CategoryCode: "MISC",
			CountryCode:  "ZZ",
		}},
	}
	return spec, nil
}

// torTargetsCacheKey is the key used to store tor targets.
const torTargetsCacheKey = "tortargets.state"

type torTargetsCache struct {
	Expire  time.Time
	Targets map[string]model.OOAPITorTarget
}

func (ttc *torTargetsCache) didExpire() bool {
	return time.Now().After(ttc.Expire)
}

func loadTorTargetsCache(store model.KeyValueStore) (*torTargetsCache, error) {
	data, err := store.Get(torTargetsCacheKey)
	if err != nil {
		return nil, err
	}

	data = obfuscate.Apply(data)

	var cache torTargetsCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}

	return &cache, nil
}

func storeTorTargetsCache(store model.KeyValueStore, cache *torTargetsCache) error {
	data, err := json.Marshal(cache)
	if err != nil {
		return err
	}

	data = obfuscate.Apply(data)

	return store.Set(torTargetsCacheKey, data)
}

// refreshTorTargetsCache ensures the tor targets cache is fresh.
func (c *Client) refreshTorTargetsCache(ctx context.Context, pi *ProbeInfo, token string) error {
	// load from the cache
	cache, err := loadTorTargetsCache(c.store)

	// if there's no error and the cache did not expire, we're good
	if err == nil && !cache.didExpire() {
		return nil
	}

	// otherwise fetch from the backend
	targets, err := c.fetchTorTargets(ctx, pi, token)
	if err != nil {
		return err
	}

	// store in cache
	cache = &torTargetsCache{
		Expire:  time.Now().Add(72 * time.Hour),
		Targets: targets,
	}
	return storeTorTargetsCache(c.store, cache)
}

func (c *Client) fetchTorTargets(ctx context.Context, pi *ProbeInfo, token string) (map[string]model.OOAPITorTarget, error) {
	// create the query string
	query := url.Values{}
	query.Add("country_code", pi.ProbeCC)

	// create the URL
	URL := &url.URL{
		Scheme:   "https",
		Host:     c.endpoint,
		Path:     "/api/v1/test-list/tor-targets",
		RawQuery: query.Encode(),
	}

	// create the request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, URL.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	// perform the round trip
	resp, err := c.txp.RoundTrip(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// handle HTTP request failures
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, errUnauthorized
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %d %s", ErrHTTPFailure, resp.StatusCode, resp.Status)
	}

	// read the response body
	rawRespBody, err := netxlite.ReadAllContext(ctx, resp.Body)
	if err != nil {
		return nil, err
	}

	// parse tor targets
	var targets map[string]model.OOAPITorTarget
	if err := json.Unmarshal(rawRespBody, &targets); err != nil {
		return nil, err
	}

	return targets, nil
}
