package tlsmiddlebox

import (
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"sync"
	"time"
	"unsafe"

	"github.com/ooni/probe-cli/v3/internal/logx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"golang.org/x/sys/unix"
)

func parseQuotedPacket(buf []byte) (*model.ArchivalICMPQuotation, error) {
	if len(buf) < 8 {
		return nil, fmt.Errorf("tcp quote too short")
	}

	quotedPacket := &model.ArchivalICMPQuotation{
		Protocol:  6,
		SrcPort:   int(binary.BigEndian.Uint16(buf[0:2])),
		DstPort:   int(binary.BigEndian.Uint16(buf[2:4])),
		TCPSeqNum: binary.BigEndian.Uint32(buf[4:8]),
	}

	return quotedPacket, nil
}

func probeTCP(address string, ttl int, timeoutMS int, wg *sync.WaitGroup, logger model.Logger, index int64) (*ICMPIteration, error) {
	defer wg.Done()
	host, portString, err := net.SplitHostPort(address)

	port, err := strconv.Atoi(portString)

	fd, err := unix.Socket(unix.AF_INET, unix.SOCK_STREAM, unix.IPPROTO_TCP)

	if err != nil {
		return nil, err
	}

	defer unix.Close(fd)

	if err := unix.SetsockoptInt(fd, unix.IPPROTO_IP, unix.IP_RECVERR, 1); err != nil {
		return nil, err
	}

	if err := unix.SetNonblock(fd, true); err != nil {
		return nil, err
	}

	if err := unix.SetsockoptInt(fd, unix.IPPROTO_IP, unix.IP_TTL, ttl); err != nil {
		return nil, err
	}

	timestampFlags := unix.SOF_TIMESTAMPING_TX_SOFTWARE |
		unix.SOF_TIMESTAMPING_RX_SOFTWARE |
		unix.SOF_TIMESTAMPING_SOFTWARE

	if err := unix.SetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_TIMESTAMPING, timestampFlags); err != nil {
		return nil, err
	}

	ip := net.ParseIP(host).To4()
	if ip == nil {
		return nil, fmt.Errorf("invalid IPv4 address")
	}

	var addr [4]byte
	copy(addr[:], ip)

	sa := &unix.SockaddrInet4{
		Port: port,
		Addr: addr,
	}

	var txTime *time.Time

	txTimeVal := time.Now()
	txTime = &txTimeVal

	err = unix.Connect(fd, sa)
	if err != nil && err != unix.EINPROGRESS {
		return nil, err
	}

	pfds := []unix.PollFd{
		{
			Fd:     int32(fd),
			Events: unix.POLLOUT | unix.POLLERR,
		},
	}

	n, err := unix.Poll(pfds, timeoutMS)
	if err != nil {
		return nil, err
	}

	if n == 0 {
		ol := logx.NewOperationLogger(logger, "Traceroute #%d TTL %d %s TIMEOUT", index, ttl, address)
		ol.Stop(nil)
		ii_timeout := &ICMPIteration{
			TTL: ttl,
			ICMPError: &model.ArchivalICMPErrorMessage{
				Timeout: "yes",
			},
		}
		return ii_timeout, nil
	}

	if pfds[0].Revents&unix.POLLERR != 0 {

		buf := make([]byte, 64)
		oob := make([]byte, 512)

		var data []byte
		var rxTime *time.Time
		var eeType byte
		var eeCode byte
		var ip net.IP
		var quotedPacket *model.ArchivalICMPQuotation

		for {
			n, oobn, _, _, err := unix.Recvmsg(fd, buf, oob, unix.MSG_ERRQUEUE)

			if err == unix.EAGAIN {
				break
			}

			if err != nil {
				return nil, err
			}

			quoted := buf[:n]

			quotedPacket, err = parseQuotedPacket(quoted)

			if err != nil {
				fmt.Println("quote parse failed:", err)
				return nil, err
			}

			cms, err := unix.ParseSocketControlMessage(oob[:oobn])
			if err != nil {
				return nil, err
			}

			for _, cm := range cms {

				switch {
				case cm.Header.Level == unix.SOL_SOCKET &&
					cm.Header.Type == unix.SO_TIMESTAMPING:
					var ts [3]unix.Timespec
					if len(cm.Data) >= int(unsafe.Sizeof([3]unix.Timespec{})) {
						ts = *(*[3]unix.Timespec)(unsafe.Pointer(&cm.Data[0]))
					}

					if ts[0].Sec != 0 {
						t := time.Unix(ts[0].Sec, ts[0].Nsec)
						rxTime = &t
					}

				case cm.Header.Level == unix.IPPROTO_IP &&
					cm.Header.Type == unix.IP_RECVERR:
					data = cm.Data

					if len(data) < 16 {
						continue
					}

					eeType = data[5]
					eeCode = data[6]

					if len(data) >= 24 {
						family := binary.LittleEndian.Uint16(data[16:18])

						if family == unix.AF_INET {
							ip = net.IP(data[20:24])
						}
					}

				}
			}

		}

		var t0, t float64

		if txTime != nil {
			t0 = float64(txTime.UnixMilli())
		}

		if rxTime != nil {
			t = float64(rxTime.UnixMilli())
		}

		ii := &ICMPIteration{
			TTL: ttl,
			ICMPError: &model.ArchivalICMPErrorMessage{
				Timeout: "no",
				SrcIP:   ip.String(),
				Type:    int(eeType),
				Code:    int(eeCode),
				T0:      t0,
				T:       t,
			},
		}
		if quotedPacket != nil && err == nil {
			ii.ICMPError.Quote = *quotedPacket
		}

		ol := logx.NewOperationLogger(logger, "Traceroute #%d TTL %d %s Router %s", index, ttl, address, ip.String())
		ol.Stop(err)
		return ii, nil
	}

	if pfds[0].Revents&unix.POLLOUT != 0 {
		soerr, err := unix.GetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_ERROR)
		if err != nil {
			return nil, err
		}

		if soerr == 0 {
			ol := logx.NewOperationLogger(logger, "Traceroute #%d TTL %d %s CONNECTED", index, ttl, address)
			ol.Stop(nil)
			ii_connected := &ICMPIteration{
				TTL: ttl,
				ICMPError: &model.ArchivalICMPErrorMessage{
					Connected: "yes",
				},
			}
			return ii_connected, nil
		} else {
			ol := logx.NewOperationLogger(logger, "Traceroute #%d TTL %d %s SO_ERROR %v", index, ttl, address, unix.Errno(soerr))
			ol.Stop(nil)
			ii_error := &ICMPIteration{
				TTL: ttl,
				ICMPError: &model.ArchivalICMPErrorMessage{
					Error: unix.Errno(soerr).Error(),
				},
			}
			return ii_error, nil
		}
	}

	return nil, nil
}
