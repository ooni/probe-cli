//go:build aix || darwin || dragonfly || freebsd || (js && wasm) || linux || nacl || netbsd || openbsd || solaris

package tlsmiddlebox

import (
	"encoding/hex"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
	"golang.org/x/sys/unix"
)

type mockUnixOpsImpl struct {
	SocketFunc        func(domain, typ, proto int) (int, error)
	CloseFunc         func(fd int) error
	SetsockoptIntFunc func(fd, level, opt, value int) error
	SetNonblockFunc   func(fd int, nonblocking bool) error
	ConnectFunc       func(fd int, sa unix.Sockaddr) error
	PollFunc          func(fds []unix.PollFd, timeout int) (int, error)
	RecvmsgFunc       func(fd int, p, oob []byte, flags int) (int, int, int, unix.Sockaddr, error)
	GetsockoptIntFunc func(fd, level, opt int) (int, error)
}

func (m *mockUnixOpsImpl) Socket(domain, typ, proto int) (int, error) {
	return m.SocketFunc(domain, typ, proto)
}

func (m *mockUnixOpsImpl) Close(fd int) error {
	return m.CloseFunc(fd)
}

func (m *mockUnixOpsImpl) SetsockoptInt(fd, level, opt, value int) error {
	return m.SetsockoptIntFunc(fd, level, opt, value)
}

func (m *mockUnixOpsImpl) SetNonblock(fd int, nonblocking bool) error {
	return m.SetNonblockFunc(fd, nonblocking)
}

func (m *mockUnixOpsImpl) Connect(fd int, sa unix.Sockaddr) error {
	return m.ConnectFunc(fd, sa)
}

func (m *mockUnixOpsImpl) Poll(fds []unix.PollFd, timeout int) (int, error) {
	return m.PollFunc(fds, timeout)
}

func (m *mockUnixOpsImpl) Recvmsg(fd int, p, oob []byte, flags int) (int, int, int, unix.Sockaddr, error) {
	return m.RecvmsgFunc(fd, p, oob, flags)
}

func (m *mockUnixOpsImpl) GetsockoptInt(fd, level, opt int) (int, error) {
	return m.GetsockoptIntFunc(fd, level, opt)
}

func defineMockUnixOpsImpl() *mockUnixOpsImpl {
	return &mockUnixOpsImpl{
		SocketFunc: func(domain, typ, proto int) (int, error) {
			return 1, nil
		},

		CloseFunc: func(fd int) error {
			return nil
		},

		SetsockoptIntFunc: func(fd, level, opt, value int) error {
			return nil
		},

		SetNonblockFunc: func(fd int, nonblocking bool) error {
			return nil
		},

		ConnectFunc: func(fd int, sa unix.Sockaddr) error {
			return nil
		},

		PollFunc: func(fds []unix.PollFd, timeout int) (int, error) {
			return 0, nil
		},

		RecvmsgFunc: func(fd int, p, oob []byte, flags int) (int, int, int, unix.Sockaddr, error) {
			return 0, 0, 0, nil, nil
		},
		GetsockoptIntFunc: func(fd, level, opt int) (int, error) {
			return 0, nil
		},
	}
}

func TestParseQuotedPacket(t *testing.T) {
	t.Run("buffer too short", func(t *testing.T) {
		buf := []byte("Hello")
		quotedPacket, err := parseQuotedPacket(buf)
		if quotedPacket == nil && (err == nil || err.Error() != "tcp quote too short") {
			t.Fatal("failed to error due to quote being too short")
		}
	})

	t.Run("success case", func(t *testing.T) {
		buf := []byte{0xed, 0x6e, 0x01, 0xbb, 0x16, 0xed, 0xf5, 0x4c}
		quotedPacket, err := parseQuotedPacket(buf)

		if err != nil {
			t.Fatal("unexpected error:", err)
		}

		if quotedPacket == nil {
			t.Fatal("expected *model.ArchivalICMPQuotation, got nil")
		}

		if quotedPacket.Protocol != 6 {
			t.Fatalf("expected protocol 6, got %d", quotedPacket.Protocol)
		}

		if quotedPacket.SrcPort != 60782 {
			t.Fatalf("expected source port 60782, got %d", quotedPacket.SrcPort)
		}

		if quotedPacket.DstPort != 443 {
			t.Fatalf("expected destination port 443, got %d", quotedPacket.DstPort)
		}

		if quotedPacket.TCPSeqNum != 384693580 {
			t.Fatalf("expected TCP seq num 384693580, got %d", quotedPacket.TCPSeqNum)
		}
	})
}

func TestTracerouteTCP(t *testing.T) {
	wg := new(sync.WaitGroup)
	ttl := 2
	index := int64(1)

	t.Run("invalid address and port format", func(t *testing.T) {
		wg.Add(1)
		address := "1.2.3.4"
		zeroTime := time.Now()
		unixOpsImpl := defineMockUnixOpsImpl()
		ii, err := tracerouteTCPWithOps(index, zeroTime, address, ttl, 3000, wg, model.DiscardLogger, "safe", unixOpsImpl)
		if ii != nil {
			t.Fatalf("expected nil, got %T", ii)
		}

		if err.Error() != "address 1.2.3.4: missing port in address" {
			t.Fatalf("expected 'address 1.2.3.4: missing port in address', got %s", err.Error())
		}
	})

	t.Run("invalid IPv4 address", func(t *testing.T) {
		wg.Add(1)
		address := "298.125.34.4:443"
		zeroTime := time.Now()
		unixOpsImpl := defineMockUnixOpsImpl()
		ii, err := tracerouteTCPWithOps(index, zeroTime, address, ttl, 3000, wg, model.DiscardLogger, "safe", unixOpsImpl)
		if ii != nil {
			t.Fatalf("expected nil, got %T", ii)
		}

		if err.Error() != "invalid IPv4 address" {
			t.Fatalf("expected 'invalid IPv4 address', got %s", err.Error())
		}

	})

	t.Run("invalid port number", func(t *testing.T) {
		wg.Add(1)
		address := "298.125.34.4:spot"
		zeroTime := time.Now()
		unixOpsImpl := defineMockUnixOpsImpl()
		ii, err := tracerouteTCPWithOps(index, zeroTime, address, ttl, 3000, wg, model.DiscardLogger, "safe", unixOpsImpl)
		if ii != nil {
			t.Fatalf("expected nil, got %T", ii)
		}

		if err.Error() != "strconv.Atoi: parsing \"spot\": invalid syntax" {
			t.Fatalf("expected 'strconv.Atoi: parsing \"spot\": invalid syntax', got %s", err.Error())
		}
	})

	t.Run("timeout", func(t *testing.T) {
		wg.Add(1)
		address := "127.0.0.1:1"
		zeroTime := time.Now()
		unixOpsImpl := defineMockUnixOpsImpl()

		unixOpsImpl.PollFunc = func(fds []unix.PollFd, timeout int) (int, error) {
			fds[0].Revents = unix.POLLOUT
			return 0, nil
		}

		ii, err := tracerouteTCPWithOps(index, zeroTime, address, ttl, 3000, wg, model.DiscardLogger, "safe", unixOpsImpl)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if ii == nil {
			t.Fatal("expected ICMP Iteration, got nil")
		}

		if ii.ICMPError == nil {
			t.Fatal("expected ArchivalICMPErrorMessage")
		}

		if ii.TTL != ttl {
			t.Fatalf("expected TTL %d, got %d", ttl, ii.TTL)
		}

		if ii.ICMPError.Timeout != "yes" {
			t.Fatalf("expected Timeout field in ArchivalICMPErrorMessage to be 'yes', got %s", ii.ICMPError.Timeout)
		}
	})

	t.Run("read from Recvmsg with non-EAGAIN error", func(t *testing.T) {
		wg.Add(1)
		address := "127.0.0.1:1"
		zeroTime := time.Now()
		unixOpsImpl := defineMockUnixOpsImpl()
		expectedErr := errors.New("recvmsg failed")

		unixOpsImpl.PollFunc = func(fds []unix.PollFd, timeout int) (int, error) {
			fds[0].Revents = unix.POLLERR
			return 1, nil
		}

		unixOpsImpl.RecvmsgFunc = func(fd int, p, oob []byte, flags int) (int, int, int, unix.Sockaddr, error) {
			return 0, 0, 0, nil, expectedErr
		}

		ii, err := tracerouteTCPWithOps(index, zeroTime, address, ttl, 3000, wg, model.DiscardLogger, "safe", unixOpsImpl)

		if !errors.Is(err, expectedErr) {
			t.Fatalf("expected error: %v, got %v", expectedErr, err)
		}

		if ii != nil {
			t.Fatalf("expected nil, got %T", ii)
		}

	})

	t.Run("read from Recvmsg with safe mode", func(t *testing.T) {
		wg.Add(1)
		address := "127.0.0.1:1"
		zeroTime := time.Now()
		unixOpsImpl := defineMockUnixOpsImpl()

		unixOpsImpl.PollFunc = func(fds []unix.PollFd, timeout int) (int, error) {
			fds[0].Revents = unix.POLLERR
			return 1, nil
		}

		bufHex := "8c2201bbd45f1b4b00000000a002faf04ed90000020405b40402080a89f6b44300000000010303070000000000000000"
		oobHex := "400000000000000001000000250000004b32a96a00000000bbe2f6290000000000000000000000000000000000000000000000000000000000000000000000003000000000000000000000000b00000071000000020b00000000000000000000020000000ac802d10000000000000000"

		buf, err := hex.DecodeString(bufHex)
		if err != nil {
			t.Fatal(err)
		}
		oob, err := hex.DecodeString(oobHex)
		if err != nil {
			t.Fatal(err)
		}

		callRecvmsgCount := 0

		unixOpsImpl.RecvmsgFunc = func(fd int, p, oobBuf []byte, flags int) (int, int, int, unix.Sockaddr, error) {
			callRecvmsgCount++

			if callRecvmsgCount == 1 {
				copy(p, buf)
				copy(oobBuf, oob)
				return len(buf), len(oob), 0, nil, nil
			}

			return 0, 0, 0, nil, unix.EAGAIN

		}

		ii, err := tracerouteTCPWithOps(index, zeroTime, address, ttl, 3000, wg, model.DiscardLogger, "safe", unixOpsImpl)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if ii == nil {
			t.Fatal("expected ICMP Iteration, got nil")
		}

		if ii.ICMPError == nil {
			t.Fatal("expected ArchivalICMPErrorMessage")
		}

		if ii.TTL != ttl {
			t.Fatalf("expected TTL %d, got %d", ttl, ii.TTL)
		}

		if ii.ICMPError.Timeout != "no" {
			t.Fatalf("expected Timeout field in ArchivalICMPErrorMessage to be 'no', got %s", ii.ICMPError.Timeout)
		}

		if ii.ICMPError.SrcIPPrefix != "" {
			t.Fatalf("expected SrcIPPrefix field in ArchivalICMPErrorMessage to be '', got %s", ii.ICMPError.SrcIPPrefix)
		}

		if ii.ICMPError.SrcIPCountryCode != "ZZ" {
			t.Fatalf("expected SrcIPCountryCode field in ArchivalICMPErrorMessage to be 'ZZ', got %s", ii.ICMPError.SrcIPCountryCode)
		}

		if ii.ICMPError.SrcIPASN != 0 {
			t.Fatalf("expected SrcIPASN field in ArchivalICMPErrorMessage to be '0', got %d", ii.ICMPError.SrcIPASN)
		}

		if ii.ICMPError.SrcIPASNOrg != "" {
			t.Fatalf("expected SrcIPASNOrg field in ArchivalICMPErrorMessage to be '', got %s", ii.ICMPError.SrcIPASNOrg)
		}

		if ii.ICMPError.Type != 11 {
			t.Fatalf("expected Type field in ArchivalICMPErrorMessage to be 11, got %d", ii.ICMPError.Type)
		}

		if ii.ICMPError.Code != 0 {
			t.Fatalf("expected Code field in ArchivalICMPErrorMessage to be 0, got %d", ii.ICMPError.Code)
		}

		if ii.ICMPError.T0 == 0 {
			t.Fatalf("expected T0 field in ArchivalICMPErrorMessage to be nonzero, got %f", ii.ICMPError.T0)
		}

		if ii.ICMPError.T != 0 {
			t.Fatalf("expected T field in ArchivalICMPErrorMessage to be zero, got %f", ii.ICMPError.T)
		}

		if ii.ICMPError.Quote.Protocol != 0 {
			t.Fatalf("expected Protocol field in Quote to be 0, got %d", ii.ICMPError.Quote.Protocol)
		}

		if ii.ICMPError.Quote.SrcPort != 0 {
			t.Fatalf("expected SrcPort field in Quote to be 0, got %d", ii.ICMPError.Quote.SrcPort)
		}

		if ii.ICMPError.Quote.DstPort != 0 {
			t.Fatalf("expected DstPort field in Quote to be 0, got %d", ii.ICMPError.Quote.DstPort)
		}

		if ii.ICMPError.Quote.TCPSeqNum != 0 {
			t.Fatalf("expected TCPSeqNum field in Quote to be 0, got %d", ii.ICMPError.Quote.TCPSeqNum)
		}

		if len(ii.ICMPError.Quote.RemainingPayload) != 0 {
			t.Fatalf("expected length of the RemainingPayload field in Quote to be 0, got %d", len(ii.ICMPError.Quote.RemainingPayload))
		}

	})

	t.Run("read from Recvmsg with unsafe mode", func(t *testing.T) {
		wg.Add(1)
		address := "127.0.0.1:1"
		zeroTime := time.Date(2026, 9, 15, 11, 55, 54, 0, time.UTC)
		unixOpsImpl := defineMockUnixOpsImpl()

		unixOpsImpl.PollFunc = func(fds []unix.PollFd, timeout int) (int, error) {
			fds[0].Revents = unix.POLLERR
			return 1, nil
		}

		bufHex := "8c2201bbd45f1b4b00000000a002faf04ed90000020405b40402080a89f6b44300000000010303070000000000000000"
		oobHex := "400000000000000001000000250000004b32a96a00000000bbe2f6290000000000000000000000000000000000000000000000000000000000000000000000003000000000000000000000000b00000071000000020b00000000000000000000020000000ac802d10000000000000000"

		buf, err := hex.DecodeString(bufHex)
		if err != nil {
			t.Fatal(err)
		}
		oob, err := hex.DecodeString(oobHex)
		if err != nil {
			t.Fatal(err)
		}

		callRecvmsgCount := 0

		unixOpsImpl.RecvmsgFunc = func(fd int, p, oobBuf []byte, flags int) (int, int, int, unix.Sockaddr, error) {
			callRecvmsgCount++

			if callRecvmsgCount == 1 {
				copy(p, buf)
				copy(oobBuf, oob)
				return len(buf), len(oob), 0, nil, nil
			}

			return 0, 0, 0, nil, unix.EAGAIN

		}

		ii, err := tracerouteTCPWithOps(index, zeroTime, address, ttl, 3000, wg, model.DiscardLogger, "unsafe", unixOpsImpl)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if ii == nil {
			t.Fatal("expected ICMP Iteration, got nil")
		}

		if ii.ICMPError == nil {
			t.Fatal("expected ArchivalICMPErrorMessage")
		}

		if ii.TTL != ttl {
			t.Fatalf("expected TTL %d, got %d", ttl, ii.TTL)
		}

		if ii.ICMPError.Timeout != "no" {
			t.Fatalf("expected Timeout field in ArchivalICMPErrorMessage to be 'no', got %s", ii.ICMPError.Timeout)
		}

		if ii.ICMPError.SrcIPPrefix != "10.200.2.0/24" {
			t.Fatalf("expected SrcIPPrefix field in ArchivalICMPErrorMessage to be '10.200.2.0/24', got %s", ii.ICMPError.SrcIPPrefix)
		}

		if ii.ICMPError.SrcIPCountryCode != "ZZ" {
			t.Fatalf("expected SrcIPCountryCode field in ArchivalICMPErrorMessage to be 'ZZ', got %s", ii.ICMPError.SrcIPCountryCode)
		}

		if ii.ICMPError.SrcIPASN != 0 {
			t.Fatalf("expected SrcIPASN field in ArchivalICMPErrorMessage to be '0', got %d", ii.ICMPError.SrcIPASN)
		}

		if ii.ICMPError.SrcIPASNOrg != "" {
			t.Fatalf("expected SrcIPASNOrg field in ArchivalICMPErrorMessage to be '', got %s", ii.ICMPError.SrcIPASNOrg)
		}

		if ii.ICMPError.Type != 11 {
			t.Fatalf("expected Type field in ArchivalICMPErrorMessage to be 11, got %d", ii.ICMPError.Type)
		}

		if ii.ICMPError.Code != 0 {
			t.Fatalf("expected Code field in ArchivalICMPErrorMessage to be 0, got %d", ii.ICMPError.Code)
		}

		if ii.ICMPError.T0 == 0 {
			t.Fatalf("expected T0 field in ArchivalICMPErrorMessage to be nonzero, got %f", ii.ICMPError.T0)
		}

		if ii.ICMPError.T == 0 {
			t.Fatalf("expected T field in ArchivalICMPErrorMessage to be nonzero, got %f", ii.ICMPError.T)
		}

		if ii.ICMPError.Quote.Protocol != 6 {
			t.Fatalf("expected Protocol field in Quote to be 6, got %d", ii.ICMPError.Quote.Protocol)
		}

		if ii.ICMPError.Quote.SrcPort != 35874 {
			t.Fatalf("expected SrcPort field in Quote to be 35874, got %d", ii.ICMPError.Quote.SrcPort)
		}

		if ii.ICMPError.Quote.DstPort != 443 {
			t.Fatalf("expected DstPort field in Quote to be 443, got %d", ii.ICMPError.Quote.DstPort)
		}

		if ii.ICMPError.Quote.TCPSeqNum != 3563002699 {
			t.Fatalf("expected TCPSeqNum field in Quote to be 3563002699, got %d", ii.ICMPError.Quote.TCPSeqNum)
		}

		if len(ii.ICMPError.Quote.RemainingPayload) <= 0 {
			t.Fatalf("expected length of the RemainingPayload field in Quote to be greater than 0, got %d", len(ii.ICMPError.Quote.RemainingPayload))
		}

	})

	t.Run("connected", func(t *testing.T) {
		wg.Add(1)
		address := "127.0.0.1:1"
		zeroTime := time.Now()
		unixOpsImpl := defineMockUnixOpsImpl()

		unixOpsImpl.PollFunc = func(fds []unix.PollFd, timeout int) (int, error) {
			fds[0].Revents = unix.POLLOUT
			return 1, nil
		}

		ii, err := tracerouteTCPWithOps(index, zeroTime, address, ttl, 3000, wg, model.DiscardLogger, "safe", unixOpsImpl)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if ii == nil {
			t.Fatal("expected ICMP Iteration, got nil")
		}

		if ii.ICMPError == nil {
			t.Fatal("expected ArchivalICMPErrorMessage")
		}

		if ii.TTL != ttl {
			t.Fatalf("expected TTL %d, got %d", ttl, ii.TTL)
		}

		if ii.ICMPError.Connected != "yes" {
			t.Fatalf("expected Connected field in ArchivalICMPErrorMessage to be 'yes', got %s", ii.ICMPError.Connected)
		}

	})

	t.Run("soerror is not zero", func(t *testing.T) {
		wg.Add(1)
		address := "127.0.0.1:1"
		zeroTime := time.Now()
		unixOpsImpl := defineMockUnixOpsImpl()

		unixOpsImpl.PollFunc = func(fds []unix.PollFd, timeout int) (int, error) {
			fds[0].Revents = unix.POLLOUT
			return 1, nil
		}

		unixOpsImpl.GetsockoptIntFunc = func(fd, level, opt int) (int, error) {
			return int(unix.ECONNREFUSED), nil
		}

		ii, err := tracerouteTCPWithOps(index, zeroTime, address, ttl, 3000, wg, model.DiscardLogger, "safe", unixOpsImpl)

		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		if ii == nil {
			t.Fatal("expected ICMP Iteration, got nil")
		}

		if ii.ICMPError == nil {
			t.Fatal("expected ArchivalICMPErrorMessage")
		}

		if ii.TTL != ttl {
			t.Fatalf("expected TTL %d, got %d", ttl, ii.TTL)
		}

		if ii.ICMPError.Error == "" {
			t.Fatalf("expected Error field in ArchivalICMPErrorMessage to be populated")
		}
	})

}
