package engineresolver

//
// Persistent on-disk state
//

import (
	"errors"
	"sort"
)

// TODO(bassosimone): we may want to change the key and rename or
// remove the old file inside the statedir

// storekey is the key used by the key value store to store
// the state required by this package.
const storekey = "sessionresolver.state"

// resolverinfo contains info about a resolver.
type resolverinfo struct {
	// URL is the URL of a resolver.
	URL string

	// Score is the score of a resolver.
	Score float64
}

// ErrNilKVStore indicates that the KVStore is nil.
var ErrNilKVStore = errors.New("sessionresolver: kvstore is nil")

// readstate reads the resolver state from disk
func (r *Resolver) readstate() ([]*resolverinfo, error) {
	if r.KVStore == nil {
		return nil, ErrNilKVStore
	}
	data, err := r.KVStore.Get(storekey)
	if err != nil {
		return nil, err
	}
	var ri []*resolverinfo
	if err := r.codec().Decode(data, &ri); err != nil {
		return nil, err
	}
	return ri, nil
}

// errNoEntries indicates that no entry remained after we pruned
// all the available entries in readstateandprune.
var errNoEntries = errors.New("sessionresolver: no available entries")

// readstateandprune reads the state from disk and removes all the
// entries that we don't actually support.
func (r *Resolver) readstateandprune() ([]*resolverinfo, error) {
	ri, err := r.readstate()
	if err != nil {
		return nil, err
	}
	var out []*resolverinfo
	for _, e := range ri {
		if _, found := allbyurl[e.URL]; !found {
			continue // we don't support this specific entry
		}
		out = append(out, e)
	}
	if len(out) <= 0 {
		return nil, errNoEntries
	}
	return out, nil
}

// sortstate sorts the state by descending score
func sortstate(ri []*resolverinfo) {
	sort.SliceStable(ri, func(i, j int) bool {
		return ri[i].Score >= ri[j].Score
	})
}

// readstatedefault reads the state from disk and merges the state
// so that all supported entries are represented.
func (r *Resolver) readstatedefault() []*resolverinfo {
	ri, _ := r.readstateandprune()
	here := make(map[string]bool)
	for _, e := range ri {
		here[e.URL] = true // record what we already have
	}
	for _, e := range allmakers {
		if _, found := here[e.url]; found {
			continue // already here so no need to add
		}
		ri = append(ri, &resolverinfo{
			URL:   e.url,
			Score: e.score,
		})
	}
	sortstate(ri)
	return ri
}

// writestate writes the state to the kvstore.
func (r *Resolver) writestate(ri []*resolverinfo) error {
	if r.KVStore == nil {
		return ErrNilKVStore
	}
	data, err := r.codec().Encode(ri)
	if err != nil {
		return err
	}
	return r.KVStore.Set(storekey, data)
}
