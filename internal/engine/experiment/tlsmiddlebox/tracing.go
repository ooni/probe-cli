package tlsmiddlebox

import (
	"context"
	"crypto/tls"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/experiment/tlsmiddlebox/internal"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// MeasureTLS outputs a TraceEvent with the iterative trace for the passSNI and the targetSNI
func (m *Measurer) MeasureTLS(ctx context.Context, addr string, targetSNI string, tlsEvents chan<- *CompleteTrace) {
	out := &CompleteTrace{}
	passSNI := m.config.snipass()
	passTrace := m.IterativeTrace(ctx, addr, passSNI)
	out.PassTrace = passTrace
	targetTrace := m.IterativeTrace(ctx, addr, targetSNI)
	out.TargetTrace = targetTrace
	select {
	case tlsEvents <- out:
	default:
		return
	}
}

// IterativeTrace calls the iterativeTrace and populates the TraceEvent with iteration results
func (m *Measurer) IterativeTrace(ctx context.Context, addr string, sni string) (trace *TraceEvent) {
	iterations := m.config.iterations()
	out := make(chan *IterEvent, iterations)
	trace = &TraceEvent{
		Address:    addr,
		SNI:        sni,
		Iterations: []*IterEvent{},
	}
	m.iterativeTrace(ctx, addr, sni, iterations, out)
	iterEvents := extractEvents(out) // align the iteration results before modeling them
	trace.AddIterations(iterEvents)
	return trace
}

func (m *Measurer) iterativeTrace(ctx context.Context, addr string, sni string,
	iterations int, out chan<- *IterEvent) {
	ticker := time.NewTicker(m.config.delay())
	wg := new(sync.WaitGroup)
	for i := 1; i <= iterations; i++ {
		wg.Add(1)
		go iterAsync(ctx, addr, sni, i, out, wg)
		<-ticker.C
	}
	wg.Wait()
}

// Single Iteration for network tracing
func iterAsync(ctx context.Context, addr string, sni string,
	ttl int, out chan<- *IterEvent, wg *sync.WaitGroup) {
	defer wg.Done()
	select {
	case out <- HandshakeWithTTL(ctx, addr, sni, ttl):
	default:
		return
	}
}

// This handles the conn and calls the handshake function after setting the TTL value
func HandshakeWithTTL(ctx context.Context, addr string, sni string, ttl int) (out *IterEvent) {
	logger := model.ValidLoggerOrDefault(nil)
	out = &IterEvent{
		TTL:     ttl,
		Failure: nil,
	}
	// we use the net.Dialer instead of netxlite.Dialer
	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		errStr := err.Error()
		out.Failure = &errStr
		return
	}
	defer conn.Close()
	err = internal.SetConnTTL(conn, ttl)
	if err != nil {
		errStr := err.Error()
		out.Failure = &errStr
		return
	}
	performHandshake(ctx, conn, sni, logger, out)
	internal.ResetConnTTL(conn) // reset the TTL to make sure the conn closes successfully
	return
}

// perform a TLS Handshake given a net.Conn and populate the IterEvent
func performHandshake(ctx context.Context, conn net.Conn, sni string,
	logger model.Logger, in *IterEvent) {
	h := netxlite.NewTLSHandshakerStdlib(logger)
	start := time.Now()
	_, _, err := h.Handshake(ctx, conn, genTLSConfig(sni))
	elapsed := time.Since(start)
	in.Duration = elapsed.Milliseconds()
	// using the stdlib to record errors
	if err != nil {
		errStr := err.Error()
		in.Failure = &errStr
	}
}

// generate the tls.Config from a given SNI
func genTLSConfig(SNI string) *tls.Config {
	return &tls.Config{
		RootCAs:            netxlite.NewDefaultCertPool(),
		ServerName:         SNI,
		NextProtos:         []string{"h2", "http/1.1"},
		InsecureSkipVerify: true,
	}
}

// extractEvents takes in a channel and outputs an aligned array
func extractEvents(traceEvents <-chan *IterEvent) (out []*IterEvent) {
	tmpEvents := GetTraceEvents(traceEvents)
	out = alignIterEvents(tmpEvents)
	return
}

// alignIterEvents sorts the iterEvents according to increasing TTL
// and also stops when we receive a success or a connection_reset
func alignIterEvents(in []*IterEvent) (out []*IterEvent) {
	out = []*IterEvent{}
	sort.Slice(in, func(i int, j int) bool {
		return in[i].TTL < in[j].TTL
	})
	for _, ev := range in {
		out = append(out, ev)
		if ev.Failure == nil || *ev.Failure == "connection_reset" {
			break
		}
	}
	return
}

// GetTraceEvents extracts the contents of an IterEvent buffered channel to an array
func GetTraceEvents(traceEvents <-chan *IterEvent) (out []*IterEvent) {
	for {
		select {
		case ev := <-traceEvents:
			out = append(out, ev)
		default:
			return
		}
	}
}

func GetTLSEvents(tcpEvents <-chan *CompleteTrace) (out []*CompleteTrace) {
	for {
		select {
		case ev := <-tcpEvents:
			out = append(out, ev)
		default:
			return
		}
	}
}
