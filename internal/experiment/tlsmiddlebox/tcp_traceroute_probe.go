package tlsmiddlebox

import (
	"encoding/binary"
	"fmt"
	"net"
	"strconv"
	"sync"

	"github.com/ooni/probe-cli/v3/internal/logx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"golang.org/x/sys/unix"
)

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
		return nil, nil
	}

	//fmt.Printf("revents = %#x\n", pfds[0].Revents)

	if pfds[0].Revents&unix.POLLERR != 0 {

		buf := make([]byte, 64)
		oob := make([]byte, 512)

		_, oobn, _, _, err := unix.Recvmsg(fd, buf, oob, unix.MSG_ERRQUEUE)
		//fmt.Printf("Recvmsg: n=%d oob=%d flags=%#x err=%v\n", n, oobn, flags, err)

		if err == nil {
			cms, err := unix.ParseSocketControlMessage(oob[:oobn])
			if err != nil {
				return nil, err
			}

			//fmt.Printf("received %d control messages\n", len(cms))

			for _, cm := range cms {
				if cm.Header.Level != unix.IPPROTO_IP ||
					cm.Header.Type != unix.IP_RECVERR {
					continue
				}

				data := cm.Data

				if len(data) < 16 {
					continue
				}

				// struct sock_extended_err
				// ee_errno := binary.LittleEndian.Uint32(data[0:4])
				// ee_origin := data[4]
				ee_type := data[5]
				ee_code := data[6]

				// fmt.Printf("errno  = %d\n", ee_errno)
				// fmt.Printf("origin = %d\n", ee_origin)
				// fmt.Printf("type   = %d\n", ee_type)
				// fmt.Printf("code   = %d\n", ee_code)

				// offender sockaddr_in follows sock_extended_err
				if len(data) >= 24 {
					family := binary.LittleEndian.Uint16(data[16:18])

					if family == unix.AF_INET {
						ip := net.IP(data[20:24])
						ii := &ICMPIteration{
							TTL: ttl,
							ICMPError: &model.ArchivalICMPErrorMessage{
								Timeout: "no",
								SrcIP:   ip.String(),
								Type:    int(ee_type),
								Code:    int(ee_code),
							},
						}
						ol := logx.NewOperationLogger(logger, "Traceroute #%d TTL %d %s Router %s", index, ttl, address, ip.String())
						ol.Stop(err)
						// fmt.Printf("offender = %s\n", ip.String())
						return ii, nil
					}
				}
			}
		}

		return nil, nil

	}

	if pfds[0].Revents&unix.POLLOUT != 0 {
		soerr, err := unix.GetsockoptInt(fd, unix.SOL_SOCKET, unix.SO_ERROR)
		if err != nil {
			return nil, err
		}

		if soerr == 0 {
			ol := logx.NewOperationLogger(logger, "Traceroute #%d TTL %d %s CONNECTED", index, ttl, address)
			ol.Stop(nil)
		} else {
			ol := logx.NewOperationLogger(logger, "Traceroute #%d TTL %d %s SO_ERROR %v", index, ttl, address, unix.Errno(soerr))
			ol.Stop(nil)
		}
	}

	return nil, nil
}
