package tunnel

/*
          This file is adapted from client/snowflake.go
    in git.torproject.org/pluggable-transports/snowflake.git
                whose license is the following

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
	"errors"
	"io"
	"net"
	"path/filepath"
	"sync"

	goptlib "git.torproject.org/pluggable-transports/goptlib.git"
	"gitlab.com/yawning/obfs4.git/transports/base"
	"gitlab.com/yawning/obfs4.git/transports/obfs4"
)

// obfs4Ctrl contains the control variables that allow us
// to correctly shutdown a running obfs4.
type obfs4Ctrl struct {
	// cancel allows to stop the obfs4 listener.
	cancel context.CancelFunc

	// factory is the factory to create obfs4 conns.
	factory base.ClientFactory

	// parsedargs contains the parsed arguments.
	parsedargs interface{}
}

// OBFS4 controls an obfs4 pluggable transport.
type OBFS4 struct {
	// Address is the mandatory destination address.
	Address string

	// Cert contains the mandatory certificate parameter.
	Cert string

	// DataDir is the mandatory directory where to store data.
	DataDir string

	// IATMode contains the mandatory iat-mode parameter.
	IATMode string

	// Logger contains the optional logger.
	Logger Logger

	// mu provides mutual exclusion for accessing ctrl.
	mu sync.Mutex

	// ctrl contains the control variables allowing us to
	// shutdown a running OBFS4 instance.
	ctrl *obfs4Ctrl
}

// logger returns the Logger, if set, or the defaultLogger.
func (pt *OBFS4) logger() Logger {
	if pt.Logger != nil {
		return pt.Logger
	}
	return defaultLogger
}

// forward forwards the traffic from left to right and from right to left
// and closes the done channel when it is done. This function DOES NOT
// take ownership of the left, right net.Conn arguments.
func (pt *OBFS4) forward(left, right net.Conn, done chan struct{}) {
	defer close(done)
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
// function takes ownership of the two connections and ensures that
// they are closed when we are done.
func (pt *OBFS4) forwardWithContext(ctx context.Context, left, right net.Conn) {
	defer left.Close()
	defer right.Close()
	done := make(chan struct{})
	go pt.forward(left, right, done)
	select {
	case <-ctx.Done():
	case <-done:
	}
}

// handleSocksConn handles a new SocksConn connection by establishing
// the corresponding OBFS4 connection and forwarding traffic. This
// function takes ownership of the socksConn argument.
func (pt *OBFS4) handleSocksConn(ctx context.Context, socksConn *goptlib.SocksConn) {
	err := socksConn.Grant(&net.TCPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		pt.logger().Warnf("obfs4: socksConn.Grant error: %s", err)
		return
	}
	o4conn, err := pt.ctrl.factory.Dial(
		"tcp", pt.Address, net.Dial, pt.ctrl.parsedargs)
	if err != nil {
		socksConn.Close()
		pt.logger().Warnf("obfs4: factory.Dial error: %s", err)
		return
	}
	pt.forwardWithContext(ctx, socksConn, o4conn)
}

// acceptLoop accepts and handles local socks connection. This function
// takes ownership of the socks listener.
func (pt *OBFS4) acceptLoop(ctx context.Context, ln *goptlib.SocksListener) {
	defer ln.Close()
	for {
		conn, err := ln.AcceptSocks()
		if err != nil {
			if err, ok := err.(net.Error); ok && err.Temporary() {
				continue
			}
			pt.logger().Warnf("obfs4: socks accept error: %s", err)
			return
		}
		pt.logger().Debugf("obfs4: SOCKS accepted: %v", conn.Req)
		go pt.handleSocksConn(ctx, conn)
	}
}

// Start starts the pluggable transport.
func (pt *OBFS4) Start(ctx context.Context) error {
	defer pt.mu.Unlock()
	pt.mu.Lock()
	if pt.ctrl != nil {
		return errors.New("obfs4: already started")
	}
	// TODO(bassosimone): ensure the mandatory arguments are set.
	ctx, cancel := context.WithCancel(ctx)
	txp := &obfs4.Transport{}
	factory, err := txp.ClientFactory(filepath.Join(pt.DataDir, "obfs4"))
	if err != nil {
		cancel()
		return err
	}
	args := &goptlib.Args{
		"cert":     []string{pt.Cert},
		"iat-mode": []string{pt.IATMode},
	}
	parsedargs, err := factory.ParseArgs(args)
	if err != nil {
		cancel()
		return nil
	}
	// TODO(bassosimone): be able to recover when SOCKS dies?
	ln, err := goptlib.ListenSocks("tcp", "127.0.0.1:0")
	if err != nil {
		cancel()
		return err
	}
	pt.logger().Infof("obfs4: started SOCKS listener at %v.", ln.Addr())
	go pt.acceptLoop(ctx, ln)
	pt.ctrl = &obfs4Ctrl{
		cancel:     cancel,
		factory:    factory,
		parsedargs: parsedargs,
	}
	return nil
}

// Stop stops the pluggable transport.
func (pt *OBFS4) Stop() {
	defer pt.mu.Unlock()
	pt.mu.Lock()
	if pt.ctrl != nil {
		pt.ctrl.cancel()
	}
}
