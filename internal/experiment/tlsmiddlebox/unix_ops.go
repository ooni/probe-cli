//go:build aix || darwin || dragonfly || freebsd || (js && wasm) || linux || nacl || netbsd || openbsd || solaris

package tlsmiddlebox

import "golang.org/x/sys/unix"

type unixOps interface {
	Socket(domain, typ, proto int) (int, error)
	Close(fd int) error
	SetsockoptInt(fd, level, opt, value int) error
	SetNonblock(fd int, nonblocking bool) error
	Connect(fd int, sa unix.Sockaddr) error
	Poll(fds []unix.PollFd, timeout int) (int, error)
	Recvmsg(fd int, p, oob []byte, flags int) (int, int, int, unix.Sockaddr, error)
	GetsockoptInt(fd, level, opt int) (int, error)
}

type unixOpsImpl struct{}

func (unixOpsImpl) Socket(domain, typ, proto int) (int, error) {
	return unix.Socket(domain, typ, proto)
}

func (unixOpsImpl) Close(fd int) error {
	return unix.Close(fd)
}

func (unixOpsImpl) SetsockoptInt(fd, level, opt, value int) error {
	return unix.SetsockoptInt(fd, level, opt, value)
}

func (unixOpsImpl) SetNonblock(fd int, nonblocking bool) error {
	return unix.SetNonblock(fd, nonblocking)
}

func (unixOpsImpl) Connect(fd int, sa unix.Sockaddr) error {
	return unix.Connect(fd, sa)
}

func (unixOpsImpl) Poll(fds []unix.PollFd, timeout int) (int, error) {
	return unix.Poll(fds, timeout)
}

func (unixOpsImpl) Recvmsg(fd int, p, oob []byte, flags int) (int, int, int, unix.Sockaddr, error) {
	return unix.Recvmsg(fd, p, oob, flags)
}

func (unixOpsImpl) GetsockoptInt(fd, level, opt int) (int, error) {
	return unix.GetsockoptInt(fd, level, opt)
}
