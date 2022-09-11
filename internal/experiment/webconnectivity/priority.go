package webconnectivity

// prioritySelector selects the connection with the highest priority.
type prioritySelector struct {
	// c is the channel used to select priority
	c chan any

	// m contains a map from known addresses to their flags
	m map[string]int64
}

// newPrioritySelector creates a new prioritySelector instance.
func newPrioritySelector(addrs []DNSEntry) *prioritySelector {
	c := make(chan any)
	c <- true // give a single goroutine permission to fetch the body
	ps := &prioritySelector{
		c: c,
		m: map[string]int64{},
	}
	for _, addr := range addrs {
		ps.m[addr.Addr] = addr.Flags
	}
	return ps
}

// permissionToFetch returns whether this ready-to-use connection
// is permitted to perform a round trip and fetch the webpage.
func (ps *prioritySelector) permissionToFetch(address string) bool {
	flags := ps.m[address]
	if (flags & DNSAddrFlagSystemResolver) == 0 {
		return false // see https://github.com/ooni/probe/issues/2258
	}
	select {
	case <-ps.c:
		return true
	default:
		return false
	}
}
