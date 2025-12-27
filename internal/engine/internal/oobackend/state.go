package oobackend

import (
	"errors"
	"math/rand"
	"sort"
	"sync"
	"time"
)

// storekey is the key used by the key value store to store
// the state required by this package.
const storekey = "oobackend.state"

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

// strategyState contains the strategy state.
type strategyState struct {
	NextReset  time.Time
	Strategies []*strategyInfo
}

// readstate reads the strategies state from disk
func (c *Client) readstate() (*strategyState, error) {
	data, err := c.KVStore.Get(storekey)
	if err != nil {
		return nil, err
	}
	var ss strategyState
	if err := c.jsonCodec().Decode(data, &ss); err != nil {
		return nil, err
	}
	return &ss, nil
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
func (c *Client) readstateandprune() (*strategyState, error) {
	ss, err := c.readstate()
	if err != nil {
		return nil, err
	}
	out := strategyState{NextReset: ss.NextReset}
	for _, e := range ss.Strategies {
		if _, found := allStrategies[e.URL]; !found {
			continue // we don't support this specific entry
		}
		out.Strategies = append(out.Strategies, e)
	}
	if len(out.Strategies) <= 0 {
		return nil, errNoEntries
	}
	return &out, nil
}

// sortstate sorts state by descending score.
func (c *Client) sortstate(ss *strategyState) {
	sort.SliceStable(ss.Strategies, func(i, j int) bool {
		return ss.Strategies[i].Score >= ss.Strategies[j].Score
	})
}

// readstatedefault reads the state from disk and merges the state
// so that all supported entries are represented.
func (c *Client) readstatedefault() *strategyState {
	ss, _ := c.readstateandprune()
	here := make(map[string]bool)
	for _, e := range ss.Strategies {
		here[e.URL] = true // record what we already have
	}
	for url, score := range allStrategies {
		if _, found := here[url]; found {
			continue // already here so no need to add
		}
		ss.Strategies = append(ss.Strategies, &strategyInfo{
			URL:   url,
			Score: score,
		})
	}
	c.sortstate(ss)
	return ss
}

// readstatemaybedefault is the top-level function for reading strategies
// where we reset to default values with low probability.
func (c *Client) readstatemaybedefault(seed int64) *strategyState {
	ss := c.readstatedefault()
	now := time.Now()
	if ss.NextReset.IsZero() || now.After(ss.NextReset) {
		out := &strategyState{NextReset: now.Add(c.nextwait(seed))}
		for url, score := range allStrategies {
			out.Strategies = append(out.Strategies, &strategyInfo{
				URL:   url,
				Score: score,
			})
		}
		return out
	}
	return ss
}

// nextwait computes the next wait time.
func (c *Client) nextwait(seed int64) time.Duration {
	rng := rand.New(rand.NewSource(seed))
	const mean = 10 * time.Second
	const low = mean / 10  // 10% of mean
	const high = mean * 25 // 250% of mean
	delta := time.Duration(rng.ExpFloat64() * float64(mean))
	if delta <= low {
		return low
	}
	if delta >= high {
		return high
	}
	return delta
}

// writestate writes the state on the kvstore.
func (c *Client) writestate(ss *strategyState) error {
	data, err := c.jsonCodec().Encode(ss)
	if err != nil {
		return err
	}
	return c.KVStore.Set(storekey, data)
}
