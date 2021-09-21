package netxlite

import (
	"context"
	"encoding/json"
	"net"
	"os"
	"sync"
	"time"

	"github.com/ooni/probe-cli/v3/internal/atomicx"
	"github.com/ooni/probe-cli/v3/internal/netxlite/quicx"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// bandwidthStats contains bandwidth stats.
type bandwidthStats struct {
	// Timestamp is the timestamp when we saved this snapshot.
	Timestamp time.Time

	// Read is the number of bytes read using Read.
	Read int64

	// ReadFrom is the number of bytes read using ReadFrom.
	ReadFrom int64

	// Write is the number of bytes written using Write.
	Write int64

	// WriteTo is the number of bytes written using WriteTo.
	WriteTo int64
}

// bandwidthMonitor monitors the bandwidth usage.
type bandwidthMonitor struct {
	enabled *atomicx.Int64
	stats   bandwidthStats
	mu      sync.Mutex
}

// MonitorBandwidth configures bandwidth monitoring. The filename
// argument is the name of the file where to write snapshots. By
// default bandwidth monitoring is disabled and you only enable it
// by calling this function once in your main function.
func MonitorBandwidth(ctx context.Context, filename string) {
	bwmonitor.enabled.Add(1)
	go bwmonitor.measure(ctx, filename)
}

// measure performs periodic measurements.
func (bwmon *bandwidthMonitor) measure(ctx context.Context, filename string) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case t := <-ticker.C:
			bwmon.saveSnapshot(t, filename)
		case <-ctx.Done():
			return
		}
	}
}

// saveSnapshot appends the snapshot to the snapshots file.
func (bwmon *bandwidthMonitor) saveSnapshot(t time.Time, filename string) {
	bwmon.mu.Lock()
	bwmon.stats.Timestamp = t
	data, err := json.Marshal(bwmon.stats)
	bwmon.stats = bandwidthStats{}
	bwmon.mu.Unlock()
	data = append(data, '\n')
	runtimex.PanicOnError(err, "json.Marshal failed")
	const flags = os.O_APPEND | os.O_CREATE | os.O_WRONLY
	filep, err := os.OpenFile(filename, flags, 0644)
	if err != nil {
		return
	}
	defer filep.Close()
	if _, err := filep.Write(data); err != nil {
		filep.Close()
		return
	}
}

// MaybeWrapConn possibly wraps a net.Conn to add bandwidth monitoring. If there is
// an error this function immediately returns an error. Bandwidth monitoring is
// disabled by default, but can be enabled when required.
func (bwmon *bandwidthMonitor) MaybeWrapConn(conn net.Conn, err error) (net.Conn, error) {
	if bwmon.enabled.Load() == 0 {
		return conn, err
	}
	if err != nil {
		return nil, err
	}
	return &bwmonConn{Conn: conn, bwmon: bwmon}, nil
}

// OnRead measures the results of Conn.Read.
func (bwmon *bandwidthMonitor) OnRead(count int, err error) (int, error) {
	bwmon.mu.Lock()
	bwmon.stats.Read += int64(count)
	bwmon.mu.Unlock()
	return count, err
}

// OnWrite measures the results of Conn.Write.
func (bwmon *bandwidthMonitor) OnWrite(count int, err error) (int, error) {
	bwmon.mu.Lock()
	bwmon.stats.Write += int64(count)
	bwmon.mu.Unlock()
	return count, err
}

// OnWriteTo measures the results of UDPLikeConn.WriteTo.
func (bwmon *bandwidthMonitor) OnWriteTo(count int, err error) (int, error) {
	bwmon.mu.Lock()
	bwmon.stats.WriteTo += int64(count)
	bwmon.mu.Unlock()
	return count, err
}

// OnReadFrom measures the results of UDPLikeConn.ReadFrom.
func (bwmon *bandwidthMonitor) OnReadFrom(
	count int, addr net.Addr, err error) (int, net.Addr, error) {
	bwmon.mu.Lock()
	bwmon.stats.ReadFrom += int64(count)
	bwmon.mu.Unlock()
	return count, addr, err
}

// bwmonConn wraps a net.Conn to add bandwidth monitoring.
type bwmonConn struct {
	net.Conn
	bwmon *bandwidthMonitor
}

// Read implements net.Conn.Read.
func (c *bwmonConn) Read(b []byte) (int, error) {
	return c.bwmon.OnRead(c.Conn.Read(b))
}

// Read implements net.Conn.Read.
func (c *bwmonConn) Write(b []byte) (int, error) {
	return c.bwmon.OnWrite(c.Conn.Write(b))
}

// MaybeWrapUDPLikeConn possibly wraps a quicx.UDPLikeConn to add bandwidth
// monitoring. If there is an error this function immediately returns an
// error. Bandwidth monitoring is disabled by default, but can be
// enabled when required.
func (bwmon *bandwidthMonitor) MaybeWrapUDPLikeConn(
	conn quicx.UDPLikeConn, err error) (quicx.UDPLikeConn, error) {
	if bwmon.enabled.Load() == 0 {
		return conn, err
	}
	if err != nil {
		return nil, err
	}
	return &bwmonUDPLikeConn{UDPLikeConn: conn, bwmon: bwmon}, nil
}

// bwmonUDPLikeConn wraps a quicx.UDPLikeConn to add bandwidth monitoring.
type bwmonUDPLikeConn struct {
	quicx.UDPLikeConn
	bwmon *bandwidthMonitor
}

// WriteTo implements quicx.UDPLikeConn.WriteTo.
func (c *bwmonUDPLikeConn) WriteTo(p []byte, addr net.Addr) (int, error) {
	return c.bwmon.OnWriteTo(c.UDPLikeConn.WriteTo(p, addr))
}

// ReadFrom implements quicx.UDPLikeConn.ReadFrom.
func (c *bwmonUDPLikeConn) ReadFrom(b []byte) (int, net.Addr, error) {
	return c.bwmon.OnReadFrom(c.UDPLikeConn.ReadFrom(b))
}

// bwmonitor is the bandwidth monitor singleton
var bwmonitor = &bandwidthMonitor{
	enabled: &atomicx.Int64{},
}
