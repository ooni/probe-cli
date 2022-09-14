package webconnectivity

//
// DNSResolvers
//
// Generated by `boilerplate' using the multi-resolver template.
//

import (
	"context"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/atomicx"
	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// Resolves the URL's domain using several resolvers.
//
// The zero value of this structure IS NOT valid and you MUST initialize
// all the fields marked as MANDATORY before using this structure.
type DNSResolvers struct {
	// DNSCache is the MANDATORY DNS cache.
	DNSCache *DNSCache

	// Domain is the MANDATORY domain to resolve.
	Domain string

	// IDGenerator is the MANDATORY atomic int64 to generate task IDs.
	IDGenerator *atomicx.Int64

	// Logger is the MANDATORY logger to use.
	Logger model.Logger

	// TestKeys is MANDATORY and contains the TestKeys.
	TestKeys *TestKeys

	// URL is the MANDATORY URL we're measuring.
	URL *url.URL

	// ZeroTime is the MANDATORY zero time of the measurement.
	ZeroTime time.Time

	// WaitGroup is the MANDATORY wait group this task belongs to.
	WaitGroup *sync.WaitGroup

	// CookieJar contains the OPTIONAL cookie jar, used for redirects.
	CookieJar http.CookieJar

	// Referer contains the OPTIONAL referer, used for redirects.
	Referer string

	// Session is the OPTIONAL session. If the session is set, we will use
	// it to start the task that issues the control request. This request must
	// only be sent during the first iteration. It would be pointless to
	// issue such a request for subsequent redirects, because the TH will
	// always follow the redirect chain caused by the provided URL.
	Session model.ExperimentSession

	// THAddr is the OPTIONAL test helper address.
	THAddr string

	// UDPAddress is the OPTIONAL address of the UDP resolver to use. If this
	// field is not set we use a default one (e.g., `8.8.8.8:53`).
	UDPAddress string
}

// Start starts this task in a background goroutine.
func (t *DNSResolvers) Start(ctx context.Context) {
	t.WaitGroup.Add(1)
	go func() {
		defer t.WaitGroup.Done() // synchronize with the parent
		t.Run(ctx)
	}()
}

// run performs a DNS lookup and returns the looked up addrs
func (t *DNSResolvers) run(parentCtx context.Context) []DNSEntry {
	// create output channels for the lookup
	systemOut := make(chan []string)
	udpOut := make(chan []string)
	httpsOut := make(chan []string)
	whoamiSystemV4Out := make(chan []DNSWhoamiInfoEntry)
	whoamiUDPv4Out := make(chan []DNSWhoamiInfoEntry)

	// TODO(bassosimone): add opportunistic support for detecting
	// whether DNS queries are answered regardless of dest addr by
	// sending a few queries to root DNS servers

	udpAddress := t.udpAddress()

	// start asynchronous lookups
	go t.lookupHostSystem(parentCtx, systemOut)
	go t.lookupHostUDP(parentCtx, udpAddress, udpOut)
	go t.lookupHostDNSOverHTTPS(parentCtx, httpsOut)
	go t.whoamiSystemV4(parentCtx, whoamiSystemV4Out)
	go t.whoamiUDPv4(parentCtx, udpAddress, whoamiUDPv4Out)

	// collect resulting IP addresses (which may be nil/empty lists)
	systemAddrs := <-systemOut
	udpAddrs := <-udpOut
	httpsAddrs := <-httpsOut

	// collect whoami results (which also may be nil/empty)
	whoamiSystemV4 := <-whoamiSystemV4Out
	whoamiUDPv4 := <-whoamiUDPv4Out
	t.TestKeys.WithDNSWhoami(func(di *DNSWhoamiInfo) {
		di.SystemV4 = whoamiSystemV4
		di.UDPv4[udpAddress] = whoamiUDPv4
	})

	// merge the resolved IP addresses
	merged := map[string]*DNSEntry{}
	for _, addr := range systemAddrs {
		if _, found := merged[addr]; !found {
			merged[addr] = &DNSEntry{}
		}
		merged[addr].Addr = addr
		merged[addr].Flags |= DNSAddrFlagSystemResolver
	}
	for _, addr := range udpAddrs {
		if _, found := merged[addr]; !found {
			merged[addr] = &DNSEntry{}
		}
		merged[addr].Addr = addr
		merged[addr].Flags |= DNSAddrFlagUDP
	}
	for _, addr := range httpsAddrs {
		if _, found := merged[addr]; !found {
			merged[addr] = &DNSEntry{}
		}
		merged[addr].Addr = addr
		merged[addr].Flags |= DNSAddrFlagHTTPS
	}
	// implementation note: we don't remove bogons because accessing
	// them can lead us to discover block pages
	var entries []DNSEntry
	for _, entry := range merged {
		entries = append(entries, *entry)
	}

	return entries
}

// Run runs this task in the current goroutine.
func (t *DNSResolvers) Run(parentCtx context.Context) {
	var (
		addresses []DNSEntry
		found     bool
	)

	// attempt to use the dns cache
	addresses, found = t.DNSCache.Get(t.Domain)

	if !found {
		// fall back to performing a real dns lookup
		addresses = t.run(parentCtx)

		// insert the addresses we just looked us into the cache
		t.DNSCache.Set(t.Domain, addresses)

		log.Infof("using resolved addrs: %+v", addresses)
	} else {
		log.Infof("using previously-cached addrs: %+v", addresses)
	}

	// create priority selector
	ps := newPrioritySelector(parentCtx, t.ZeroTime, t.TestKeys, t.Logger, addresses)

	// fan out a number of child async tasks to use the IP addrs
	t.startCleartextFlows(parentCtx, ps, addresses)
	t.startSecureFlows(parentCtx, ps, addresses)
	t.maybeStartControlFlow(parentCtx, ps, addresses)
}

// whoamiSystemV4 performs a DNS whoami lookup for the system resolver. This function must
// always emit an ouput on the [out] channel to synchronize with the caller func.
func (t *DNSResolvers) whoamiSystemV4(parentCtx context.Context, out chan<- []DNSWhoamiInfoEntry) {
	value, _ := DNSWhoamiSingleton.SystemV4(parentCtx)
	t.Logger.Infof("DNS whoami for system resolver: %+v", value)
	out <- value
}

// whoamiUDPv4 performs a DNS whoami lookup for the given UDP resolver. This function must
// always emit an ouput on the [out] channel to synchronize with the caller func.
func (t *DNSResolvers) whoamiUDPv4(parentCtx context.Context, udpAddress string, out chan<- []DNSWhoamiInfoEntry) {
	value, _ := DNSWhoamiSingleton.UDPv4(parentCtx, udpAddress)
	t.Logger.Infof("DNS whoami for %s/udp resolver: %+v", udpAddress, value)
	out <- value
}

// lookupHostSystem performs a DNS lookup using the system resolver. This function must
// always emit an ouput on the [out] channel to synchronize with the caller func.
func (t *DNSResolvers) lookupHostSystem(parentCtx context.Context, out chan<- []string) {
	// create context with attached a timeout
	const timeout = 4 * time.Second
	lookupCtx, lookpCancel := context.WithTimeout(parentCtx, timeout)
	defer lookpCancel()

	// create trace's index
	index := t.IDGenerator.Add(1)

	// create trace
	trace := measurexlite.NewTrace(index, t.ZeroTime)

	// start the operation logger
	ol := measurexlite.NewOperationLogger(
		t.Logger, "[#%d] lookup %s using system", index, t.Domain,
	)

	// runs the lookup
	reso := trace.NewStdlibResolver(t.Logger)
	addrs, err := reso.LookupHost(lookupCtx, t.Domain)
	t.TestKeys.AppendQueries(trace.DNSLookupsFromRoundTrip()...)
	ol.Stop(err)
	out <- addrs
}

// lookupHostUDP performs a DNS lookup using an UDP resolver. This function must always
// emit an ouput on the [out] channel to synchronize with the caller func.
func (t *DNSResolvers) lookupHostUDP(parentCtx context.Context, udpAddress string, out chan<- []string) {
	// create context with attached a timeout
	const timeout = 4 * time.Second
	lookupCtx, lookpCancel := context.WithTimeout(parentCtx, timeout)
	defer lookpCancel()

	// create trace's index
	index := t.IDGenerator.Add(1)

	// create trace
	trace := measurexlite.NewTrace(index, t.ZeroTime)

	// start the operation logger
	ol := measurexlite.NewOperationLogger(
		t.Logger, "[#%d] lookup %s using %s", index, t.Domain, udpAddress,
	)

	// runs the lookup
	dialer := netxlite.NewDialerWithoutResolver(t.Logger)
	reso := trace.NewParallelUDPResolver(t.Logger, dialer, udpAddress)
	addrs, err := reso.LookupHost(lookupCtx, t.Domain)

	// saves the results making sure we split Do53 queries from other queries
	do53, other := t.do53SplitQueries(trace.DNSLookupsFromRoundTrip())
	t.TestKeys.AppendQueries(do53...)
	t.TestKeys.WithTestKeysDo53(func(tkd *TestKeysDo53) {
		tkd.Queries = append(tkd.Queries, other...)
		tkd.NetworkEvents = append(tkd.NetworkEvents, trace.NetworkEvents()...)
	})

	ol.Stop(err)
	out <- addrs

	// wait for late DNS replies
	t.WaitGroup.Add(1)
	go t.waitForLateReplies(parentCtx, trace)
}

// Waits for late DNS replies.
func (t *DNSResolvers) waitForLateReplies(parentCtx context.Context, trace *measurexlite.Trace) {
	defer t.WaitGroup.Done()
	const lateTimeout = 500 * time.Millisecond
	events := trace.DelayedDNSResponseWithTimeout(parentCtx, lateTimeout)
	if length := len(events); length > 0 {
		t.Logger.Warnf("got %d late DNS replies", length)
	}
	t.TestKeys.AppendDNSLateReplies(events...)
}

// Divides queries generated by Do53 in Do53-proper queries and other queries.
func (t *DNSResolvers) do53SplitQueries(
	input []*model.ArchivalDNSLookupResult) (do53, other []*model.ArchivalDNSLookupResult) {
	for _, query := range input {
		switch query.Engine {
		case "udp", "tcp":
			do53 = append(do53, query)
		default:
			other = append(other, query)
		}
	}
	return
}

// TODO(bassosimone): maybe cycle through a bunch of well known addresses

// Returns the UDP resolver we should be using by default.
func (t *DNSResolvers) udpAddress() string {
	if t.UDPAddress != "" {
		return t.UDPAddress
	}
	return "8.8.4.4:53"
}

// OpportunisticDNSOverHTTPS allows to perform opportunistic DNS-over-HTTPS
// measurements as part of Web Connectivity.
type OpportunisticDNSOverHTTPS struct {
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

// MaybeNextURL returns the next URL to measure, if any. Our aim is to perform
// periodic, opportunistic DoH measurements as part of Web Connectivity.
func (o *OpportunisticDNSOverHTTPS) MaybeNextURL() (string, bool) {
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

// TODO(bassosimone): consider whether factoring out this code
// and storing the state on disk instead of using memory

// TODO(bassosimone): consider unifying somehow this code and
// the systemresolver code (or maybe just the list of resolvers)

// OpportunisticDNSOverHTTPSSingleton is the singleton used to keep
// track of the opportunistic DNS-over-HTTPS measurements state.
var OpportunisticDNSOverHTTPSSingleton = &OpportunisticDNSOverHTTPS{
	interval: 0,
	mu:       &sync.Mutex{},
	rnd:      rand.New(rand.NewSource(time.Now().UnixNano())),
	t:        time.Time{},
	urls: []string{
		"https://mozilla.cloudflare-dns.com/dns-query",
		"https://dns.nextdns.io/dns-query",
		"https://dns.google/dns-query",
		"https://dns.quad9.net/dns-query",
	},
}

// lookupHostDNSOverHTTPS performs a DNS lookup using a DoH resolver. This function must
// always emit an ouput on the [out] channel to synchronize with the caller func.
func (t *DNSResolvers) lookupHostDNSOverHTTPS(parentCtx context.Context, out chan<- []string) {
	// obtain an opportunistic DoH URL
	URL, good := OpportunisticDNSOverHTTPSSingleton.MaybeNextURL()
	if !good {
		// no need to perform opportunistic DoH at this time but we still
		// need to fake out a lookup to please our caller
		out <- []string{}
		return
	}

	// create context with attached a timeout
	const timeout = 4 * time.Second
	lookupCtx, lookpCancel := context.WithTimeout(parentCtx, timeout)
	defer lookpCancel()

	// create trace's index
	index := t.IDGenerator.Add(1)

	// create trace
	trace := measurexlite.NewTrace(index, t.ZeroTime)

	// start the operation logger
	ol := measurexlite.NewOperationLogger(
		t.Logger, "[#%d] lookup %s using %s", index, t.Domain, URL,
	)

	// runs the lookup
	reso := trace.NewParallelDNSOverHTTPSResolver(t.Logger, URL)
	addrs, err := reso.LookupHost(lookupCtx, t.Domain)
	reso.CloseIdleConnections()

	// save results making sure we properly split DoH queries from other queries
	doh, other := t.dohSplitQueries(trace.DNSLookupsFromRoundTrip())
	t.TestKeys.AppendQueries(doh...)
	t.TestKeys.WithTestKeysDoH(func(tkdh *TestKeysDoH) {
		tkdh.Queries = append(tkdh.Queries, other...)
		tkdh.NetworkEvents = append(tkdh.NetworkEvents, trace.NetworkEvents()...)
		tkdh.TCPConnect = append(tkdh.TCPConnect, trace.TCPConnects()...)
		tkdh.TLSHandshakes = append(tkdh.TLSHandshakes, trace.TLSHandshakes()...)
	})

	ol.Stop(err)
	out <- addrs
}

// Divides queries generated by DoH in DoH-proper queries and other queries.
func (t *DNSResolvers) dohSplitQueries(
	input []*model.ArchivalDNSLookupResult) (doh, other []*model.ArchivalDNSLookupResult) {
	for _, query := range input {
		switch query.Engine {
		case "doh":
			doh = append(doh, query)
		default:
			other = append(other, query)
		}
	}
	return
}

// startCleartextFlows starts a TCP measurement flow for each IP addr.
func (t *DNSResolvers) startCleartextFlows(
	ctx context.Context,
	ps *prioritySelector,
	addresses []DNSEntry,
) {
	if t.URL.Scheme != "http" {
		// Do not bother with measuring HTTP when the user
		// has asked us to measure an HTTPS URL.
		return
	}
	port := "80"
	if urlPort := t.URL.Port(); urlPort != "" {
		port = urlPort
	}
	for _, addr := range addresses {
		task := &CleartextFlow{
			Address:         net.JoinHostPort(addr.Addr, port),
			DNSCache:        t.DNSCache,
			IDGenerator:     t.IDGenerator,
			Logger:          t.Logger,
			TestKeys:        t.TestKeys,
			ZeroTime:        t.ZeroTime,
			WaitGroup:       t.WaitGroup,
			CookieJar:       t.CookieJar,
			FollowRedirects: t.URL.Scheme == "http",
			HostHeader:      t.URL.Host,
			PrioSelector:    ps,
			Referer:         t.Referer,
			UDPAddress:      t.UDPAddress,
			URLPath:         t.URL.Path,
			URLRawQuery:     t.URL.RawQuery,
		}
		task.Start(ctx)
	}
}

// startSecureFlows starts a TCP+TLS measurement flow for each IP addr.
func (t *DNSResolvers) startSecureFlows(
	ctx context.Context,
	ps *prioritySelector,
	addresses []DNSEntry,
) {
	if t.URL.Scheme != "https" {
		// When the scheme is not HTTPS we fetch using HTTP
		ps = nil
	}
	port := "443"
	if urlPort := t.URL.Port(); urlPort != "" {
		if t.URL.Scheme != "https" {
			// If the URL is like http://example.com:8080/, we don't know
			// which would be the correct port where to use HTTPS.
			return
		}
		port = urlPort
	}
	for _, addr := range addresses {
		task := &SecureFlow{
			Address:         net.JoinHostPort(addr.Addr, port),
			DNSCache:        t.DNSCache,
			IDGenerator:     t.IDGenerator,
			Logger:          t.Logger,
			TestKeys:        t.TestKeys,
			ZeroTime:        t.ZeroTime,
			WaitGroup:       t.WaitGroup,
			ALPN:            []string{"h2", "http/1.1"},
			CookieJar:       t.CookieJar,
			FollowRedirects: t.URL.Scheme == "https",
			SNI:             t.URL.Hostname(),
			HostHeader:      t.URL.Host,
			PrioSelector:    ps,
			Referer:         t.Referer,
			UDPAddress:      t.UDPAddress,
			URLPath:         t.URL.Path,
			URLRawQuery:     t.URL.RawQuery,
		}
		task.Start(ctx)
	}
}

// maybeStartControlFlow starts the control flow iff .Session and .THAddr are set.
func (t *DNSResolvers) maybeStartControlFlow(
	ctx context.Context,
	ps *prioritySelector,
	addresses []DNSEntry,
) {
	// note: for subsequent requests we don't set .Session and .THAddr hence
	// we are not going to query the test helper more than once
	if t.Session != nil && t.THAddr != "" {
		var addrs []string
		for _, addr := range addresses {
			addrs = append(addrs, addr.Addr)
		}
		ctrl := &Control{
			Addresses:                addrs,
			ExtraMeasurementsStarter: t, // allows starting follow-up measurement flows
			Logger:                   t.Logger,
			PrioSelector:             ps,
			TestKeys:                 t.TestKeys,
			Session:                  t.Session,
			THAddr:                   t.THAddr,
			URL:                      t.URL,
			WaitGroup:                t.WaitGroup,
		}
		ctrl.Start(ctx)
	}
}
