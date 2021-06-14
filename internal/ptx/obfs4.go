package tunnel

/*
         This file is derived from client/snowflake.go
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
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"net"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	pt "git.torproject.org/pluggable-transports/goptlib.git"
	"gitlab.com/yawning/obfs4.git/transports/base"
	"gitlab.com/yawning/obfs4.git/transports/obfs4"
)

// OBFS4 controls an obfs4 pluggable transport. You should not
// change any public field once you have called Start because
// that would likely lead to data races. Make sure you initialize
// all the mandatory fields before calling Start.
type OBFS4 struct {
	// Address is the mandatory destination address.
	Address string

	// Cert contains the mandatory certificate parameter.
	Cert string

	// Dial is the optional dialer. If set, we will use this dialer
	// instead of net.Dial when establishing an obfs4 conn. Overriding
	// this field is the way to perform measurements.
	Dial func(network string, address string) (net.Conn, error)

	// DataDir is the mandatory directory where to store data.
	DataDir string

	// Fingerprint is the mandatory bridge fingerprint.
	Fingerprint string

	// IATMode contains the mandatory iat-mode parameter.
	IATMode string

	// Logger contains the optional logger.
	Logger Logger

	// mu provides mutual exclusion for accessing internals.
	mu sync.Mutex

	// cancel allows to stop the obfs4 listener.
	cancel context.CancelFunc

	// factory is the factory to create obfs4 conns.
	factory base.ClientFactory

	// laddr is the listen address.
	laddr net.Addr

	// parsedargs contains the parsed obfs4 arguments.
	parsedargs interface{}
}

// logger returns the Logger, if set, or the defaultLogger.
func (txp *OBFS4) logger() Logger {
	if txp.Logger != nil {
		return txp.Logger
	}
	return defaultLogger
}

// forward forwards the traffic from left to right and from right to left
// and closes the done channel when it is done. This function DOES NOT
// take ownership of the left, right net.Conn arguments.
func (txp *OBFS4) forward(left, right net.Conn, done chan struct{}) {
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
func (txp *OBFS4) forwardWithContext(ctx context.Context, left, right net.Conn) {
	defer left.Close()
	defer right.Close()
	done := make(chan struct{})
	go txp.forward(left, right, done)
	select {
	case <-ctx.Done():
	case <-done:
	}
}

// dial calls txp.Dial, if set, or net.Dial, otherwise.
func (txp *OBFS4) dial(network, address string) (net.Conn, error) {
	if txp.Dial != nil {
		return txp.Dial(network, address)
	}
	return net.Dial(network, address)
}

// dialWithContext performs the obfs4 dial honouring the context. The
// signature of this method is the same of obfs4 factory's Dial.
func (txp *OBFS4) dialWithContext(
	ctx context.Context, network, address string,
	dial func(network, address string) (net.Conn, error),
	parsedargs interface{},
) (net.Conn, error) {
	connch, errorch := make(chan net.Conn), make(chan error, 1)
	go func() {
		conn, err := txp.factory.Dial(network, address, dial, parsedargs)
		if err != nil {
			errorch <- err // buffered channel
			return
		}
		select {
		case connch <- conn:
		default:
			conn.Close() // context won the race
		}
	}()
	select {
	case err := <-errorch:
		return nil, err
	case conn := <-connch:
		return conn, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// handleSocksConn handles a new SocksConn connection by establishing
// the corresponding OBFS4 connection and forwarding traffic. This
// function TAKES OWNERSHIP of the socksConn argument.
func (txp *OBFS4) handleSocksConn(ctx context.Context, socksConn *pt.SocksConn) {
	err := socksConn.Grant(&net.TCPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		txp.logger().Warnf("obfs4: socksConn.Grant error: %s", err)
		return
	}
	o4conn, err := txp.dialWithContext(
		ctx, "tcp", txp.Address, txp.dial, txp.parsedargs)
	if err != nil {
		socksConn.Close() // we own it
		txp.logger().Warnf("obfs4: factory.Dial error: %s", err)
		return
	}
	txp.forwardWithContext(ctx, socksConn, o4conn)
}

// acceptLoop accepts and handles local socks connection. This function
// TAKES OWNERSHIP of the socks listener.
func (txp *OBFS4) acceptLoop(ctx context.Context, ln *pt.SocksListener) {
	defer ln.Close()
	for {
		conn, err := ln.AcceptSocks()
		if err != nil {
			if err, ok := err.(net.Error); ok && err.Temporary() {
				continue
			}
			txp.logger().Warnf("obfs4: socks accept error: %s", err)
			return
		}
		txp.logger().Debugf("obfs4: SOCKS accepted: %v", conn.Req)
		go txp.handleSocksConn(ctx, conn)
	}
}

// ErrOBFS4Config is an OBFS4 configuration error.
var ErrOBFS4Config = errors.New("obfs4: config error")

// validateConfig ensures mandatory arguments are correctly set.
func (txp *OBFS4) validateConfig() error {
	if txp.Address == "" {
		return fmt.Errorf("%w: txp.Address is empty", ErrOBFS4Config)
	}
	if txp.Cert == "" {
		return fmt.Errorf("%w: txp.Cert is empty", ErrOBFS4Config)
	}
	if txp.DataDir == "" {
		return fmt.Errorf("%w: txp.DataDir is empty", ErrOBFS4Config)
	}
	if txp.Fingerprint == "" {
		return fmt.Errorf("%w: txp.Fingerprint is empty", ErrOBFS4Config)
	}
	if txp.IATMode == "" {
		return fmt.Errorf("%w: txp.IATMode empty", ErrOBFS4Config)
	}
	return nil
}

// newFactory creates an obfs4 factory instance.
func (txp *OBFS4) newFactory() (base.ClientFactory, error) {
	o4f := &obfs4.Transport{}
	return o4f.ClientFactory(filepath.Join(txp.DataDir, "obfs4"))
}

// parseargs parses the obfs4 arguments.
func (txp *OBFS4) parseargs(factory base.ClientFactory) (interface{}, error) {
	args := &pt.Args{"cert": []string{txp.Cert}, "iat-mode": []string{txp.IATMode}}
	return factory.ParseArgs(args)
}

// edit modifies the content of the pluggable transport. This function
// should be the only function that mutates the data structure. It must
// be called while holding the txp.mu mutex.
func (txp *OBFS4) edit(
	cancel context.CancelFunc, factory base.ClientFactory,
	parsedargs interface{}, laddr net.Addr) {
	txp.cancel = cancel
	txp.factory = factory
	txp.parsedargs = parsedargs
	txp.laddr = laddr
}

// Addr returns the listening address. This function should not
// be called after you have called the Stop method or before the
// Start method has successfully returned. When invoked in such
// conditions, this function will return nil. Otherwise, it will
// return the valid net.Addr where we are listening.
func (txp *OBFS4) Addr() net.Addr {
	return txp.laddr
}

// Start starts the pluggable transport. The pluggable transport will
// run in a background goroutine until txp.Stop is called. Attempting to
// call Start when the pluggable transport is already running is a
// no-op causing no error.
func (txp *OBFS4) Start() error {
	txp.mu.Lock()
	defer txp.mu.Unlock()
	if txp.cancel != nil {
		return nil // already started
	}
	if err := txp.validateConfig(); err != nil {
		return err
	}
	factory, err := txp.newFactory()
	if err != nil {
		return err
	}
	parsedargs, err := txp.parseargs(factory)
	if err != nil {
		return err
	}
	// TODO(bassosimone): be able to recover when SOCKS dies?
	ln, err := pt.ListenSocks("tcp", "127.0.0.1:0")
	if err != nil {
		return err
	}
	txp.logger().Infof("obfs4: started socks listener at %v.", ln.Addr())
	ctx, cancel := context.WithCancel(context.Background())
	txp.edit(cancel, factory, parsedargs, ln.Addr())
	go txp.acceptLoop(ctx, ln)
	return nil
}

// Stop stops the pluggable transport. This method is idempotent
// and asks the background goroutine to stop just once.
func (txp *OBFS4) Stop() {
	defer txp.mu.Unlock()
	txp.mu.Lock()
	if txp.cancel != nil {
		txp.cancel() // cancel is idempotent
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
func (txp *OBFS4) AsClientTransportPluginArgument() string {
	return fmt.Sprintf("obfs4 socks5 %s", txp.laddr.String())
}

// AsBridgeArgument returns the argument to be passed to
// the tor command line to declare this bridge.
func (txp *OBFS4) AsBridgeArgument() string {
	return fmt.Sprintf("obfs4 %s %s cert=%s iat-mode=%s",
		txp.Address, txp.Fingerprint, txp.Cert, txp.IATMode)
}

// ErrWrongBridgeType indicates that the parser we're currently
// using does not recognize the specified bridge type.
var ErrWrongBridgeType = errors.New("tunnel: wrong bridge type")

// ErrParseBridgeLine is an error when parsing the bridge line.
var ErrParseBridgeLine = errors.New("tunnel: cannot parse bridge line")

// OBFS4BridgeLineParser parses a bridge line to an OBFS4 data
// structure, or returns an error. We return the ErrWrongBridgeType
// in case the bridge type does not match the expected bridge type. We
// also return ErrParseBridgeLine on parse error. The expected format
// for an obfs4 bridge line is the one returned by the
// https://bridges.torproject.org website. The following
// is the pattern recognized by this function:
//
//     obfs4 <address>:<port> <fingerprint> cert=<cert> iat-mode=<mode>
//
// Note that the relative order of `cert` and `iat-mode` does
// not matter, but we expect both options to be present.
//
// We also recognize the case where the line starts with the
// string "Bridge", to support the way in which bridges are
// specified in the `tor` configuration file.
type OBFS4BridgeLineParser struct {
	// BridgeLine contains the bridge line to parse.
	BridgeLine string

	// DataDir contains the data directory.
	DataDir string
}

// Parse parses the OBFS4BridgeLine into an OBFS4 structure or an error.
func (p *OBFS4BridgeLineParser) Parse() (*OBFS4, error) {
	vals := strings.Split(p.BridgeLine, " ")
	blp := &obfs4BridgeLineParserCtx{
		bridgeKeyword: make(chan *obfs4BridgeLineParserState),
		bridgeType:    make(chan *obfs4BridgeLineParserState),
		endpoint:      make(chan *obfs4BridgeLineParserState),
		fingerprint:   make(chan *obfs4BridgeLineParserState),
		options:       make(chan *obfs4BridgeLineParserState),
		nextOptions:   make(chan *obfs4BridgeLineParserState),
		err:           make(chan error),
		result:        make(chan *OBFS4),
		wg:            &sync.WaitGroup{},
	}
	launch := func(f func()) {
		blp.wg.Add(1) // count goro as running
		go f()
	}
	launch(blp.parseBridgeKeyword)
	launch(blp.parseBridgeType)
	launch(blp.parseEndpoint)
	launch(blp.parseFingerprint)
	launch(blp.parseOptions)
	launch(blp.parseNextOptions)
	blp.bridgeKeyword <- &obfs4BridgeLineParserState{ // kick off
		vals: vals,
		o4:   &OBFS4{DataDir: p.DataDir},
	}
	var (
		err    error
		result *OBFS4
	)
	select {
	case err = <-blp.err:
	case result = <-blp.result:
		return result, nil
	}
	close(blp.bridgeKeyword)
	close(blp.bridgeType)
	close(blp.endpoint)
	close(blp.fingerprint)
	close(blp.options)
	close(blp.nextOptions)
	blp.wg.Wait() // join the goros
	return result, err
}

// obfs4BridgeLineParserState contains the parser state (i.e., the
// "piece" that is to be worked on by the "stations").
type obfs4BridgeLineParserState struct {
	// vals contains the not-parsed-yet tokens
	vals []string

	// o4 contains the output structure
	o4 *OBFS4
}

// obfs4BridgeLineParserCtx contains the parser context (i.e., the
// context grouping the variables used by the parse "stations").
type obfs4BridgeLineParserCtx struct {
	// bridgeKeyword is the input of the parseBridgeKeyword parser.
	bridgeKeyword chan *obfs4BridgeLineParserState

	// bridgeType is the input of the parseBridgeType parser.
	bridgeType chan *obfs4BridgeLineParserState

	// endpoint is the input of the parseEndpoint parser.
	endpoint chan *obfs4BridgeLineParserState

	// fingerprint is the input for the parseFingerprint state.
	fingerprint chan *obfs4BridgeLineParserState

	// options is the input of the parseOptions parser.
	options chan *obfs4BridgeLineParserState

	// nextOptions is the input of the parseNextOptions parser.
	nextOptions chan *obfs4BridgeLineParserState

	// err is an output indicating that parsing failed.
	err chan error

	// result is an output indicating that parsing succeded.
	result chan *OBFS4

	// wg counts the number of running goroutines
	wg *sync.WaitGroup
}

// parseBridgeKeyword parses the optional "bridge" keyword.
func (p *obfs4BridgeLineParserCtx) parseBridgeKeyword() {
	defer p.wg.Done()
	for s := range p.bridgeKeyword {
		if len(s.vals) >= 1 && strings.ToLower(s.vals[0]) == "bridge" {
			s.vals = s.vals[1:] // just skip the keyword
		}
		p.bridgeType <- s
	}
}

// parseBridgeType parses the mandatory bridge type ("obfs4").
func (p *obfs4BridgeLineParserCtx) parseBridgeType() {
	defer p.wg.Done()
	for s := range p.bridgeType {
		if len(s.vals) < 1 {
			p.err <- fmt.Errorf("%w: missing bridge type", ErrParseBridgeLine)
			continue
		}
		if s.vals[0] != "obfs4" {
			p.err <- fmt.Errorf(
				"%w: expected 'obfs4', found '%s'", ErrWrongBridgeType, s.vals[0])
			continue
		}
		s.vals = s.vals[1:]
		p.endpoint <- s
	}
}

// parseEndpoint parses the mandatory bridge endpoint. We expect the
// endpoint to be like 1.2.3.4:5678 or like [::1:ef:3:4]:5678.
func (p *obfs4BridgeLineParserCtx) parseEndpoint() {
	defer p.wg.Done()
	for s := range p.endpoint {
		if len(s.vals) < 1 {
			p.err <- fmt.Errorf("%w: missing bridge endpoint", ErrParseBridgeLine)
			continue
		}
		if _, _, err := net.SplitHostPort(s.vals[0]); err != nil {
			p.err <- fmt.Errorf("%w: %s", ErrParseBridgeLine, err.Error())
			continue
		}
		s.o4.Address = s.vals[0]
		s.vals = s.vals[1:]
		p.fingerprint <- s
	}
}

// parseFingerprint parses the fingerprint.
func (p *obfs4BridgeLineParserCtx) parseFingerprint() {
	defer p.wg.Done()
	for s := range p.fingerprint {
		if len(s.vals) < 1 {
			p.err <- fmt.Errorf("%w: missing bridge fingerprint", ErrParseBridgeLine)
			continue
		}
		re := regexp.MustCompile("^[A-Fa-f0-9]{40}$")
		if !re.MatchString(s.vals[0]) {
			p.err <- fmt.Errorf("%w: invalid bridge fingerprint", ErrParseBridgeLine)
			continue
		}
		s.o4.Fingerprint = s.vals[0]
		s.vals = s.vals[1:]
		p.options <- s
	}
}

// parseOptions parses the options.
func (p *obfs4BridgeLineParserCtx) parseOptions() {
	defer p.wg.Done()
	for s := range p.options {
		if len(s.vals) < 1 {
			if s.o4.Cert == "" {
				p.err <- fmt.Errorf("%w: missing bridge cert", ErrParseBridgeLine)
				continue
			}
			if s.o4.IATMode == "" {
				p.err <- fmt.Errorf("%w: missing bridge iat-mode", ErrParseBridgeLine)
				continue
			}
			p.result <- s.o4
			continue
		}
		v := s.vals[0]
		s.vals = s.vals[1:]
		if strings.HasPrefix(v, "cert=") {
			v = v[len("cert="):]
			cert := v + "=="
			if _, err := base64.StdEncoding.DecodeString(cert); err != nil {
				p.err <- fmt.Errorf(
					"%w: cannot parse cert: %s", ErrParseBridgeLine, err.Error())
				continue
			}
			s.o4.Cert = v
			p.nextOptions <- s // avoid self deadlock
			continue
		}
		if strings.HasPrefix(v, "iat-mode") {
			v = v[len("iat-mode="):]
			if _, err := strconv.Atoi(v); err != nil {
				p.err <- fmt.Errorf(
					"%w: cannot parse iat-mode: %s", ErrParseBridgeLine, err.Error())
				continue
			}
			s.o4.IATMode = v
			p.nextOptions <- s // avoid self deadlock
			continue
		}
		p.err <- fmt.Errorf("%w: invalid option: %s", ErrParseBridgeLine, v)
	}
}

// parseOptions parses the options.
func (p *obfs4BridgeLineParserCtx) parseNextOptions() {
	defer p.wg.Done()
	for s := range p.nextOptions {
		p.options <- s
	}
}
