package tlsmiddlebox

//
// Iterative network tracing
//

import (
	"context"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"sort"
	"sync"
	"syscall"
	"time"

	"github.com/ooni/probe-cli/v3/internal/logx"
	"github.com/ooni/probe-cli/v3/internal/measurexlite"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	utls "gitlab.com/yawning/utls.git"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
)

// ClientIDs to map configurable inputs to uTLS fingerprints
// We use a non-zero index to map to each ClientID
var ClientIDs = map[int]*utls.ClientHelloID{
	1: &utls.HelloGolang,
	2: &utls.HelloChrome_Auto,
	3: &utls.HelloFirefox_Auto,
	4: &utls.HelloIOS_Auto,
}

// ICMP listener for TTL-exceeded messages
func (m *Measurer) ListenTTLExceeded(ctx context.Context, logger model.Logger, target string) error {
	// The target string contains both the IP and port
	host, _, err := net.SplitHostPort(target)
	if err != nil {
		return err
	}

	targetIP := net.ParseIP(host)
	if targetIP == nil {
		return fmt.Errorf("invalid target IP: %q", target)
	}

	fmt.Println("Target IP:", host)

	conn, err := icmp.ListenPacket("ip4:1", "0.0.0.0")
	if err != nil {
		return err
	}

	defer conn.Close()

	go func() {
		<-ctx.Done()
		conn.Close()
	}()

	buf := make([]byte, 1500)

	for {
		n, peer, err := conn.ReadFrom(buf)
		if err != nil {
			return err
		}

		msg, err := icmp.ParseMessage(1, buf[:n])
		if err != nil {
			continue
		}

		switch msg.Type {
		case ipv4.ICMPTypeTimeExceeded:
			te, ok := msg.Body.(*icmp.TimeExceeded)
			if !ok {
				continue
			}

			data := te.Data

			innerIP, err := ipv4.ParseHeader(data)
			if err != nil {
				continue
			}

			if !innerIP.Dst.Equal(targetIP) {
				continue
			}

			ipHeaderLen := innerIP.Len
			tcpData := te.Data[ipHeaderLen:]

			srcPort := binary.BigEndian.Uint16(tcpData[0:2])

			ttl := -1

			//randomize the base values here
			if srcPort > 4000 && srcPort < 5000 {
				ttl = int(srcPort) - 4000
			} else if srcPort > 5000 {
				ttl = int(srcPort) - 5000
			}

			// dstPort := binary.BigEndian.Uint16(tcpData[2:4])

			// fmt.Printf(
			// "TTL expired for probe -> src=%s dst=%s ttl=%d router=%s src_port=%d dst_port=%d\n",
			// innerIP.Src,
			// innerIP.Dst,
			// innerIP.TTL,
			// peer,
			// srcPort,
			// dstPort,
			// )

			logger.Infof("TTL expired probe --> Source IP=%s Initial TTL:%d", peer, ttl)

			// logger.Infof("ICMP TTL exceeded from %v", peer)
			// logger.Infof("ICMP message %v", te)
		}
	}
}

// TLSTrace performs tracing using control and target SNI
func (m *Measurer) TLSTrace(ctx context.Context, index int64, zeroTime time.Time, logger model.Logger,
	address string, targetSNI string, trace *CompleteTrace) {

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go m.ListenTTLExceeded(ctx, logger, address)

	// perform an iterative trace with the control SNI
	trace.ControlTrace = m.startIterativeTrace(ctx, index, zeroTime, logger, address, m.config.snicontrol(), "control")
	// perform an iterative trace with the target SNI
	trace.TargetTrace = m.startIterativeTrace(ctx, index, zeroTime, logger, address, targetSNI, "censored")
}

// startIterativeTrace creates a Trace and calls iterativeTrace
func (m *Measurer) startIterativeTrace(ctx context.Context, index int64, zeroTime time.Time, logger model.Logger,
	address string, sni string, controlOrCensored string) (tr *IterativeTrace) {
	tr = &IterativeTrace{
		SNI:        sni,
		Iterations: []*Iteration{},
	}
	maxTTL := m.config.maxttl()
	if controlOrCensored == "control" {
		m.traceWithIncreasingTTLs(ctx, index, zeroTime, logger, address, sni, maxTTL, tr, 4000)
	} else {
		m.traceWithIncreasingTTLs(ctx, index, zeroTime, logger, address, sni, maxTTL, tr, 5000)
	}
	tr.Iterations = alignIterations(tr.Iterations)
	return
}

// traceWithIncreasingTTLs performs iterative tracing with increasing TTL values
func (m *Measurer) traceWithIncreasingTTLs(ctx context.Context, index int64, zeroTime time.Time, logger model.Logger,
	address string, sni string, maxTTL int64, trace *IterativeTrace, basePort int) {
	ticker := time.NewTicker(m.config.delay())
	wg := new(sync.WaitGroup)
	for i := int64(1); i <= maxTTL; i++ {
		wg.Add(1)
		go m.handshakeWithTTL(ctx, index, zeroTime, logger, address, sni, int(i), trace, wg, basePort)
		<-ticker.C
	}
	wg.Wait()
}

// handshakeWithTTL performs the TLS Handshake using the passed ttl value
func (m *Measurer) handshakeWithTTL(ctx context.Context, index int64, zeroTime time.Time, logger model.Logger,
	address string, sni string, ttl int, tr *IterativeTrace, wg *sync.WaitGroup, basePort int) {
	defer wg.Done()
	trace := measurexlite.NewTrace(index, zeroTime)
	// 1. Connect to the target IP
	// TODO(DecFox, bassosimone): Do we need a trace for this TCP connect?

	localPort := basePort + int(index*1000) + ttl
	d := NewDialerTTLWrapper(localPort)

	ol := logx.NewOperationLogger(logger, "Handshake Trace #%d TTL %d %s %s", index, ttl, address, sni)
	conn, err := d.DialContext(ctx, "tcp", address)
	if err != nil {
		iteration := newIterationFromHandshake(ttl, err, nil, nil)
		tr.addIterations(iteration)
		ol.Stop(err)
		return
	}
	defer conn.Close()
	// 2. Set the TTL to the passed value
	err = setConnTTL(conn, ttl)
	if err != nil {
		iteration := newIterationFromHandshake(ttl, err, nil, nil)
		tr.addIterations(iteration)
		ol.Stop(err)
		return
	}
	// 3. Perform the handshake and extract the SO_ERROR value (if any)
	// Note: we switch to a uTLS Handshaker if the configured ClientID is non-zero
	thx := trace.NewTLSHandshakerStdlib(logger)
	clientId := m.config.clientid()
	if clientId > 0 {
		thx = trace.NewTLSHandshakerUTLS(logger, ClientIDs[clientId])
	}
	_, err = thx.Handshake(ctx, conn, genTLSConfig(sni))
	ol.Stop(err)
	soErr := extractSoError(conn)
	// 4. reset the TTL value to ensure that conn closes successfully
	// Note: Do not check for errors here
	_ = setConnTTL(conn, 64)
	iteration := newIterationFromHandshake(ttl, nil, soErr, trace.FirstTLSHandshakeOrNil())
	tr.addIterations(iteration)
}

// extractSoError fetches the SO_ERROR value and returns a non-nil error if
// it qualifies as a valid ICMP soft error
// Note: The passed conn must be of type dialerTTLWrapperConn
func extractSoError(conn net.Conn) error {
	soErrno, err := getSoErr(conn)
	if err != nil || errors.Is(soErrno, syscall.Errno(0)) {
		return nil
	}
	soErr := netxlite.MaybeNewErrWrapper(netxlite.ClassifyGenericError, netxlite.TLSHandshakeOperation, soErrno)
	return soErr
}

// genTLSConfig generates tls.Config from a given SNI
func genTLSConfig(sni string) *tls.Config {
	// See https://github.com/ooni/probe/issues/2413 to understand
	// why we're using nil to force netxlite to use the cached
	// default Mozilla cert pool.
	return &tls.Config{ // #nosec G402 - we need to use a large TLS versions range for measuring
		RootCAs:            nil,
		ServerName:         sni,
		NextProtos:         []string{"h2", "http/1.1"},
		InsecureSkipVerify: true, // #nosec G402 - it's fine to skip verify in a nettest
	}
}

// alignIterEvents sorts the iterEvents according to increasing TTL
// and stops when we receive a nil or connection_reset
func alignIterations(in []*Iteration) (out []*Iteration) {
	out = []*Iteration{}
	sort.Slice(in, func(i int, j int) bool {
		return in[i].TTL < in[j].TTL
	})
	for _, iter := range in {
		out = append(out, iter)
		if iter.Handshake.Failure == nil || *iter.Handshake.Failure == netxlite.FailureConnectionReset {
			break
		}
	}
	return out
}
