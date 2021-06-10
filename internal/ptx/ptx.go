package ptx

/*-
         This file is derived from client/snowflake.go
    in git.torproject.org/pluggable-transports/snowflake.git
                whose license is the following:

================================================================================

Copyright (c) 2016, Serene Han, Arlo Breault
Copyright (c) 2019-2020, The Tor Project, Inc

Redistribution and use in source and binary forms, with or without modification,
are permitted provided that the following conditions are met:

  * Redistributions of source code must retain the above copyright notice, this
list of conditions and the following disclaimer.

  * Redistributions in binary form must reproduce the above copyright notice,
this list of conditions and the following disclaimer in the documentation and/or
other materials provided with the distribution.

  * Neither the names of the copyright owners nor the names of its
contributors may be used to endorse or promote products derived from this
software without specific prior written permission.

THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS" AND
ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT HOLDER OR CONTRIBUTORS BE LIABLE FOR
ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
(INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON
ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
(INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
================================================================================
*/

import (
	"context"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"

	pt "git.torproject.org/pluggable-transports/goptlib.git"
)

// PTDialer is a generic pluggable transports dialer.
type PTDialer interface {
	// DialContext establishes a connection to the pluggable
	// transport backend according to PT-specific configuration
	// and returns you such a connection.
	DialContext(ctx context.Context) (net.Conn, error)

	// AsBridgeArgument returns the argument to be passed to
	// the tor command line to declare this bridge.
	AsBridgeArgument() string

	// Name returns the pluggable transport name.
	Name() string
}

// Listener is a generic pluggable transports listener. Make sure
// you fill the mandatory fields before using. Do not modify public
// fields after you called Start, since this causes data races.
type Listener struct {
	// ContextDialer is the MANDATORY pluggable transports dialer
	// to use. Both SnowflakeDialer and OBFS4Dialer implement this
	// interface and can be thus safely used here.
	ContextDialer PTDialer

	// Logger is the optional logger. When not set, this library
	// will not emit logs. (But the underlying pluggable transport
	// may still emit its own log messages.)
	Logger Logger

	// mu provides mutual exclusion for accessing internals.
	mu sync.Mutex

	// cancel allows to stop the listener.
	cancel context.CancelFunc

	// laddr is the listen address.
	laddr net.Addr
}

// logger returns the Logger, if set, or the defaultLogger.
func (lst *Listener) logger() Logger {
	if lst.Logger != nil {
		return lst.Logger
	}
	return defaultLogger
}

// forward forwards the traffic from left to right and from right to left
// and closes the done channel when it is done. This function DOES NOT
// take ownership of the left, right net.Conn arguments.
func (lst *Listener) forward(left, right net.Conn, done chan struct{}) {
	defer close(done) // signal termination
	wg := new(sync.WaitGroup)
	wg.Add(2)
	go func() {
		defer wg.Done()
		io.Copy(left, right)
	}()
	go func() {
		defer wg.Done()
		io.Copy(right, left)
	}()
	wg.Wait()
}

// forwardWithContext forwards the traffic from left to right and
// form right to left, interrupting when the context is done. This
// function TAKES OWNERSHIP of the two connections and ensures
// that they are closed when we are done.
func (lst *Listener) forwardWithContext(ctx context.Context, left, right net.Conn) {
	defer left.Close()
	defer right.Close()
	done := make(chan struct{})
	go lst.forward(left, right, done)
	select {
	case <-ctx.Done():
	case <-done:
	}
}

// handleSocksConn handles a new SocksConn connection by establishing
// the corresponding PT connection and forwarding traffic. This
// function TAKES OWNERSHIP of the socksConn argument.
func (lst *Listener) handleSocksConn(ctx context.Context, socksConn *pt.SocksConn) {
	err := socksConn.Grant(&net.TCPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		lst.logger().Warnf("ptx: socksConn.Grant error: %s", err)
		return
	}
	ptConn, err := lst.ContextDialer.DialContext(ctx)
	if err != nil {
		socksConn.Close() // we own it
		lst.logger().Warnf("ptx: ContextDialer.DialContext error: %s", err)
		return
	}
	lst.forwardWithContext(ctx, socksConn, ptConn)
}

// acceptLoop accepts and handles local socks connection. This function
// TAKES OWNERSHIP of the socks listener.
func (lst *Listener) acceptLoop(ctx context.Context, ln *pt.SocksListener) {
	defer ln.Close()
	for {
		conn, err := ln.AcceptSocks()
		if err != nil {
			if err, ok := err.(net.Error); ok && err.Temporary() {
				continue
			}
			lst.logger().Warnf("ptx: socks accept error: %s", err)
			return
		}
		lst.logger().Infof("ptx: SOCKS accepted: %v", conn.Req)
		go lst.handleSocksConn(ctx, conn)
	}
}

// Addr returns the listening address. This function should not
// be called after you have called the Stop method or before the
// Start method has successfully returned. When invoked in such
// conditions, this function will return nil. Otherwise, it will
// return the valid net.Addr where we are listening.
func (lst *Listener) Addr() net.Addr {
	return lst.laddr
}

// Start starts the pluggable transport Listener. The pluggable transport will
// run in a background goroutine until txp.Stop is called. Attempting to
// call Start when the pluggable transport is already running is a
// no-op causing no error and no data races.
func (lst *Listener) Start() error {
	lst.mu.Lock()
	defer lst.mu.Unlock()
	if lst.cancel != nil {
		return nil // already started
	}
	// TODO(bassosimone): be able to recover when SOCKS dies?
	ln, err := pt.ListenSocks("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	lst.laddr = ln.Addr()
	ctx, cancel := context.WithCancel(context.Background())
	lst.cancel = cancel
	go lst.acceptLoop(ctx, ln)
	lst.logger().Infof("ptx: started socks listener at %v", ln.Addr())
	lst.logger().Debugf("ptx: test with `%s`", lst.torCmdLine())
	return nil
}

// torCmdLine prints the command line for testing this listener.
func (lst *Listener) torCmdLine() string {
	return strings.Join([]string{
		"tor",
		"DataDirectory",
		"testdata",
		"UseBridges",
		"1",
		"ClientTransportPlugin",
		"'" + lst.AsClientTransportPluginArgument() + "'",
		"Bridge",
		"'" + lst.ContextDialer.AsBridgeArgument() + "'",
	}, " ")
}

// Stop stops the pluggable transport. This method is idempotent
// and asks the background goroutine to stop just once. Also, this
// method is safe to call from any goroutine.
func (lst *Listener) Stop() {
	defer lst.mu.Unlock()
	lst.mu.Lock()
	if lst.cancel != nil {
		lst.cancel() // cancel is idempotent
	}
}

// AsClientTransportPluginArgument converts the current configuration
// of the pluggable transport to a ClientTransportPlugin argument to be
// passed to the tor daemon command line. This function must be
// called after Start and before Stop so that we have a valid Addr.
//
// Assuming that we are listening at 127.0.0.1:12345, then this
// function will return the following string:
//
//     obfs4 socks5 127.0.0.1:12345
//
// The correct configuration line for the `torrc` would be:
//
//     ClientTransportPlugin obfs4 socks5 127.0.0.1:12345
//
// Since we pass configuration to tor using the command line, it
// is more convenient to us to avoid including ClientTransportPlugin
// in the returned string. In fact, ClientTransportPlugin and its
// arguments need to be two consecutive argv strings.
func (lst *Listener) AsClientTransportPluginArgument() string {
	return fmt.Sprintf("%s socks5 %s", lst.ContextDialer.Name(), lst.laddr.String())
}
