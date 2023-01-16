package bytecounter

//
// Implementation of Counter
//

import "sync/atomic"

// Counter counts bytes sent and received.
type Counter struct {
	// Received contains the bytes received. You MUST initialize
	// this field, or you can just use the New factory.
	Received *atomic.Int64

	// Sent contains the bytes sent. You MUST initialize
	// this field, or you can just use the New factory.
	Sent *atomic.Int64
}

// New creates a new Counter.
func New() *Counter {
	return &Counter{Received: &atomic.Int64{}, Sent: &atomic.Int64{}}
}

// CountBytesSent adds count to the bytes sent counter.
func (c *Counter) CountBytesSent(count int) {
	c.Sent.Add(int64(count))
}

// CountKibiBytesSent adds 1024*count to the bytes sent counter.
func (c *Counter) CountKibiBytesSent(count float64) {
	c.Sent.Add(int64(1024 * count))
}

// BytesSent returns the bytes sent so far.
func (c *Counter) BytesSent() int64 {
	return c.Sent.Load()
}

// KibiBytesSent returns the KiB sent so far.
func (c *Counter) KibiBytesSent() float64 {
	return float64(c.BytesSent()) / 1024
}

// CountBytesReceived adds count to the bytes received counter.
func (c *Counter) CountBytesReceived(count int) {
	c.Received.Add(int64(count))
}

// CountKibiBytesReceived adds 1024*count to the bytes received counter.
func (c *Counter) CountKibiBytesReceived(count float64) {
	c.Received.Add(int64(1024 * count))
}

// BytesReceived returns the bytes received so far.
func (c *Counter) BytesReceived() int64 {
	return c.Received.Load()
}

// KibiBytesReceived returns the KiB received so far.
func (c *Counter) KibiBytesReceived() float64 {
	return float64(c.BytesReceived()) / 1024
}
