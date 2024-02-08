package webconnectivityalgo

//
// DNS-over-HTTPS
//
// Code to manage DNS-over-HTTPS testing.
//

import (
	"math/rand"
	"sync"
	"time"
)

// TODO(bassosimone): consider whether factoring out this code
// and storing the state on disk instead of using memory

// TODO(bassosimone): consider unifying somehow this code and
// the systemresolver code (or maybe just the list of resolvers)

// OpportunisticDNSOverHTTPSURLProvider allows to perform opportunistic DNS-over-HTTPS
// measurements as part of Web Connectivity LTE. The zero value of this struct is not valid,
// please use [NewOpportunisticDNSOverHTTPSURLProvider] to construct.
type OpportunisticDNSOverHTTPSURLProvider struct {
	// interval is the next interval after which to measure.
	interval time.Duration

	// mu provides mutual exclusion
	mu *sync.Mutex

	// rnd is the random number generator to use.
	rnd *rand.Rand

	// t is when we last run an opportunistic measurement.
	t time.Time

	// urls contains the urls of known DoH services.
	urls []string
}

// NewOpportunisticDNSOverHTTPSURLProvider creates a new [*OpportunisticDNSOverHTTPSURLProvider].
func NewOpportunisticDNSOverHTTPSURLProvider(urls ...string) *OpportunisticDNSOverHTTPSURLProvider {
	return &OpportunisticDNSOverHTTPSURLProvider{
		interval: 0,
		mu:       &sync.Mutex{},
		rnd:      rand.New(rand.NewSource(time.Now().UnixNano())),
		t:        time.Time{},
		urls:     urls,
	}
}

// MaybeNextURL returns the next URL to measure, if any. Our aim is to perform
// periodic, opportunistic DoH measurements as part of Web Connectivity.
func (o *OpportunisticDNSOverHTTPSURLProvider) MaybeNextURL() (string, bool) {
	now := time.Now()
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.t.IsZero() || now.Sub(o.t) > o.interval {
		o.rnd.Shuffle(len(o.urls), func(i, j int) {
			o.urls[i], o.urls[j] = o.urls[j], o.urls[i]
		})
		o.t = now
		o.interval = time.Duration(20+o.rnd.Uint32()%20) * time.Second
		return o.urls[0], true
	}
	return "", false
}
