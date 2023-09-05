package testingx

import (
	"context"
	"errors"
	"net"
	"sync"

	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// DNSOverUDPUnderlyingListener is the underlying listener used by [DNSOverUDPListener].
type DNSOverUDPUnderlyingListener interface {
	ListenUDP(network string, addr *net.UDPAddr) (net.PacketConn, error)
}

// DNSOverUDPStdlibListener implements [DNSOverUDPUnderlyingListener] using the standard library.
type DNSOverUDPStdlibListener struct{}

var _ DNSOverUDPUnderlyingListener = &DNSOverUDPStdlibListener{}

// ListenUDP implements DNSOverUDPUnderlyingListener.
func (*DNSOverUDPStdlibListener) ListenUDP(network string, addr *net.UDPAddr) (net.PacketConn, error) {
	return net.ListenUDP(network, addr)
}

// DNSOverUDPListener is a DNS-over-UDP listener. The zero value of this
// struct is invalid, please use [NewDNSOverUDPListener].
type DNSOverUDPListener struct {
	cancel    context.CancelFunc
	closeOnce sync.Once
	pconn     net.PacketConn
	rtx       DNSRoundTripper
	wg        sync.WaitGroup
}

// MustNewDNSOverUDPListener creates a new [DNSOverUDPListener] using the given
// [DNSOverUDPUnderlyingListener], [DNSRoundTripper], and [*net.UDPAddr].
func MustNewDNSOverUDPListener(addr *net.UDPAddr, dul DNSOverUDPUnderlyingListener, rtx DNSRoundTripper) *DNSOverUDPListener {
	pconn := runtimex.Try1(dul.ListenUDP("udp", addr))
	ctx, cancel := context.WithCancel(context.Background())
	dl := &DNSOverUDPListener{
		cancel:    cancel,
		closeOnce: sync.Once{},
		pconn:     pconn,
		rtx:       rtx,
		wg:        sync.WaitGroup{},
	}
	dl.wg.Add(1)
	go dl.mainloop(ctx)
	return dl
}

// LocalAddr returns the connection address. The return value is nil after you called Close.
func (dl *DNSOverUDPListener) LocalAddr() net.Addr {
	return dl.pconn.LocalAddr()
}

// Close implements io.Closer.
func (dl *DNSOverUDPListener) Close() (err error) {
	dl.closeOnce.Do(func() {
		// close the connection to interrupt ReadFrom or WriteTo
		err = dl.pconn.Close()

		// cancel the context to interrupt the round tripper
		dl.cancel()

		// wait for the background goroutine to join
		dl.wg.Wait()
	})
	return err
}

func (dl *DNSOverUDPListener) mainloop(ctx context.Context) {
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

		// perform the round trip
		rawResp, err := dl.rtx.RoundTrip(ctx, rawReq)

		// on error, just ignore the message
		if err != nil {
			continue
		}

		// emit the message and ignore any error; we'll notice ErrClosed
		// in the next ReadFrom call and stop the loop
		_, _ = dl.pconn.WriteTo(rawResp, addr)
	}
}
