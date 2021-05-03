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
	"errors"
	"math/rand"
	"net"
	"strings"
	"sync"
	"time"

	pt "git.torproject.org/pluggable-transports/goptlib.git"
	sf "git.torproject.org/pluggable-transports/snowflake.git/client/lib"
	"git.torproject.org/pluggable-transports/snowflake.git/common/nat"
	"github.com/pion/webrtc/v3"
)

// TODO(bassosimone):
//
// 1. make sure there's log scrubbing
//
// 2. adapt logging to the way in which we log
//
// 3. do not use the pt library?

// snowflakeCtrl contains the control variables that allow us
// to correctly shutdown a running snowflake.
type snowflakeCtrl struct {
	// listeners contains the listeners.
	listener net.Listener

	// shutdown is the channel used to signal shutdown.
	shutdown chan struct{}

	// wg is the sync.WaitGroup used to track the running dispatch loops.
	wg *sync.WaitGroup
}

// Snowflake is the snowflake pluggable transport. You SHOULD NOT modify
// any public field once you have called Start, because doing that will most
// likely lead to data races in your code.
type Snowflake struct {
	// BrokerURL is the optional broker URL.
	BrokerURL string

	// Capacity indicates the number of multiplexed WebRTC peers to use.
	Capacity int

	// FrontDomain is the optional front domain.
	FrontDomain string

	// ICEServersCommas contains an optional command-separated list of ICE servers.
	ICEServersCommas string

	// KeepLocalAddresses indicates whether to keep local LAN
	// addresses ICE candidates.
	KeepLocalAddresses bool

	// Logger contains the optional logger.
	Logger Logger

	// UnsafeLogging indicates whether to NOT scrub logs.
	UnsafeLogging bool

	// mu provides mutual exclusion for accessing ctrl.
	mu sync.Mutex

	// ctrl contains the control variables allowing us to
	// shutdown a running snowflake instance.
	ctrl *snowflakeCtrl
}

// capacity returns the value of Capacity is set, or the default
// value for capacity if Capacity is not set.
func (sfk *Snowflake) capacity() int {
	const defaultSnowflakeCapacity = 1
	if sfk.Capacity > 0 {
		return sfk.Capacity
	}
	return defaultSnowflakeCapacity
}

// logger returns the Logger, if set, or the defaultLogger.
func (sfk *Snowflake) logger() Logger {
	if sfk.Logger != nil {
		return sfk.Logger
	}
	return defaultLogger
}

// socksAcceptLoop accepts local SOCKS connections and passes them to the handler.
func (sfk *Snowflake) socksAcceptLoop(
	ln *pt.SocksListener, tongue sf.Tongue, shutdown chan struct{}, wg *sync.WaitGroup) {
	defer ln.Close()
	for {
		conn, err := ln.AcceptSocks()
		if err != nil {
			if err, ok := err.(net.Error); ok && err.Temporary() {
				continue
			}
			sfk.logger().Warnf("snowflake: SOCKS accept error: %s", err)
			break
		}
		sfk.logger().Debugf("snowflake: SOCKS accepted: %v", conn.Req)
		wg.Add(1) // one more conn running
		go sfk.dispatchSocksConn(conn, tongue, shutdown, wg)
	}
}

// dispatchSocksConn reads from the socks conn and writes to the snowflake
// transport, reads from the snowflake transport and writes to the conn.
func (sfk *Snowflake) dispatchSocksConn(
	conn *pt.SocksConn, tongue sf.Tongue, shutdown chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()
	defer conn.Close()
	err := conn.Grant(&net.TCPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		sfk.logger().Warnf("snowflake: conn.Grant error: %s", err)
		return
	}
	handler := make(chan struct{})
	go func() {
		defer close(handler)
		if err = sf.Handler(conn, tongue); err != nil {
			sfk.logger().Warnf("snowflake: handler error: %s", err)
		}
	}()
	select {
	case <-shutdown:
		sfk.logger().Debugf("Received shutdown signal")
	case <-handler:
		sfk.logger().Debugf("Handler ended")
	}
}

// s is a comma-separated list of ICE server URLs.
func (sfk *Snowflake) parseIceServers(s string) []webrtc.ICEServer {
	var servers []webrtc.ICEServer
	s = strings.TrimSpace(s)
	if len(s) == 0 {
		return nil
	}
	urls := strings.Split(s, ",")
	for _, url := range urls {
		url = strings.TrimSpace(url)
		servers = append(servers, webrtc.ICEServer{
			URLs: []string{url},
		})
	}
	return servers
}

// alreadyRunning returns true if we're already running.
func (sfk *Snowflake) alreadyRunning() bool {
	defer sfk.mu.Unlock()
	sfk.mu.Lock()
	return sfk.ctrl != nil
}

// Start starts the snowflake pluggable transport.
func (sfk *Snowflake) Start() error {
	if sfk.alreadyRunning() {
		return errors.New("snowflake: already running")
	}
	sfk.logger().Infof("snowflake: starting client")

	// TODO(bassosimone): what happens if the user does not pass any ICE server?
	iceServers := sfk.parseIceServers(sfk.ICEServersCommas)

	// chooses a random subset of servers from inputs
	// TODO(bassosimone): maybe we should not use the default randomness source here
	rand.Seed(time.Now().UnixNano())
	rand.Shuffle(len(iceServers), func(i, j int) {
		iceServers[i], iceServers[j] = iceServers[j], iceServers[i]
	})
	if len(iceServers) > 2 {
		iceServers = iceServers[:(len(iceServers)+1)/2]
	}
	sfk.logger().Debugf("snowflake: using ICE servers:")
	for _, server := range iceServers {
		sfk.logger().Debugf("snowflake: url: %v", strings.Join(server.URLs, " "))
	}

	// Use potentially domain-fronting broker to rendezvous.
	broker, err := sf.NewBrokerChannel(
		sfk.BrokerURL, sfk.FrontDomain, sf.CreateBrokerTransport(),
		sfk.KeepLocalAddresses)
	if err != nil {
		return err
	}
	go sfk.updateNATType(iceServers, broker)

	// Create a new WebRTCDialer to use as the |Tongue| to catch snowflakes
	dialer := sf.NewWebRTCDialer(broker, iceServers, sfk.capacity())

	shutdown := make(chan struct{})
	var wg sync.WaitGroup

	// TODO: Be able to recover when SOCKS dies.
	ln, err := pt.ListenSocks("tcp", "127.0.0.1:0")
	if err != nil {
		//pt.CmethodError(methodName, err.Error())
		return err
	}
	sfk.logger().Infof("snowflake: started SOCKS listener at %v.", ln.Addr())
	go sfk.socksAcceptLoop(ln, dialer, shutdown, &wg)

	defer sfk.mu.Unlock()
	sfk.mu.Lock()
	sfk.ctrl = &snowflakeCtrl{
		listener: ln,
		shutdown: shutdown,
		wg:       &wg,
	}
	return nil
}

// TODO(bassosimone): we need to implement .Info, .Warn, .Debug

// Stop stops a running snowflake instance
func (sfk *Snowflake) Stop() {
	sfk.logger().Infof("snowflake: stopping")
	defer sfk.mu.Lock()
	ctrl := sfk.ctrl
	sfk.ctrl = nil
	sfk.mu.Unlock()
	ctrl.listener.Close()
	close(ctrl.shutdown)
	ctrl.wg.Wait()
	sfk.logger().Infof("snowflake: done.")
}

// updateNATType loops through all provided STUN servers until we exhaust the list or find
// one that is compatable with RFC 5780.
func (sfk *Snowflake) updateNATType(servers []webrtc.ICEServer, broker *sf.BrokerChannel) {
	var (
		restrictedNAT bool
		err           error
	)
	for _, server := range servers {
		addr := strings.TrimPrefix(server.URLs[0], "stun:")
		restrictedNAT, err = nat.CheckIfRestrictedNAT(addr)
		if err == nil {
			if restrictedNAT {
				broker.SetNATType(nat.NATRestricted)
			} else {
				broker.SetNATType(nat.NATUnrestricted)
			}
			break
		}
	}
	if err != nil {
		broker.SetNATType(nat.NATUnknown)
	}
}
