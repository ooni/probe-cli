package oobackend

import (
	"errors"
	"math/rand"
	"sort"
	"sync"
)

// storekey is the key used by the key value store to store
// the state required by this package.
const storekey = "oobackend.state"

// TODO(bassosimone): exponential delay between resets?

// strategyInfo contains info about a strategy.
type strategyInfo struct {
	// URL is the URL of a strategy.
	URL string

	// Score is the score of a strategy.
	Score float64

	// mu protects this structure.
	mu sync.Mutex
}

// updatescore updates the score with the most recent result.
func (si *strategyInfo) updatescore(err error) {
	const param = 0.9 // derivative!!!
	v := func() float64 {
		switch err {
		case nil:
			return 1
		default:
			return 0
		}
	}()
	defer si.mu.Unlock()
	si.mu.Lock()
	si.Score = v*param + si.Score*(1-param)
}

// readstate reads the strategies state from disk
func (c *Client) readstate() ([]*strategyInfo, error) {
	data, err := c.KVStore.Get(storekey)
	if err != nil {
		return nil, err
	}
	var si []*strategyInfo
	if err := c.jsonCodec().Decode(data, &si); err != nil {
		return nil, err
	}
	return si, nil
}

// errNoEntries indicates that no entry remained after we pruned
// all the available entries in readstateandprune.
var errNoEntries = errors.New("sessionresolver: no available entries")

// allStrategies contains all the known strategies along
// with their default score.
var allStrategies = map[string]float64{
	"https://ps1.ooni.io/":                 1,
	"https://ps2.ooni.io/":                 0.75,
	"https://dkyhjv0wpi2dk.cloudfront.net": 0.25,
	"tunnel://psiphon/":                    0,
}

// readstateandprune reads the state from disk and removes all the
// entries that we don't actually support.
func (c *Client) readstateandprune() ([]*strategyInfo, error) {
	si, err := c.readstate()
	if err != nil {
		return nil, err
	}
	var out []*strategyInfo
	for _, e := range si {
		if _, found := allStrategies[e.URL]; !found {
			continue // we don't support this specific entry
		}
		out = append(out, e)
	}
	if len(out) <= 0 {
		return nil, errNoEntries
	}
	return out, nil
}

// sortstate sorts state by descending score.
func (c *Client) sortstate(si []*strategyInfo) {
	sort.SliceStable(si, func(i, j int) bool {
		return si[i].Score >= si[j].Score
	})
}

// readstatedefault reads the state from disk and merges the state
// so that all supported entries are represented.
func (c *Client) readstatedefault() []*strategyInfo {
	si, _ := c.readstateandprune()
	here := make(map[string]bool)
	for _, e := range si {
		here[e.URL] = true // record what we already have
	}
	for url, score := range allStrategies {
		if _, found := here[url]; found {
			continue // already here so no need to add
		}
		si = append(si, &strategyInfo{
			URL:   url,
			Score: score,
		})
	}
	c.sortstate(si)
	return si
}

// readstatemaybedefault is the top-level function for reading strategies
// where we reset to default values with low probability.
func (c *Client) readstatemaybedefault(seed int64) []*strategyInfo {
	rng := rand.New(rand.NewSource(seed))
	const lowProbability = 0.1
	if rng.Float64() <= lowProbability {
		var out []*strategyInfo
		for url, score := range allStrategies {
			out = append(out, &strategyInfo{
				URL:   url,
				Score: score,
			})
		}
		return out
	}
	return c.readstatedefault()
}

// writestate writes the state on the kvstore.
func (c *Client) writestate(ri []*strategyInfo) error {
	data, err := c.jsonCodec().Encode(ri)
	if err != nil {
		return err
	}
	return c.KVStore.Set(storekey, data)
}
