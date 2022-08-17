package measurexlite

import (
	"net"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/testingx"
)

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

func TestWrapNetConn(t *testing.T) {
	t.Run("WrapNetConn wraps the conn", func(t *testing.T) {
		underlying := &mocks.Conn{}
		zeroTime := time.Now()
		trace := NewTrace(0, zeroTime)
		conn := trace.WrapNetConn(underlying)
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
		trace := NewTrace(0, zeroTime)
		trace.TimeNowFn = td.Now // deterministic time counting
		conn := trace.WrapNetConn(underlying)
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
			Tags:      []string{},
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
		trace.NetworkEvent = make(chan *model.ArchivalNetworkEvent) // no buffer
		conn := trace.WrapNetConn(underlying)
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
		trace := NewTrace(0, zeroTime)
		trace.TimeNowFn = td.Now // deterministic time tracking
		conn := trace.WrapNetConn(underlying)
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
			Tags:      []string{},
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
		trace.NetworkEvent = make(chan *model.ArchivalNetworkEvent) // no buffer
		conn := trace.WrapNetConn(underlying)
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
		conn := trace.WrapUDPLikeConn(underlying)
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
				}, nil
			},
		}
		zeroTime := time.Now()
		td := testingx.NewTimeDeterministic(zeroTime)
		trace := NewTrace(0, zeroTime)
		trace.TimeNowFn = td.Now // deterministic time counting
		conn := trace.WrapUDPLikeConn(underlying)
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
			Tags:      []string{},
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
				}, nil
			},
		}
		zeroTime := time.Now()
		trace := NewTrace(0, zeroTime)
		trace.NetworkEvent = make(chan *model.ArchivalNetworkEvent) // no buffer
		conn := trace.WrapUDPLikeConn(underlying)
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
		trace := NewTrace(0, zeroTime)
		trace.TimeNowFn = td.Now // deterministic time tracking
		conn := trace.WrapUDPLikeConn(underlying)
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
			Tags:      []string{},
		}
		got := events[0]
		if diff := cmp.Diff(expect, got); diff != "" {
			t.Fatal(diff)
		}
	})

	t.Run("Write discards the event when the buffer is full", func(t *testing.T) {
		underlying := &mocks.UDPLikeConn{
			MockWriteTo: func(b []byte, addr net.Addr) (int, error) {
				return len(b), nil
			},
		}
		zeroTime := time.Now()
		trace := NewTrace(0, zeroTime)
		trace.NetworkEvent = make(chan *model.ArchivalNetworkEvent) // no buffer
		conn := trace.WrapUDPLikeConn(underlying)
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

func TestNewAnnotationArchivalNetworkEvent(t *testing.T) {
	var (
		index     int64 = 3
		duration        = 250 * time.Millisecond
		operation       = "tls_handshake_start"
	)
	expect := &model.ArchivalNetworkEvent{
		Address:   "",
		Failure:   nil,
		NumBytes:  0,
		Operation: operation,
		Proto:     "",
		T:         duration.Seconds(),
		Tags:      []string{},
	}
	got := NewAnnotationArchivalNetworkEvent(
		index, duration, operation,
	)
	if diff := cmp.Diff(expect, got); diff != "" {
		t.Fatal(diff)
	}
}
