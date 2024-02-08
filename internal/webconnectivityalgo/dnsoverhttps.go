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

// TODO(bassosimone): consider unifying somehow this code and
// the systemresolver code (or maybe just the list of resolvers)

// OpportunisticDNSOverHTTPSURLProvider allows to perform opportunistic DNS-over-HTTPS
// measurements as part of Web Connectivity LTE. The zero value of this struct is not valid,
// please use [NewOpportunisticDNSOverHTTPSURLProvider] to construct.
//
// Implementation note: this code uses memory to keep track of the resolvers and know
// when to perform the next opportunistic check. It seems pointless to use the disk since
// invocations of Web Connectivity typically consist of multiple URLs and therefore run
// for a few minutes. Hence, storing state on disk seems a bit overkill here.
type OpportunisticDNSOverHTTPSURLProvider struct {
	// interval is the next interval after which to measure.
	interval time.Duration

	// mu provides mutual exclusion
	mu *sync.Mutex

	// rnd is the random number generator to use.
	rnd *rand.Rand

	// t is when we last run an opportunistic measurement.
	t time.Time

	// timeNow is the function to get the current time.
	timeNow func() time.Time

	// urls contains the urls of known DoH services.
	urls []string
}

// NewOpportunisticDNSOverHTTPSURLProvider creates a new [*OpportunisticDNSOverHTTPSURLProvider].
func NewOpportunisticDNSOverHTTPSURLProvider(urls ...string) *OpportunisticDNSOverHTTPSURLProvider {
	o := &OpportunisticDNSOverHTTPSURLProvider{
		interval: 0,
		mu:       &sync.Mutex{},
		rnd:      nil, // configured below
		t:        time.Time{},
		timeNow:  time.Now,
		urls:     urls,
	}
	o.seed(o.timeNow()) // allow unit tests to reconfigure the seed we use
	return o
}

func (o *OpportunisticDNSOverHTTPSURLProvider) seed(t time.Time) {
	o.rnd = rand.New(rand.NewSource(t.UnixNano()))
}

// MaybeNextURL returns the next URL to measure, if any. Our aim is to perform
// periodic, opportunistic DoH measurements as part of Web Connectivity.
func (o *OpportunisticDNSOverHTTPSURLProvider) MaybeNextURL() (string, bool) {
	// obtain the current time
	now := o.timeNow()

	// make sure there's mutual exclusion
	o.mu.Lock()
	defer o.mu.Unlock()

	// Make sure we run periodically but now always, since there is no point in
	// always using DNS-over-HTTPS rather the aim is to opportunistically try using
	// it so to collect data on whether it's actually WAI.
	if len(o.urls) > 0 && (o.t.IsZero() || now.Sub(o.t) > o.interval) {

		// shuffle the list according to the selected random profile
		o.rnd.Shuffle(len(o.urls), func(i, j int) {
			o.urls[i], o.urls[j] = o.urls[j], o.urls[i]
		})

		// register the current invocation and remember to run again later
		o.t = now
		o.interval = time.Duration(20+o.rnd.Uint32()%20) * time.Second

		// return the selected URL to the caller
		return o.urls[0], true
	}

	return "", false
}
