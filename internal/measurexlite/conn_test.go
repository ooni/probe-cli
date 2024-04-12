package measurexlite

import (
	"net"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/mocks"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

func TestRemoteAddrProvider(t *testing.T) {
	conn := &mocks.Conn{
		MockRemoteAddr: func() net.Addr {
			return &mocks.Addr{
				MockString: func() string {
					return "1.1.1.1:443"
				},
				MockNetwork: func() string {
					return "tcp"
				},
			}
		},
	}
	if safeRemoteAddrNetwork(conn) != "tcp" {
		t.Fatal("unexpected network")
	}
	if safeRemoteAddrString(conn) != "1.1.1.1:443" {
		t.Fatal("unexpected string")
	}
}

func TestMaybeClose(t *testing.T) {
	t.Run("with nil conn", func(t *testing.T) {
		var conn net.Conn = nil
		MaybeClose(conn)
	})

	t.Run("with nonnil conn", func(t *testing.T) {
		var called bool
		conn := &mocks.Conn{
			MockClose: func() error {
				called = true
				return nil
			},
		}
		if err := MaybeClose(conn); err != nil {
			t.Fatal(err)
		}
		if !called {
			t.Fatal("not called")
		}
	})
}

func TestMaybeCloseUDPLikeConn(t *testing.T) {
	t.Run("with nil conn", func(t *testing.T) {
		var conn model.UDPLikeConn = nil
		MaybeCloseUDPLikeConn(conn)
	})

	t.Run("with nonnil conn", func(t *testing.T) {
		var called bool
		conn := &mocks.UDPLikeConn{
			MockClose: func() error {
				called = true
				return nil
			},
		}
		if err := MaybeCloseUDPLikeConn(conn); err != nil {
			t.Fatal(err)
		}
		if !called {
			t.Fatal("not called")
		}
	})
}

func TestWrapNetConn(t *testing.T) {
	t.Run("WrapNetConn wraps the conn", func(t *testing.T) {
		underlying := &mocks.Conn{}
		zeroTime := time.Now()
		trace := NewTrace(0, zeroTime)
		conn := trace.MaybeWrapNetConn(underlying)
		ct := conn.(*connTrace)
		if ct.Conn != underlying {
			t.Fatal("invalid underlying")
		}
		if ct.tx != trace {
			t.Fatal("invalid trace")
		}
	})

	t.Run("Read saves a trace", func(t *testing.T) {
		underlying := &mocks.Conn{
			MockRead: func(b []byte) (int, error) {
				return len(b), nil
			},
			MockRemoteAddr: func() net.Addr {
				return &mocks.Addr{
					MockNetwork: func() string {
						return "tcp"
					},
					MockString: func() string {
						return "1.1.1.1:443"
					},
				}
			},
		}
		zeroTime := time.Now()
		td := testingx.NewTimeDeterministic(zeroTime)
		trace := NewTrace(0, zeroTime, "antani")
		trace.timeNowFn = td.Now // deterministic time counting
		conn := trace.MaybeWrapNetConn(underlying)
		const bufsiz = 128
		buffer := make([]byte, bufsiz)
		count, err := conn.Read(buffer)
		if count != bufsiz {
			t.Fatal("invalid count")
		}
		if err != nil {
			t.Fatal("invalid err")
		}

		t.Run("we update the trace's byte received map", func(t *testing.T) {
			stats := trace.CloneBytesReceivedMap()
			if len(stats) != 1 {
				t.Fatal("expected to see just one entry")
			}
			if stats["1.1.1.1:443 tcp"] != 128 {
				t.Fatal("expected to know we received 128 bytes")
			}
		})

		events := trace.NetworkEvents()
		if len(events) != 1 {
			t.Fatal("did not save network events")
		}
		expect := &model.ArchivalNetworkEvent{
			Address:   "1.1.1.1:443",
			Failure:   nil,
			NumBytes:  bufsiz,
			Operation: netxlite.ReadOperation,
			Proto:     "tcp",
			T:         1.0,
			Tags:      []string{"antani"},
		}
		got := events[0]
		if diff := cmp.Diff(expect, got); diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("Read discards the event when the buffer is full", func(t *testing.T) {
		underlying := &mocks.Conn{
			MockRead: func(b []byte) (int, error) {
				return len(b), nil
			},
			MockRemoteAddr: func() net.Addr {
				return &mocks.Addr{
					MockNetwork: func() string {
						return "tcp"
					},
					MockString: func() string {
						return "1.1.1.1:443"
					},
				}
			},
		}
		zeroTime := time.Now()
		trace := NewTrace(0, zeroTime)
		trace.networkEvent = make(chan *model.ArchivalNetworkEvent) // no buffer
		conn := trace.MaybeWrapNetConn(underlying)
		const bufsiz = 128
		buffer := make([]byte, bufsiz)
		count, err := conn.Read(buffer)
		if count != bufsiz {
			t.Fatal("invalid count")
		}
		if err != nil {
			t.Fatal("invalid err")
		}
		events := trace.NetworkEvents()
		if len(events) != 0 {
			t.Fatal("expected no network events")
		}
	})

	t.Run("Write saves a trace", func(t *testing.T) {
		underlying := &mocks.Conn{
			MockWrite: func(b []byte) (int, error) {
				return len(b), nil
			},
			MockRemoteAddr: func() net.Addr {
				return &mocks.Addr{
					MockNetwork: func() string {
						return "tcp"
					},
					MockString: func() string {
						return "1.1.1.1:443"
					},
				}
			},
		}
		zeroTime := time.Now()
		td := testingx.NewTimeDeterministic(zeroTime)
		trace := NewTrace(0, zeroTime, "antani")
		trace.timeNowFn = td.Now // deterministic time tracking
		conn := trace.MaybeWrapNetConn(underlying)
		const bufsiz = 128
		buffer := make([]byte, bufsiz)
		count, err := conn.Write(buffer)
		if count != bufsiz {
			t.Fatal("invalid count")
		}
		if err != nil {
			t.Fatal("invalid err")
		}
		events := trace.NetworkEvents()
		if len(events) != 1 {
			t.Fatal("did not save network events")
		}
		expect := &model.ArchivalNetworkEvent{
			Address:   "1.1.1.1:443",
			Failure:   nil,
			NumBytes:  bufsiz,
			Operation: netxlite.WriteOperation,
			Proto:     "tcp",
			T:         1.0,
			Tags:      []string{"antani"},
		}
		got := events[0]
		if diff := cmp.Diff(expect, got); diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("Write discards the event when the buffer is full", func(t *testing.T) {
		underlying := &mocks.Conn{
			MockWrite: func(b []byte) (int, error) {
				return len(b), nil
			},
			MockRemoteAddr: func() net.Addr {
				return &mocks.Addr{
					MockNetwork: func() string {
						return "tcp"
					},
					MockString: func() string {
						return "1.1.1.1:443"
					},
				}
			},
		}
		zeroTime := time.Now()
		trace := NewTrace(0, zeroTime)
		trace.networkEvent = make(chan *model.ArchivalNetworkEvent) // no buffer
		conn := trace.MaybeWrapNetConn(underlying)
		const bufsiz = 128
		buffer := make([]byte, bufsiz)
		count, err := conn.Write(buffer)
		if count != bufsiz {
			t.Fatal("invalid count")
		}
		if err != nil {
			t.Fatal("invalid err")
		}
		events := trace.NetworkEvents()
		if len(events) != 0 {
			t.Fatal("expected no network events")
		}
	})
}

func TestWrapUDPLikeConn(t *testing.T) {
	t.Run("WrapUDPLikeConn wraps the conn", func(t *testing.T) {
		underlying := &mocks.UDPLikeConn{}
		zeroTime := time.Now()
		trace := NewTrace(0, zeroTime)
		conn := trace.MaybeWrapUDPLikeConn(underlying)
		ct := conn.(*udpLikeConnTrace)
		if ct.UDPLikeConn != underlying {
			t.Fatal("invalid underlying")
		}
		if ct.tx != trace {
			t.Fatal("invalid trace")
		}
	})

	t.Run("ReadFrom saves a trace", func(t *testing.T) {
		underlying := &mocks.UDPLikeConn{
			MockReadFrom: func(b []byte) (int, net.Addr, error) {
				return len(b), &mocks.Addr{
					MockString: func() string {
						return "1.1.1.1:443"
					},
					MockNetwork: func() string {
						return "udp"
					},
				}, nil
			},
		}
		zeroTime := time.Now()
		td := testingx.NewTimeDeterministic(zeroTime)
		trace := NewTrace(0, zeroTime, "antani")
		trace.timeNowFn = td.Now // deterministic time counting
		conn := trace.MaybeWrapUDPLikeConn(underlying)
		const bufsiz = 128
		buffer := make([]byte, bufsiz)
		count, addr, err := conn.ReadFrom(buffer)
		if count != bufsiz {
			t.Fatal("invalid count")
		}
		if addr.String() != "1.1.1.1:443" {
			t.Fatal("invalid address")
		}
		if err != nil {
			t.Fatal("invalid err")
		}

		t.Run("we update the trace's byte received map", func(t *testing.T) {
			stats := trace.CloneBytesReceivedMap()
			if len(stats) != 1 {
				t.Fatal("expected to see just one entry")
			}
			if stats["1.1.1.1:443 udp"] != 128 {
				t.Fatal("expected to know we received 128 bytes")
			}
		})

		events := trace.NetworkEvents()
		if len(events) != 1 {
			t.Fatal("did not save network events")
		}
		expect := &model.ArchivalNetworkEvent{
			Address:   "1.1.1.1:443",
			Failure:   nil,
			NumBytes:  bufsiz,
			Operation: "read_from",
			Proto:     "udp",
			T:         1.0,
			Tags:      []string{"antani"},
		}
		got := events[0]
		if diff := cmp.Diff(expect, got); diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("ReadFrom discards the event when the buffer is full", func(t *testing.T) {
		underlying := &mocks.UDPLikeConn{
			MockReadFrom: func(b []byte) (int, net.Addr, error) {
				return len(b), &mocks.Addr{
					MockString: func() string {
						return "1.1.1.1:443"
					},
					MockNetwork: func() string {
						return "udp"
					},
				}, nil
			},
		}
		zeroTime := time.Now()
		trace := NewTrace(0, zeroTime)
		trace.networkEvent = make(chan *model.ArchivalNetworkEvent) // no buffer
		conn := trace.MaybeWrapUDPLikeConn(underlying)
		const bufsiz = 128
		buffer := make([]byte, bufsiz)
		count, addr, err := conn.ReadFrom(buffer)
		if count != bufsiz {
			t.Fatal("invalid count")
		}
		if addr.String() != "1.1.1.1:443" {
			t.Fatal("invalid address")
		}
		if err != nil {
			t.Fatal("invalid err")
		}
		events := trace.NetworkEvents()
		if len(events) != 0 {
			t.Fatal("expected no network events")
		}
	})

	t.Run("WriteTo saves a trace", func(t *testing.T) {
		underlying := &mocks.UDPLikeConn{
			MockWriteTo: func(b []byte, addr net.Addr) (int, error) {
				return len(b), nil
			},
		}
		zeroTime := time.Now()
		td := testingx.NewTimeDeterministic(zeroTime)
		trace := NewTrace(0, zeroTime, "antani")
		trace.timeNowFn = td.Now // deterministic time tracking
		conn := trace.MaybeWrapUDPLikeConn(underlying)
		const bufsiz = 128
		buffer := make([]byte, bufsiz)
		addr := &mocks.Addr{
			MockString: func() string {
				return "1.1.1.1:443"
			},
		}
		count, err := conn.WriteTo(buffer, addr)
		if count != bufsiz {
			t.Fatal("invalid count")
		}
		if err != nil {
			t.Fatal("invalid err")
		}
		events := trace.NetworkEvents()
		if len(events) != 1 {
			t.Fatal("did not save network events")
		}
		expect := &model.ArchivalNetworkEvent{
			Address:   "1.1.1.1:443",
			Failure:   nil,
			NumBytes:  bufsiz,
			Operation: "write_to",
			Proto:     "udp",
			T:         1.0,
			Tags:      []string{"antani"},
		}
		got := events[0]
		if diff := cmp.Diff(expect, got); diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("WriteTo discards the event when the buffer is full", func(t *testing.T) {
		underlying := &mocks.UDPLikeConn{
			MockWriteTo: func(b []byte, addr net.Addr) (int, error) {
				return len(b), nil
			},
		}
		zeroTime := time.Now()
		trace := NewTrace(0, zeroTime)
		trace.networkEvent = make(chan *model.ArchivalNetworkEvent) // no buffer
		conn := trace.MaybeWrapUDPLikeConn(underlying)
		const bufsiz = 128
		buffer := make([]byte, bufsiz)
		addr := &mocks.Addr{
			MockString: func() string {
				return "1.1.1.1:443"
			},
		}
		count, err := conn.WriteTo(buffer, addr)
		if count != bufsiz {
			t.Fatal("invalid count")
		}
		if err != nil {
			t.Fatal("invalid err")
		}
		events := trace.NetworkEvents()
		if len(events) != 0 {
			t.Fatal("expected no network events")
		}
	})
}

func TestFirstNetworkEvent(t *testing.T) {
	t.Run("returns nil when buffer is empty", func(t *testing.T) {
		zeroTime := time.Now()
		trace := NewTrace(0, zeroTime)
		got := trace.FirstNetworkEventOrNil()
		if got != nil {
			t.Fatal("expected nil event")
		}
	})

	t.Run("return first non-nil network event", func(t *testing.T) {
		filler := func(tx *Trace, events []*model.ArchivalNetworkEvent) {
			for _, ev := range events {
				tx.networkEvent <- ev
			}
		}
		zeroTime := time.Now()
		trace := NewTrace(0, zeroTime)
		expect := []*model.ArchivalNetworkEvent{{
			Address:   "1.1.1.1:443",
			Failure:   nil,
			NumBytes:  0,
			Operation: "read_from",
			Proto:     "udp",
			T:         1.0,
			Tags:      []string{},
		}, {
			Address:   "1.1.1.1:443",
			Failure:   nil,
			NumBytes:  0,
			Operation: "write_to",
			Proto:     "udp",
			T:         1.0,
			Tags:      []string{},
		}}
		filler(trace, expect)
		got := trace.FirstNetworkEventOrNil()
		if diff := cmp.Diff(got, expect[0]); diff != "" {
			t.Fatal(diff)
		}
	})
}

func TestNewAnnotationArchivalNetworkEvent(t *testing.T) {
	var (
		index     int64 = 3
		duration        = 250 * time.Millisecond
		operation       = "tls_handshake_start"
	)
	expect := &model.ArchivalNetworkEvent{
		Address:       "",
		Failure:       nil,
		NumBytes:      0,
		Operation:     operation,
		Proto:         "",
		T0:            duration.Seconds(),
		T:             duration.Seconds(),
		TransactionID: index,
		Tags:          []string{"antani"},
	}
	got := NewAnnotationArchivalNetworkEvent(
		index, duration, operation, "antani",
	)
	if diff := cmp.Diff(expect, got); diff != "" {
		t.Fatal(diff)
	}
}

func TestTrace_updateBytesReceivedMapNetConn(t *testing.T) {
	t.Run("we handle tcp4, tcp6, udp4 and udp6 like they were tcp and udp", func(t *testing.T) {
		// create a new trace
		tx := NewTrace(0, time.Now())

		// insert stats for tcp, tcp4 and tcp6
		tx.updateBytesReceivedMapNetConn("tcp", "1.2.3.4:5678", 10)
		tx.updateBytesReceivedMapNetConn("tcp4", "1.2.3.4:5678", 100)
		tx.updateBytesReceivedMapNetConn("tcp", "[::1]:5678", 10)
		tx.updateBytesReceivedMapNetConn("tcp6", "[::1]:5678", 100)

		// insert stats for udp, udp4 and udp6
		tx.updateBytesReceivedMapNetConn("udp", "1.2.3.4:5678", 10)
		tx.updateBytesReceivedMapNetConn("udp4", "1.2.3.4:5678", 100)
		tx.updateBytesReceivedMapNetConn("udp", "[::1]:5678", 10)
		tx.updateBytesReceivedMapNetConn("udp6", "[::1]:5678", 100)

		// make sure the result is the expected one
		expected := map[string]int64{
			"1.2.3.4:5678 tcp": 110,
			"[::1]:5678 tcp":   110,
			"1.2.3.4:5678 udp": 110,
			"[::1]:5678 udp":   110,
		}
		got := tx.CloneBytesReceivedMap()
		if diff := cmp.Diff(expected, got); diff != "" {
			t.Fatal(diff)
		}
	})
}

func TestTrace_maybeUpdateBytesReceivedMapUDPLikeConn(t *testing.T) {
	t.Run("we ignore cases where the address is nil", func(t *testing.T) {
		// create a new trace
		tx := NewTrace(0, time.Now())

		// insert stats with a nil address
		tx.maybeUpdateBytesReceivedMapUDPLikeConn(nil, 128)

		// inserts stats with a good address
		goodAddr := &mocks.Addr{
			MockString: func() string {
				return "1.2.3.4:5678"
			},
			MockNetwork: func() string {
				return "udp"
			},
		}
		tx.maybeUpdateBytesReceivedMapUDPLikeConn(goodAddr, 128)

		// make sure the result is the expected one
		expected := map[string]int64{
			"1.2.3.4:5678 udp": 128,
		}
		got := tx.CloneBytesReceivedMap()
		if diff := cmp.Diff(expected, got); diff != "" {
			t.Fatal(diff)
		}
	})
}
