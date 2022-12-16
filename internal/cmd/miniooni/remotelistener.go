package main

//
// Common code for listening for TCP conns
//

import (
	"errors"
	"net"
	"sync"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// remoteConnWrapper wraps a net.Conn to implement the specific
// protocol used by this remote transport.
type remoteConnWrapper func(conn net.Conn) (remoteConn, error)

// remoteListenerFactory implements remoteServerListenerFactory.
type remoteListenerFactory struct {
	iface    string
	port     string
	wrapconn remoteConnWrapper
}

var _ remoteServerListenerFactory = &remoteListenerFactory{}

// Listen implements remoteServerListenerFactory.
func (slf *remoteListenerFactory) Listen() (remoteServerListener, error) {
	dev, err := net.InterfaceByName(slf.iface)
	if err != nil {
		return nil, err
	}
	cidrs, err := dev.Addrs()
	if err != nil {
		return nil, err
	}
	lst := []net.Listener{}
	for _, cidr := range cidrs {
		addr, _, err := net.ParseCIDR(cidr.String())
		if err != nil {
			return nil, err
		}
		if netxlite.IsBogon(addr.String()) {
			// We don't care about listening on link local IPv6 addresses
			// and listening will fail anyway, so...
			continue
		}
		endpoint := net.JoinHostPort(addr.String(), slf.port)
		listener, err := net.Listen("tcp", endpoint)
		if err != nil {
			return nil, err
		}
		log.Infof("remotelistener: listening at %s", listener.Addr().String())
		lst = append(lst, listener)
	}
	wl := &remoteListener{
		closeOnce: &sync.Once{},
		wrapconn:  slf.wrapconn,
		isclosed:  make(chan any),
		listeners: lst,
		newconnch: make(chan remoteConn),
		startOnce: &sync.Once{},
	}
	return wl, nil
}

// remoteListener implements remoteServerListener.
type remoteListener struct {
	closeOnce *sync.Once
	isclosed  chan any
	listeners []net.Listener
	newconnch chan remoteConn
	startOnce *sync.Once
	wrapconn  remoteConnWrapper
}

var _ remoteServerListener = &remoteListener{}

// Accept implements remoteServerListener.
func (rsl *remoteListener) Accept() (remoteConn, error) {
	rsl.startOnce.Do(rsl.startAccepting)
	select {
	case conn := <-rsl.newconnch:
		return conn, nil
	case <-rsl.isclosed:
		return nil, net.ErrClosed
	}
}

// startAccepting starts accepting incoming connections.
func (rsl *remoteListener) startAccepting() {
	for _, lst := range rsl.listeners {
		go rsl.acceptloop(lst)
	}
}

// acceptloop is the accept loop.
func (rsl *remoteListener) acceptloop(listener net.Listener) {
	for {
		conn, err := listener.Accept()
		if err != nil && errors.Is(err, net.ErrClosed) {
			return
		}
		if err != nil {
			log.Warnf("remotelistener: listener.Accept failed: %s", err.Error())
			continue
		}
		go rsl.wrapAndDispatchConn(conn)
	}
}

// wrapAndDispatchConn wraps the connection and then dispatches it to
// the code that will route incoming and outgoing packets.
func (rsl *remoteListener) wrapAndDispatchConn(conn net.Conn) {
	wrapped, err := rsl.wrapconn(conn)
	if err != nil {
		log.Warnf("remotelistener: rsl.wrap failed: %s", err.Error())
		conn.Close()
		return
	}
	select {
	case rsl.newconnch <- wrapped:
	case <-rsl.isclosed:
		conn.Close()
		return
	}
}

// Close implements remoteServerListener.
func (rsl *remoteListener) Close() error {
	var err error
	rsl.closeOnce.Do(func() {
		for _, lst := range rsl.listeners {
			if e := lst.Close(); e != nil && err == nil {
				err = e
			}
		}
		close(rsl.isclosed)
	})
	return err
}
