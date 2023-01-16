package webconnectivitylte

//
// Determine which connection(s) are allowed to fetch the webpage
// by giving higher priority to the system resolver, then to the
// UDP resolver, then to the DoH resolver, then to the TH.
//
// This sorting reflects the likelyhood that we will se a blockpage
// because the system resolver is the most likely to be blocked
// (e.g., in Italy). The UDP resolver is also blocked in countries
// with more censorship (e.g., in China). The DoH resolver and
// the TH have more or less the same priority here, but we needed
// to choose one of them to have higher priority.
//
// Note that this functionality is where Web Connectivity LTE
// diverges from websteps, which will instead fetch all the available
// webpages. To adhere to the Web Connectivity model, we need to
// have a single fetch per redirect. However, by allowing all the
// resolvers plus the TH to provide us with addresses, we increase
// our chances of detecting more kinds of censorship.
//
// Also note that this implementation basically makes the
// https://github.com/ooni/probe/issues/2258 issue obsolete,
// since now we're considering all resolvers.
//
// See https://github.com/ooni/probe/issues/2276.
//

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// errNotPermittedToFetch indicates we're not permitted to fetch a webpage
// because another goroutine have already fetched that webpage.
var errNotPermittedToFetch = errors.New("webconnectivity: not permitted to fetch")

// prioritySelector selects the connection with the highest priority.
type prioritySelector struct {
	// ch is the channel used to ask for priority
	ch chan *priorityRequest

	// logger is the logger to use
	logger model.Logger

	// m contains a map from known addresses to their flags
	m map[string]int64

	// nhttps is the number of addrs resolved using DoH
	nhttps int

	// nsystem is the number of addrs resolved using the system resolver
	nsystem int

	// nudp is the nunber of addrs resolver using UDP
	nudp int

	// tk contains the TestKeys.
	tk *TestKeys

	// zeroTime is the zero time of the current measurement
	zeroTime time.Time
}

// priorityRequest is a request to get priority for fetching the webpage
// over other concurrent connections that are doing the same.
type priorityRequest struct {
	// addr is the address we're using
	addr string

	// resp is the buffered channel where the response will appear
	resp chan bool
}

// newPrioritySelector creates a new prioritySelector instance.
func newPrioritySelector(
	ctx context.Context,
	zeroTime time.Time,
	tk *TestKeys,
	logger model.Logger,
	addrs []DNSEntry,
) *prioritySelector {
	ps := &prioritySelector{
		ch:       make(chan *priorityRequest),
		logger:   logger,
		m:        map[string]int64{},
		nhttps:   0,
		nsystem:  0,
		nudp:     0,
		tk:       tk,
		zeroTime: zeroTime,
	}
	ps.log("create with %+v", addrs)
	for _, addr := range addrs {
		flags := addr.Flags
		ps.m[addr.Addr] = flags
		if (flags & DNSAddrFlagSystemResolver) != 0 {
			ps.nsystem++
		}
		if (flags & DNSAddrFlagUDP) != 0 {
			ps.nudp++
		}
		if (flags & DNSAddrFlagHTTPS) != 0 {
			ps.nhttps++
		}
	}
	go ps.selector(ctx)
	return ps
}

// log formats and emits a ConnPriorityLogEntry
func (ps *prioritySelector) log(format string, v ...any) {
	ps.tk.AppendConnPriorityLogEntry(&ConnPriorityLogEntry{
		Msg: fmt.Sprintf(format, v...),
		T:   time.Since(ps.zeroTime).Seconds(),
	})
	ps.logger.Infof("prioritySelector: "+format, v...)
}

// permissionToFetch returns whether this ready-to-use connection
// is permitted to perform a round trip and fetch the webpage.
func (ps *prioritySelector) permissionToFetch(address string) bool {
	ipAddr, _, err := net.SplitHostPort(address)
	runtimex.PanicOnError(err, "net.SplitHostPort failed")
	r := &priorityRequest{
		addr: ipAddr,
		resp: make(chan bool, 1), // buffer to simplify selector() implementation
	}
	select {
	case <-time.After(10 * time.Millisecond):
		ps.log("conn %s: denied permission: timed out sending", address)
		return false
	case ps.ch <- r:
		select {
		case <-time.After(time.Second):
			ps.log("conn %s: denied permission: timed out receiving", address)
			return false
		case v := <-r.resp:
			ps.log("conn %s: granted permission: %+v", address, v)
			return v
		}
	}
}

// selector grants permission to the highest priority request that
// arrives within a reasonable time frame. This function runs into the
// background goroutine and terminates when [ctx] is done.
//
// This function implements https://github.com/ooni/probe/issues/2276.
func (ps *prioritySelector) selector(ctx context.Context) {
	// Implementation note: setting an arbitrary timeout here would
	// be ~an issue because we want this goroutine to be available in
	// case the only connections from which we could fetch a webpage
	// are the ones using TH addresses. However, we know the TH could
	// require a long time to complete due to timeouts caused by IP
	// addresses provided by the probe.
	//
	// See https://explorer.ooni.org/measurement/20220911T105037Z_webconnectivity_IT_30722_n1_ruzuQ219SmIO9SrT?input=http%3A%2F%2Festrenosli.org%2F
	// for a measurement where a too-short timeout prevented us from
	// attempting to fetch a webpage from TH-resolved addrs.
	//
	// See https://explorer.ooni.org/measurement/20220911T194527Z_webconnectivity_IT_30722_n1_jufRZGay0Db9Ge4v?input=http%3A%2F%2Festrenosli.org%2F
	// for a measurement where this issue was fixed.

	// await the first priority request
	var first *priorityRequest
	select {
	case <-ctx.Done():
		return
	case first = <-ps.ch:
	}

	// if this request is highest priority, grant permission
	if ps.isHighestPriority(first) {
		first.resp <- true // buffered channel
		return
	}

	// collect additional requests for up to extraTime, thus giving
	// a possibly higher priority connection time to establish
	const extraTime = 500 * time.Millisecond
	expired := time.NewTimer(extraTime)
	defer expired.Stop()
	requests := []*priorityRequest{first}
Loop:
	for {
		select {
		case <-expired.C:
			break Loop
		case r := <-ps.ch:
			requests = append(requests, r)
		case <-ctx.Done():
			return
		}
	}

	// grant permission to the highest priority request
	highPrio := ps.findHighestPriority(requests)
	highPrio.resp <- true // buffered channel

	// deny permission to all the other inflight requests
	for _, r := range requests {
		if highPrio != r {
			r.resp <- false // buffered channel
		}
	}
}

// findHighestPriority returns the highest priority request
func (ps *prioritySelector) findHighestPriority(reqs []*priorityRequest) *priorityRequest {
	runtimex.Assert(len(reqs) > 0, "findHighestPriority wants a non-empty reqs slice")
	for _, r := range reqs {
		if ps.isHighestPriority(r) {
			return r
		}
	}
	return reqs[0]
}

// isHighestPriority returns whether this request is highest priority
func (ps *prioritySelector) isHighestPriority(r *priorityRequest) bool {
	// See https://github.com/ooni/probe/issues/2276
	flags := ps.m[r.addr]
	if ps.nsystem > 0 {
		if (flags & DNSAddrFlagSystemResolver) != 0 {
			return true
		}
	} else if ps.nudp > 0 {
		if (flags & DNSAddrFlagUDP) != 0 {
			return true
		}
	} else if ps.nhttps > 0 {
		if (flags & DNSAddrFlagHTTPS) != 0 {
			return true
		}
	} else {
		// Happens when we only have addresses from the TH
		return true
	}
	return false
}
