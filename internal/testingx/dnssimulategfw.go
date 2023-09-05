package testingx

import (
	"errors"
	"net"
	"sync"

	"github.com/ooni/netem"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// DNSNumBogusResponses is a type indicating the number of bogus responses
// the [DNSSimulateGWFListener] should emit for each round trip.
type DNSNumBogusResponses int

// DNSSimulateGWFListener is a DNS-over-UDP listener that simulates the GFW behavior by
// responding with three answers, where the first two answers are invalid for the domain
// and the last answer is correct for the domain. The zero value of this struct is
// invalid, please use [NewDNSSimulateGWFListener].
type DNSSimulateGWFListener struct {
	bogusConfig *netem.DNSConfig
	closeOnce   sync.Once
	goodConfig  *netem.DNSConfig
	numBogus    DNSNumBogusResponses
	pconn       net.PacketConn
	wg          sync.WaitGroup
}

// MustNewDNSSimulateGWFListener creates a new [DNSSimulateGWFListener] using the given
// [DNSOverUDPUnderlyingListener], [*net.UDPAddr], and [*netem.DNSConfig]. The bogusConfig
// is used to prepare the bogus responses, and the good config is used to prepare the
// final response containing valid IP addresses for the domain. If numBogusResponses is
// less or equal than 1, we will force its value to be 1.
func MustNewDNSSimulateGWFListener(
	addr *net.UDPAddr,
	dul DNSOverUDPUnderlyingListener,
	bogusConfig *netem.DNSConfig,
	goodConfig *netem.DNSConfig,
	numBogusResponses DNSNumBogusResponses,
) *DNSSimulateGWFListener {
	pconn := runtimex.Try1(dul.ListenUDP("udp", addr))
	if numBogusResponses < 1 {
		numBogusResponses = 1 // as documented
	}
	dl := &DNSSimulateGWFListener{
		bogusConfig: bogusConfig,
		closeOnce:   sync.Once{},
		goodConfig:  goodConfig,
		numBogus:    numBogusResponses,
		pconn:       pconn,
		wg:          sync.WaitGroup{},
	}
	dl.wg.Add(1)
	go dl.mainloop()
	return dl
}

// LocalAddr returns the connection address. The return value is nil after you called Close.
func (dl *DNSSimulateGWFListener) LocalAddr() net.Addr {
	return dl.pconn.LocalAddr()
}

// Close implements io.Closer.
func (dl *DNSSimulateGWFListener) Close() (err error) {
	dl.closeOnce.Do(func() {
		// close the connection to interrupt ReadFrom or WriteTo
		err = dl.pconn.Close()

		// wait for the background goroutine to join
		dl.wg.Wait()
	})
	return err
}

func (dl *DNSSimulateGWFListener) mainloop() {
	// synchronize with Close
	defer dl.wg.Done()

	for {
		// read from the socket
		buffer := make([]byte, 1<<17)
		count, addr, err := dl.pconn.ReadFrom(buffer)

		// handle errors including the case in which we're closed
		if errors.Is(err, net.ErrClosed) {
			return
		}
		if err != nil {
			continue
		}

		// prepare the raw request for the round tripper
		rawReq := buffer[:count]

		// emit N >= 1 bogus responses followed by a valid response
		for idx := DNSNumBogusResponses(0); idx < dl.numBogus; idx++ {
			dl.writeResponse(addr, dl.bogusConfig, rawReq)
		}
		dl.writeResponse(addr, dl.goodConfig, rawReq)
	}
}

func (dl *DNSSimulateGWFListener) writeResponse(addr net.Addr, config *netem.DNSConfig, rawReq []byte) {
	// perform the round trip
	rawResp, err := netem.DNSServerRoundTrip(config, rawReq)

	// on error, just ignore the message
	if err != nil {
		return
	}

	// emit the message and ignore any error; we'll notice ErrClosed
	// in the next ReadFrom call and stop the loop
	_, _ = dl.pconn.WriteTo(rawResp, addr)
}
